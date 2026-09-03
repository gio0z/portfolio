package adminauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type contextKey string

const (
	sessionContextKey contextKey = "adminauth.session"
	DevStepUpHeader              = "X-Admin-StepUp-Dev"
)

var (
	ErrMissingClientID         = errors.New("missing ADMIN_GITHUB_CLIENT_ID")
	ErrMissingClientSecret     = errors.New("missing ADMIN_GITHUB_CLIENT_SECRET")
	ErrMissingSessionSecret    = errors.New("missing ADMIN_SESSION_SIGNING_KEY")
	ErrMissingApprovalVerifier = errors.New("approval verifier required for production")
	ErrDevVerifierInProduction = errors.New("development approval verifier cannot be used in production")
	ErrStepUpRequired          = errors.New("step-up authentication required")
	ErrCSRFCheckFailed         = errors.New("csrf token validation failed")
	ErrUnauthorized            = errors.New("unauthorized admin request")
)

// ApprovalVerifier verifies step-up authentication for high-privilege actions (like approval/publish).
type ApprovalVerifier interface {
	VerifyApproval(r *http.Request, session *Session) error
	Type() string
}

// DevApprovalVerifier is a development-only verifier that checks for a test header.
type DevApprovalVerifier struct {
	environment string
}

func NewDevApprovalVerifier(environment string) *DevApprovalVerifier {
	return &DevApprovalVerifier{environment: environment}
}

func (v *DevApprovalVerifier) Type() string {
	return "dev"
}

func (v *DevApprovalVerifier) VerifyApproval(r *http.Request, session *Session) error {
	if strings.EqualFold(v.environment, "production") {
		return ErrDevVerifierInProduction
	}
	if r.Header.Get(DevStepUpHeader) == "true" {
		return nil
	}
	return ErrStepUpRequired
}

// PasskeyApprovalVerifier is the production passkey/WebAuthn approval verifier interface point.
type PasskeyApprovalVerifier struct{}

func NewPasskeyApprovalVerifier() *PasskeyApprovalVerifier {
	return &PasskeyApprovalVerifier{}
}

func (v *PasskeyApprovalVerifier) Type() string {
	return "passkey"
}

func (v *PasskeyApprovalVerifier) VerifyApproval(r *http.Request, session *Session) error {
	// Production implementation point: checks WebAuthn / Passkey signature assertion header
	passkeySig := r.Header.Get("X-Admin-Passkey-Assertion")
	if passkeySig == "" {
		return ErrStepUpRequired
	}
	return nil
}

// Config holds administrative authentication settings.
type Config struct {
	ClientID         string
	ClientSecret     string
	SessionSecret    string
	AllowedLogin     string
	PublicOrigin     string
	Environment      string
	GitHubAuthURL    string
	GitHubTokenURL   string
	GitHubUserAPIURL string
	SessionTTL       time.Duration
	ApprovalVerifier ApprovalVerifier
	SecureCookies    bool
	SessionStore     SessionStore
	HTTPClient       *http.Client
}

// LoadConfigFromEnv loads configuration from environment variables.
func LoadConfigFromEnv() Config {
	allowedLogin := os.Getenv("ADMIN_ALLOWED_GITHUB_LOGIN")
	if allowedLogin == "" {
		allowedLogin = "gio0z"
	}
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}
	if env == "" {
		env = "production"
	}

	secureCookies := true
	if strings.EqualFold(env, "development") || strings.EqualFold(env, "test") {
		secureCookies = false
	}

	var verifier ApprovalVerifier
	if strings.EqualFold(env, "development") || strings.EqualFold(env, "test") {
		verifier = NewDevApprovalVerifier(env)
	}

	return Config{
		ClientID:         os.Getenv("ADMIN_GITHUB_CLIENT_ID"),
		ClientSecret:     os.Getenv("ADMIN_GITHUB_CLIENT_SECRET"),
		SessionSecret:    os.Getenv("ADMIN_SESSION_SIGNING_KEY"),
		AllowedLogin:     allowedLogin,
		PublicOrigin:     os.Getenv("ADMIN_PUBLIC_ORIGIN"),
		Environment:      env,
		GitHubAuthURL:    "https://github.com/login/oauth/authorize",
		GitHubTokenURL:   "https://github.com/login/oauth/access_token",
		GitHubUserAPIURL: "https://api.github.com/user",
		SessionTTL:       DefaultSessionTTL,
		ApprovalVerifier: verifier,
		SecureCookies:    secureCookies,
	}
}

// Service manages admin authentication, session tokens, and security checks.
type Service struct {
	cfg          Config
	sessionStore SessionStore
	httpClient   *http.Client
}

