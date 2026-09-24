// Package app owns application composition and typed configuration.
//
// Config is the single surface Task 17 consumes: every subsystem reads its
// bindings from Config, never from the process environment directly.
// LoadConfigFromEnv is the only os.Getenv boundary; New wires repository,
// artifact store, publishing service, admin auth/API, public API, MCP server,
// preview runner, and deployment publisher, failing closed in production.
package app

import (
	"fmt"
	"os"
	"strings"

	"portfolio/internal/adminauth"
)

// Config carries every binding the application needs. Secrets have no
// defaults: an empty secret means "not configured" and fails validation in
// production. Non-secret operational defaults (port, frontend dist) are
// applied by LoadConfigFromEnv.
type Config struct {
	// Port is the TCP port the server listens on. Defaults to 8080.
	Port string
	// FrontendDist is the path to the built frontend served as static files.
	// Defaults to ./frontend/dist.
	FrontendDist string
	// Env is the runtime environment. "development" and "test" relax admin
	// auth (insecure cookies, dev approval verifier). Anything else,
	// including the default, is production and fails closed.
	Env string

	// AdminClientID is ADMIN_GITHUB_CLIENT_ID. Required in production.
	AdminClientID string
	// AdminClientSecret is ADMIN_GITHUB_CLIENT_SECRET. Required in production.
	AdminClientSecret string
	// SessionSigningKey is ADMIN_SESSION_SIGNING_KEY (HMAC secret for admin
	// session + CSRF tokens, at least 32 bytes). Required in production.
	SessionSigningKey string
	// MCPTokenSecret is MCP_TOKEN_SECRET (HMAC secret minting short-lived
	// agent bearer tokens, at least 32 bytes). Required in production.
	MCPTokenSecret string
	// AdminAllowedLogin is ADMIN_ALLOWED_GITHUB_LOGIN, the single GitHub
	// account permitted into the admin area. Required in production.
	AdminAllowedLogin string
	// AdminPublicOrigin is ADMIN_PUBLIC_ORIGIN (scheme + host, no trailing
	// slash), pinning the OAuth redirect URI. Required in production.
	AdminPublicOrigin string

	// PreviewOrigin is PREVIEW_PUBLIC_ORIGIN. Required in production.
	PreviewOrigin string
	// LabOrigin is LAB_PUBLIC_ORIGIN. Required in production.
	LabOrigin string
	// PortfolioOrigin is PORTFOLIO_PUBLIC_ORIGIN. Required in production.
	PortfolioOrigin string

	// RegistryPath is REGISTRY_PATH, the SQLite registry file opened by
	// publishing.OpenRegistry. Required in production; :memory: in
	// development/test when unset.
	RegistryPath string
	// ArtifactStoreRoot is ARTIFACT_STORE_ROOT, the filesystem root for the
	// content-addressed artifact store, preview bundles, and Lab/portfolio
	// version paths. Required in production; a temp dir otherwise.
	ArtifactStoreRoot string

	// ApprovalVerifier enforces step-up authentication for approve-and-publish.
	// LoadConfigFromEnv wires PasskeyApprovalVerifier in production and
	// DevApprovalVerifier elsewhere. A programmatic Config must set it
	// explicitly: production rejects nil and dev verifiers.
	ApprovalVerifier adminauth.ApprovalVerifier
}

// IsProduction reports whether Env selects fail-closed production behavior.
// Empty Env defaults to production.
func (c Config) IsProduction() bool {
	return isProductionEnv(c.Env)
}

func isProductionEnv(env string) bool {
	return !strings.EqualFold(env, "development") && !strings.EqualFold(env, "test")
}

func normalizeEnv(env string) string {
	if env == "" {
		return "production"
	}
	return env
}

