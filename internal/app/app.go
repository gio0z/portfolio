package app

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"portfolio/internal/adminapi"
	"portfolio/internal/adminauth"
	"portfolio/internal/deployment"
	"portfolio/internal/mcppublisher"
	"portfolio/internal/preview"
	"portfolio/internal/publicapi"
	"portfolio/internal/publishing"
	"portfolio/pkg/api"
)

// App is the fully composed application. Task 17 consumes this surface for
// end-to-end tests: every subsystem is reachable as a typed field, and
// ServeHTTP exposes the complete HTTP surface (static SPA + API + auth + MCP).
type App struct {
	Config Config

	DB         *sql.DB
	Repo       publishing.Repository
	Store      publishing.ArtifactStore
	Publishing publishing.PublishingService

	// Auth is nil when admin credentials are absent outside production
	// (dev/test without OAuth configuration). Admin routes then fail
	// closed with 401 via the nil-auth stubs in adminapi.
	Auth    *adminauth.Service
	AuthMux *http.ServeMux

	AdminAPI  *adminapi.Handler
	PublicAPI *publicapi.Handler
	API       *api.Server

	// MCP is nil when MCP_TOKEN_SECRET is absent outside production.
	MCP        *mcppublisher.Server
	MCPAuth    *mcppublisher.Authenticator
	MCPAdapter *mcppublisher.Adapter

	Runner          preview.PreviewRunner
	PreviewDeployer deployment.PreviewDeployer
	Publisher       deployment.AtomicPublisher

	handler http.Handler
}

// ServeHTTP serves the full application surface: MCP endpoint, admin auth,
// API routes, and the static SPA fallback.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.handler.ServeHTTP(w, r)
}

// Close releases application resources (currently the registry database).
func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}