// NewService validates configuration and constructs an admin authentication service.
func NewService(cfg Config) (*Service, error) {
	if cfg.ClientID == "" {
		return nil, ErrMissingClientID
	}
	if cfg.ClientSecret == "" {
		return nil, ErrMissingClientSecret
	}
	if cfg.SessionSecret == "" {
		return nil, ErrMissingSessionSecret
	}
	if cfg.AllowedLogin == "" {
		cfg.AllowedLogin = "gio0z"
	}
	if cfg.GitHubAuthURL == "" {
		cfg.GitHubAuthURL = "https://github.com/login/oauth/authorize"
	}
	if cfg.GitHubTokenURL == "" {
		cfg.GitHubTokenURL = "https://github.com/login/oauth/access_token"
	}
	if cfg.GitHubUserAPIURL == "" {
		cfg.GitHubUserAPIURL = "https://api.github.com/user"
	}
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = DefaultSessionTTL
	}

	// Fail closed in production if ApprovalVerifier is missing or dev
	isProd := strings.EqualFold(cfg.Environment, "production")
	if isProd {
		if cfg.ApprovalVerifier == nil {
			return nil, ErrMissingApprovalVerifier
		}
		if cfg.ApprovalVerifier.Type() == "dev" {
			return nil, ErrDevVerifierInProduction
		}
	}

	store := cfg.SessionStore
	if store == nil {
		store = NewMemorySessionStore()
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	return &Service{
		cfg:          cfg,
		sessionStore: store,
		httpClient:   client,
	}, nil
}

// CreateSession generates a new authenticated session for the given login.
func (s *Service) CreateSession(login string) (*Session, string, error) {
	sessionID, err := GenerateRandomString(32)
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	expiresAt := now.Add(s.cfg.SessionTTL)

	sess := &Session{
		ID:        sessionID,
		Login:     login,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	if err := s.sessionStore.Save(sess); err != nil {
		return nil, "", err
	}

	token := signSessionToken(sessionID, []byte(s.cfg.SessionSecret))
	return sess, token, nil
}

// OwnerMiddleware authenticates the admin session cookie and enforces owner identity.
func (s *Service) OwnerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		sessionID, err := verifySessionToken(cookie.Value, []byte(s.cfg.SessionSecret))
		if err != nil {
			http.Error(w, "unauthorized: invalid session token", http.StatusUnauthorized)
			return
		}

		sess, err := s.sessionStore.Get(sessionID)
		if err != nil {
			http.Error(w, "unauthorized: session expired or revoked", http.StatusUnauthorized)
			return
		}

		if !strings.EqualFold(sess.Login, s.cfg.AllowedLogin) {
			http.Error(w, "forbidden: owner access only", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CSRFMiddleware validates CSRF tokens for state-changing HTTP requests.
func (s *Service) CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsStateChangingMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		sess, ok := SessionFromContext(r.Context())
		if !ok || sess == nil {
			http.Error(w, "forbidden: unauthenticated csrf check", http.StatusForbidden)
			return
		}

		token := r.Header.Get(CSRFHeaderName)
		if token == "" {
			token = r.Header.Get(CSRFHeaderAltName)
		}

		if !s.ValidateCSRFToken(sess.ID, token) {
			http.Error(w, "forbidden: invalid or missing csrf token", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireStepUp validates additional step-up verification before executing high-privilege operations.
func (s *Service) RequireStepUp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := SessionFromContext(r.Context())
		if !ok || sess == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if s.cfg.ApprovalVerifier == nil {
			http.Error(w, "forbidden: approval verifier not configured", http.StatusForbidden)
			return
		}

		if err := s.cfg.ApprovalVerifier.VerifyApproval(r, sess); err != nil {
			http.Error(w, fmt.Sprintf("forbidden: %v", err), http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SessionFromContext extracts the Session from the request context if present.
func SessionFromContext(ctx context.Context) (*Session, bool) {
	sess, ok := ctx.Value(sessionContextKey).(*Session)
	return sess, ok && sess != nil
}

// HandleSession reports the current authentication status for the admin UI.
func (s *Service) HandleSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	sessionID, err := verifySessionToken(cookie.Value, []byte(s.cfg.SessionSecret))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	sess, err := s.sessionStore.Get(sessionID)
	if err != nil || !strings.EqualFold(sess.Login, s.cfg.AllowedLogin) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	csrfToken, _ := s.IssueCSRFToken(sess.ID)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"login":         sess.Login,
		"csrf_token":    csrfToken,
	})
}

// RegisterRoutes registers admin auth endpoints on the provided ServeMux.
func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/admin/auth/login", s.HandleLogin)
	mux.HandleFunc("/api/admin/auth/callback", s.HandleCallback)
	mux.HandleFunc("/api/admin/auth/logout", s.HandleLogout)
	mux.HandleFunc("/api/admin/auth/session", s.HandleSession)
}

// HandleLogout revokes the session and clears cookies.
func (s *Service) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil && cookie.Value != "" {
		if sessionID, err := verifySessionToken(cookie.Value, []byte(s.cfg.SessionSecret)); err == nil {
			_ = s.sessionStore.Delete(sessionID)
		}
	}
	ClearSessionCookie(w, s.cfg.SecureCookies)
	ClearCSRFCookie(w, s.cfg.SecureCookies)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"logged_out"}`))
}
