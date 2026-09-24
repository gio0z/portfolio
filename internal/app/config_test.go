package app_test

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"portfolio/internal/adminauth"
	"portfolio/internal/app"
)

// testProductionConfig returns a fully-populated production config pointing at
// isolated temp paths. Every fail-closed case starts here and blanks one item.
func testProductionConfig(t *testing.T) app.Config {
	t.Helper()
	dir := t.TempDir()
	return app.Config{
		Port:              "8080",
		FrontendDist:      dir,
		Env:               "production",
		AdminClientID:     "test-client-id",
		AdminClientSecret: "test-client-secret",
		SessionSigningKey: strings.Repeat("s", 32),
		MCPTokenSecret:    strings.Repeat("m", 32),
		AdminAllowedLogin: "gio0z",
		AdminPublicOrigin: "https://admin.example.com",
		PreviewOrigin:     "https://preview.example.com",
		LabOrigin:         "https://lab.example.com",
		PortfolioOrigin:   "https://portfolio.example.com",
		RegistryPath:      filepath.Join(dir, "registry.sqlite"),
		ArtifactStoreRoot: filepath.Join(dir, "artifacts"),
		ApprovalVerifier:  adminauth.NewPasskeyApprovalVerifier(),
	}
}

func TestProductionFailsClosed(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*app.Config)
		wantEnv string
	}{
		{
			name:    "missing signing key",
			mutate:  func(c *app.Config) { c.SessionSigningKey = "" },
			wantEnv: "ADMIN_SESSION_SIGNING_KEY",
		},
		{
			name:    "short signing key",
			mutate:  func(c *app.Config) { c.SessionSigningKey = "short" },
			wantEnv: "ADMIN_SESSION_SIGNING_KEY",
		},
		{
			name:    "missing OAuth client ID",
			mutate:  func(c *app.Config) { c.AdminClientID = "" },
			wantEnv: "ADMIN_GITHUB_CLIENT_ID",
		},
		{
			name:    "missing OAuth client secret",
			mutate:  func(c *app.Config) { c.AdminClientSecret = "" },
			wantEnv: "ADMIN_GITHUB_CLIENT_SECRET",
		},
		{
			name:    "missing owner allowlist",
			mutate:  func(c *app.Config) { c.AdminAllowedLogin = "" },
			wantEnv: "ADMIN_ALLOWED_GITHUB_LOGIN",
		},
		{
			name:    "missing approval verifier",
			mutate:  func(c *app.Config) { c.ApprovalVerifier = nil },
			wantEnv: "approval verifier",
		},
		{
			name:    "dev approval verifier in production",
			mutate:  func(c *app.Config) { c.ApprovalVerifier = adminauth.NewDevApprovalVerifier("production") },
			wantEnv: "approval verifier",
		},
		{
			name:    "missing admin public origin",
			mutate:  func(c *app.Config) { c.AdminPublicOrigin = "" },
			wantEnv: "ADMIN_PUBLIC_ORIGIN",
		},
		{
			name:    "missing preview origin",
			mutate:  func(c *app.Config) { c.PreviewOrigin = "" },
			wantEnv: "PREVIEW_PUBLIC_ORIGIN",
		},
		{
			name:    "missing lab origin",
			mutate:  func(c *app.Config) { c.LabOrigin = "" },
			wantEnv: "LAB_PUBLIC_ORIGIN",
		},
		{
			name:    "missing portfolio origin",
			mutate:  func(c *app.Config) { c.PortfolioOrigin = "" },
			wantEnv: "PORTFOLIO_PUBLIC_ORIGIN",
		},
		{
			name:    "missing registry path",
			mutate:  func(c *app.Config) { c.RegistryPath = "" },
			wantEnv: "REGISTRY_PATH",
		},
		{
			name:    "missing artifact store root",
			mutate:  func(c *app.Config) { c.ArtifactStoreRoot = "" },
			wantEnv: "ARTIFACT_STORE_ROOT",
		},
		{
			name:    "missing MCP token secret",
			mutate:  func(c *app.Config) { c.MCPTokenSecret = "" },
			wantEnv: "MCP_TOKEN_SECRET",
		},
		{
			name:    "short MCP token secret",
			mutate:  func(c *app.Config) { c.MCPTokenSecret = "short" },
			wantEnv: "MCP_TOKEN_SECRET",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testProductionConfig(t)
			tc.mutate(&cfg)
			a, err := app.New(cfg)
			if err == nil {
				_ = a.Close()
				t.Fatalf("New succeeded with %s; want production fail-closed error", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantEnv) {
				t.Fatalf("error %q does not name %q", err.Error(), tc.wantEnv)
			}
		})
	}
}