// New composes every subsystem from cfg and returns the ready application.
//
// Production (Env anything other than development/test) fails closed: a
// missing signing key, OAuth credential, owner allowlist entry, approval
// verifier, public origin, registry path, artifact root, or MCP token secret
// aborts startup with an error naming the env var, never its value.
//
// Outside production, unset RegistryPath selects :memory: and an unset
// ArtifactStoreRoot selects the OS temp dir; absent admin credentials or MCP
// secret leave Auth/AuthMux and MCP nil instead of failing.
//
// Ruling: New — not main — owns os.MkdirAll for the artifact root, so the
// first boot on a fresh volume succeeds without manual mkdir — cost if wrong:
// a typo'd root silently creates a stray directory (mitigated: the resolved
// root is logged, redaction-safe).
func New(cfg Config) (*App, error) {
	cfg.Env = normalizeEnv(cfg.Env)
	if strings.TrimSpace(cfg.Port) == "" {
		cfg.Port = "8080"
	}
	if strings.TrimSpace(cfg.FrontendDist) == "" {
		cfg.FrontendDist = "./frontend/dist"
	}

	production := cfg.IsProduction()
	if production {
		if err := cfg.validateProduction(); err != nil {
			return nil, err
		}
	}

	registryPath := strings.TrimSpace(cfg.RegistryPath)
	artifactRoot := strings.TrimSpace(cfg.ArtifactStoreRoot)
	if registryPath == "" || artifactRoot == "" {
		if production {
			// Unreachable: validateProduction already rejected empties.
			// Kept as defense-in-depth so a future validation edit cannot
			// silently open an empty-path production boot.
			return nil, fmt.Errorf("app: production requires REGISTRY_PATH and ARTIFACT_STORE_ROOT")
		}
		if registryPath == "" {
			registryPath = ":memory:"
		}
		if artifactRoot == "" {
			artifactRoot = filepath.Join(os.TempDir(), "portfolio-artifacts")
		}
	}
	if err := os.MkdirAll(artifactRoot, 0o755); err != nil {
		return nil, fmt.Errorf("app: create artifact store root: %w", err)
	}

	db, err := publishing.OpenRegistry(registryPath)
	if err != nil {
		return nil, fmt.Errorf("app: open registry: %w", err)
	}
	repo := publishing.NewSQLiteRepository(db)
	store := publishing.NewLocalArtifactStore(artifactRoot, publishing.AssetPolicy{})

	// Deployment publisher: immutable version paths plus a final pointer
	// swap. Built from the same typed fields instead of
	// deployment.LoadConfigFromEnv so Config stays the single env boundary.
	//
	// Ruling: app re-states the ARTIFACT_STORE_ROOT/LAB_PUBLIC_ORIGIN/
	// PORTFOLIO_PUBLIC_ORIGIN bindings instead of calling
	// deployment.LoadConfigFromEnv — one env-reading site beats two that can
	// disagree — cost if wrong: new deployment env vars need a line here.
	lab := deployment.NewLabPublisher(artifactRoot, cfg.LabOrigin)
	portfolio := deployment.NewPortfolioCatalogPublisher(artifactRoot, cfg.PortfolioOrigin)
	atomic := deployment.NewAtomicPublisher(lab, portfolio)

	svc := publishing.NewPublishingService(repo, store, atomic)

	// Preview runner over the shared artifact root and preview origin.
	runner := preview.NewRunner(preview.Config{
		RootDir:       artifactRoot,
		PreviewOrigin: cfg.PreviewOrigin,
	})
	previewDeployer := deployment.NewPreviewDeployer(runner)

	auth, authMux, err := wireAuth(cfg, production)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	adminAPI := adminapi.NewHandler(svc, repo)
	publicAPI := publicapi.NewHandlerWithRepo(repo)
	apiServer := api.NewServer(api.WithPublishing(svc, repo, auth))

	var mcpAuth *mcppublisher.Authenticator
	var mcp *mcppublisher.Server
	adapter := &mcppublisher.Adapter{Publishing: svc, Repo: repo}
	if strings.TrimSpace(cfg.MCPTokenSecret) != "" {
		mcpAuth, err = mcppublisher.NewAuthenticator([]byte(cfg.MCPTokenSecret), mcppublisher.AuthOptions{})
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("app: MCP authenticator: %w", err)
		}
		mcp, err = mcppublisher.NewServer(mcpAuth, adapter, mcppublisher.ServerOptions{})
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("app: MCP server: %w", err)
		}
	} else if production {
		// Unreachable: validateProduction already rejected an empty secret.
		_ = db.Close()
		return nil, fmt.Errorf("app: production requires MCP_TOKEN_SECRET")
	}

	a := &App{
		Config:          cfg,
		DB:              db,
		Repo:            repo,
		Store:           store,
		Publishing:      svc,
		Auth:            auth,
		AuthMux:         authMux,
		AdminAPI:        adminAPI,
		PublicAPI:       publicAPI,
		API:             apiServer,
		MCP:             mcp,
		MCPAuth:         mcpAuth,
		MCPAdapter:      adapter,
		Runner:          runner,
		PreviewDeployer: previewDeployer,
		Publisher:       atomic,
	}
	a.handler = &spaHandler{
		staticPath: cfg.FrontendDist,
		indexPath:  "index.html",
		shells: []shellRoute{
			{prefix: "/admin", index: "/admin/index.html"},
			{prefix: "/lab", index: "/lab/index.html"},
		},
		apiServer: apiServer,
		authMux:   authMux,
		mcp:       mcp,
	}

	log.Printf("app: composed env=%s registry=%s", cfg.Env, registryPath)
	return a, nil
}