// LoadConfigFromEnv reads the full application configuration from the
// environment. Secrets default to empty (unset); operational values carry
// safe defaults. This is the only place that touches os.Getenv.
//
// Ruling: LoadConfigFromEnv auto-wires PasskeyApprovalVerifier in production
// instead of leaving verifier selection to main — the deployment runbook
// already mandates passkey in production, and construction-time wiring keeps
// a missing-verifier deployment unreachable by default — cost if wrong: a
// future non-passkey production verifier needs a new env switch here.
func LoadConfigFromEnv() Config {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}
	env = normalizeEnv(env)

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	dist := strings.TrimSpace(os.Getenv("FRONTEND_DIST"))
	if dist == "" {
		dist = "./frontend/dist"
	}

	var verifier adminauth.ApprovalVerifier
	if isProductionEnv(env) {
		verifier = adminauth.NewPasskeyApprovalVerifier()
	} else {
		verifier = adminauth.NewDevApprovalVerifier(env)
	}

	return Config{
		Port:              port,
		FrontendDist:      dist,
		Env:               env,
		AdminClientID:     strings.TrimSpace(os.Getenv("ADMIN_GITHUB_CLIENT_ID")),
		AdminClientSecret: strings.TrimSpace(os.Getenv("ADMIN_GITHUB_CLIENT_SECRET")),
		SessionSigningKey: strings.TrimSpace(os.Getenv("ADMIN_SESSION_SIGNING_KEY")),
		MCPTokenSecret:    strings.TrimSpace(os.Getenv("MCP_TOKEN_SECRET")),
		AdminAllowedLogin: strings.TrimSpace(os.Getenv("ADMIN_ALLOWED_GITHUB_LOGIN")),
		AdminPublicOrigin: strings.TrimSpace(os.Getenv("ADMIN_PUBLIC_ORIGIN")),
		PreviewOrigin:     strings.TrimSpace(os.Getenv("PREVIEW_PUBLIC_ORIGIN")),
		LabOrigin:         strings.TrimSpace(os.Getenv("LAB_PUBLIC_ORIGIN")),
		PortfolioOrigin:   strings.TrimSpace(os.Getenv("PORTFOLIO_PUBLIC_ORIGIN")),
		RegistryPath:      strings.TrimSpace(os.Getenv("REGISTRY_PATH")),
		ArtifactStoreRoot: strings.TrimSpace(os.Getenv("ARTIFACT_STORE_ROOT")),
		ApprovalVerifier:  verifier,
	}
}

// minSecretBytes is the minimum HMAC secret length for session signing and
// MCP token minting.
//
// Ruling: 32-byte minimum for both secrets — mcppublisher.NewAuthenticator
// already rejects shorter token secrets, so one rule covers both — cost if
// wrong: deployments with 16-byte legacy keys must regenerate.
const minSecretBytes = 32

// validateProduction fails closed when any production binding is missing or
// weak. Every error names the env var the operator must set; errors never
// carry secret values.
func (c Config) validateProduction() error {
	var missing []string
	required := func(value, envVar string) {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, envVar)
		}
	}
	required(c.AdminClientID, "ADMIN_GITHUB_CLIENT_ID")
	required(c.AdminClientSecret, "ADMIN_GITHUB_CLIENT_SECRET")
	required(c.SessionSigningKey, "ADMIN_SESSION_SIGNING_KEY")
	required(c.MCPTokenSecret, "MCP_TOKEN_SECRET")
	required(c.AdminAllowedLogin, "ADMIN_ALLOWED_GITHUB_LOGIN")
	required(c.AdminPublicOrigin, "ADMIN_PUBLIC_ORIGIN")
	required(c.PreviewOrigin, "PREVIEW_PUBLIC_ORIGIN")
	required(c.LabOrigin, "LAB_PUBLIC_ORIGIN")
	required(c.PortfolioOrigin, "PORTFOLIO_PUBLIC_ORIGIN")
	required(c.RegistryPath, "REGISTRY_PATH")
	required(c.ArtifactStoreRoot, "ARTIFACT_STORE_ROOT")
	if len(missing) > 0 {
		return fmt.Errorf("app: production requires %s", strings.Join(missing, ", "))
	}
	if len(c.SessionSigningKey) < minSecretBytes {
		return fmt.Errorf("app: production requires ADMIN_SESSION_SIGNING_KEY of at least %d bytes", minSecretBytes)
	}
	if len(c.MCPTokenSecret) < minSecretBytes {
		return fmt.Errorf("app: production requires MCP_TOKEN_SECRET of at least %d bytes", minSecretBytes)
	}
	if c.ApprovalVerifier == nil {
		return fmt.Errorf("app: production requires approval verifier (passkey)")
	}
	if c.ApprovalVerifier.Type() == "dev" {
		return fmt.Errorf("app: production requires approval verifier (passkey), got dev verifier")
	}
	return nil
}

// String renders Config with secrets redacted. Safe for logs.
//
// Ruling: hand-written redaction instead of fmt %#v — struct dumps leak new
// secret fields by default; an explicit allowlist fails open-safe — cost if
// wrong: each new field needs one line here.
func (c Config) String() string {
	return fmt.Sprintf(
		"Config{Port:%q FrontendDist:%q Env:%q AdminClientID:%q AdminClientSecret:<redacted> SessionSigningKey:<redacted> MCPTokenSecret:<redacted> AdminAllowedLogin:%q AdminPublicOrigin:%q PreviewOrigin:%q LabOrigin:%q PortfolioOrigin:%q RegistryPath:%q ArtifactStoreRoot:%q ApprovalVerifier:%s}",
		c.Port, c.FrontendDist, c.Env, c.AdminClientID,
		c.AdminAllowedLogin, c.AdminPublicOrigin,
		c.PreviewOrigin, c.LabOrigin, c.PortfolioOrigin,
		c.RegistryPath, c.ArtifactStoreRoot, verifierType(c.ApprovalVerifier),
	)
}

func verifierType(v adminauth.ApprovalVerifier) string {
	if v == nil {
		return "<nil>"
	}
	return v.Type()
}