func TestProductionComposesAllSubsystems(t *testing.T) {
	cfg := testProductionConfig(t)
	a, err := app.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() {
		if err := a.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	if a.Repo == nil {
		t.Error("Repo is nil")
	}
	if a.Store == nil {
		t.Error("Store is nil")
	}
	if a.Publishing == nil {
		t.Error("Publishing is nil")
	}
	if a.Auth == nil {
		t.Error("Auth is nil")
	}
	if a.AdminAPI == nil {
		t.Error("AdminAPI is nil")
	}
	if a.PublicAPI == nil {
		t.Error("PublicAPI is nil")
	}
	if a.API == nil {
		t.Error("API is nil")
	}
	if a.MCP == nil {
		t.Error("MCP is nil")
	}
	if a.Runner == nil {
		t.Error("Runner is nil")
	}
	if a.Publisher == nil {
		t.Error("Publisher is nil")
	}

	// The composed handler serves the API surface.
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/health = %d; want 200", rec.Code)
	}
}

func TestDevAllowsMissingAuth(t *testing.T) {
	dir := t.TempDir()
	cfg := app.Config{
		Env:               "test",
		Port:              "8080",
		FrontendDist:      dir,
		AdminAllowedLogin: "",
		RegistryPath:      "",
		ArtifactStoreRoot: "",
	}
	a, err := app.New(cfg)
	if err != nil {
		t.Fatalf("New in test env without credentials: %v", err)
	}
	defer func() {
		if err := a.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()
	if a.Auth != nil {
		t.Error("Auth should be nil when credentials are absent in test env")
	}
	if a.MCP != nil {
		t.Error("MCP should be nil when the token secret is absent in test env")
	}
	if a.Publishing == nil || a.Repo == nil || a.Runner == nil || a.Publisher == nil {
		t.Error("core publishing subsystems must be wired even when auth is disabled")
	}
}

func TestSecretsNotLogged(t *testing.T) {
	const (
		clientSecret = "csecret-MARKER-BBB-0123456789abcdef-0123456789abcdef"
		sessionKey   = "sesskey-MARKER-CCC-0123456789abcdef-0123456789abcdef"
		mcpSecret    = "mcpsec-MARKER-DDD-0123456789abcdef-0123456789abcdef"
	)
	dir := t.TempDir()
	t.Setenv("APP_ENV", "test")
	t.Setenv("ENVIRONMENT", "")
	t.Setenv("ADMIN_GITHUB_CLIENT_ID", "cid-public-value")
	t.Setenv("ADMIN_GITHUB_CLIENT_SECRET", clientSecret)
	t.Setenv("ADMIN_SESSION_SIGNING_KEY", sessionKey)
	t.Setenv("MCP_TOKEN_SECRET", mcpSecret)
	t.Setenv("ADMIN_ALLOWED_GITHUB_LOGIN", "gio0z")
	t.Setenv("ADMIN_PUBLIC_ORIGIN", "https://admin.example.com")
	t.Setenv("PREVIEW_PUBLIC_ORIGIN", "https://preview.example.com")
	t.Setenv("LAB_PUBLIC_ORIGIN", "https://lab.example.com")
	t.Setenv("PORTFOLIO_PUBLIC_ORIGIN", "https://portfolio.example.com")
	t.Setenv("REGISTRY_PATH", filepath.Join(dir, "registry.sqlite"))
	t.Setenv("ARTIFACT_STORE_ROOT", filepath.Join(dir, "artifacts"))

	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(orig)

	cfg := app.LoadConfigFromEnv()
	a, err := app.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	out := buf.String()
	for _, secret := range []string{clientSecret, sessionKey, mcpSecret} {
		if strings.Contains(out, secret) {
			t.Fatalf("log output contains secret value %q", secret)
		}
	}
	redacted := cfg.String()
	for _, secret := range []string{clientSecret, sessionKey, mcpSecret} {
		if strings.Contains(redacted, secret) {
			t.Fatalf("Config.String contains secret value %q", secret)
		}
	}
}

func TestLoadConfigHostDefaultsToAllInterfaces(t *testing.T) {
	t.Setenv("HOST", "")
	cfg := app.LoadConfigFromEnv()
	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want 0.0.0.0 (an unset HOST must not change existing behaviour)", cfg.Host)
	}
}

func TestLoadConfigHostFromEnv(t *testing.T) {
	// A deployment that has no firewall must be able to keep the admin-capable
	// API off every non-loopback interface.
	t.Setenv("HOST", "127.0.0.1")
	cfg := app.LoadConfigFromEnv()
	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want 127.0.0.1", cfg.Host)
	}
}