// wireAuth builds the admin auth service, or returns nils when credentials
// are absent outside production. Production always wires the passkey
// verifier; NewService enforces the rest fail-closed.
func wireAuth(cfg Config, production bool) (*adminauth.Service, *http.ServeMux, error) {
	hasCredentials := strings.TrimSpace(cfg.AdminClientID) != "" &&
		strings.TrimSpace(cfg.AdminClientSecret) != "" &&
		strings.TrimSpace(cfg.SessionSigningKey) != ""
	if !hasCredentials {
		if production {
			// Unreachable: validateProduction already rejected empties, but
			// NewService would reject them anyway. Report here so the error
			// names the admin surface instead of leaking through.
			return nil, nil, fmt.Errorf("app: production requires ADMIN_GITHUB_CLIENT_ID, ADMIN_GITHUB_CLIENT_SECRET, ADMIN_SESSION_SIGNING_KEY")
		}
		return nil, nil, nil
	}

	verifier := cfg.ApprovalVerifier
	if verifier == nil {
		if production {
			return nil, nil, fmt.Errorf("app: production requires approval verifier (passkey)")
		}
		verifier = adminauth.NewDevApprovalVerifier(cfg.Env)
	}

	allowedLogin := strings.TrimSpace(cfg.AdminAllowedLogin)
	if allowedLogin == "" {
		if production {
			return nil, nil, fmt.Errorf("app: production requires ADMIN_ALLOWED_GITHUB_LOGIN")
		}
		allowedLogin = "gio0z"
	}

	authCfg := adminauth.Config{
		ClientID:         cfg.AdminClientID,
		ClientSecret:     cfg.AdminClientSecret,
		SessionSecret:    cfg.SessionSigningKey,
		AllowedLogin:     allowedLogin,
		PublicOrigin:     cfg.AdminPublicOrigin,
		Environment:      cfg.Env,
		ApprovalVerifier: verifier,
		SecureCookies:    production,
	}
	authSvc, err := adminauth.NewService(authCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("app: admin auth: %w", err)
	}
	mux := http.NewServeMux()
	authSvc.RegisterRoutes(mux)
	return authSvc, mux, nil
}

// spaHandler serves static frontend files with an SPA fallback, delegating
// API, admin-auth, and MCP routes to the composed subsystems.
//
// Static output is a prerendered multi-page site, not a single shell: every
// public route owns a directory with its own index.html, so the handler serves
// a real file when one exists at the requested path or at its directory index.
// The fallback is reserved for the two areas that genuinely need it — the
// client-routed admin and Design Lab applications, whose React Router owns
// every path beneath them — and for any public path with no prerendered
// document.
type spaHandler struct {
	staticPath string
	indexPath  string
	shells     []shellRoute
	apiServer  *api.Server
	authMux    *http.ServeMux
	mcp        *mcppublisher.Server
}

// shellRoute is one client-routed area: every path under prefix is answered by
// the single document at index. The admin and Design Lab applications are
// single-page apps, so only their entry document exists as a file while React
// Router owns the rest.
type shellRoute struct {
	prefix string
	index  string
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.mcp != nil && r.URL.Path == mcppublisher.MCPRoute {
		h.mcp.ServeHTTP(w, r)
		return
	}
	if h.authMux != nil && strings.HasPrefix(r.URL.Path, "/api/admin/auth/") {
		h.authMux.ServeHTTP(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		h.apiServer.ServeHTTP(w, r)
		return
	}

	if h.serveFile(w, r, r.URL.Path) {
		return
	}

	// Falling back to the public index for a client-routed area would serve the
	// public site at an application URL.
	for _, shell := range h.shells {
		if matchesPrefix(r.URL.Path, shell.prefix) && h.serveFile(w, r, shell.index) {
			return
		}
	}

	indexPath := filepath.Join(h.staticPath, h.indexPath)
	if _, err := os.Stat(indexPath); err == nil {
		http.ServeFile(w, r, indexPath)
		return
	}

	h.apiServer.ServeHTTP(w, r)
}

// matchesPrefix reports whether path is prefix itself or lives beneath it.
// "/administrator" is a different route and must not match "/admin".
func matchesPrefix(path, prefix string) bool {
	if prefix == "" {
		return false
	}
	trimmed := strings.TrimSuffix(prefix, "/")
	return path == trimmed || strings.HasPrefix(path, trimmed+"/")
}

// serveFile serves the static file for urlPath, falling back to the directory
// index when urlPath names a directory. It reports whether a response was
// written.
func (h *spaHandler) serveFile(w http.ResponseWriter, r *http.Request, urlPath string) bool {
	path := filepath.Join(h.staticPath, filepath.Clean("/"+urlPath))
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		path = filepath.Join(path, h.indexPath)
		if fi, err = os.Stat(path); err != nil || fi.IsDir() {
			return false
		}
	}
	http.ServeFile(w, r, path)
	return true
}
