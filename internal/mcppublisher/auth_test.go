package mcppublisher

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testAuthenticator(t *testing.T) *Authenticator {
	t.Helper()
	auth, err := NewAuthenticator([]byte("test-secret-that-is-long-enough-32b"), AuthOptions{})
	if err != nil {
		t.Fatalf("NewAuthenticator() error = %v", err)
	}
	return auth
}

func bearerRequest(token string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/mcp/portfolio", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestAuthAcceptsValidScope(t *testing.T) {
	auth := testAuthenticator(t)
	token, _, err := auth.Issue("agent:hermes-1", "internal", []string{ScopePortfolioRead, ScopeLabDraftCreate}, time.Hour)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	id, err := auth.Authenticate(bearerRequest(token))
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if id.Actor.Identity != "agent:hermes-1" {
		t.Errorf("Actor.Identity = %q, want %q", id.Actor.Identity, "agent:hermes-1")
	}
	if id.ProfileID != "internal" {
		t.Errorf("ProfileID = %q, want %q", id.ProfileID, "internal")
	}
	if err := RequireScope(id, ScopeLabDraftCreate); err != nil {
		t.Errorf("RequireScope(valid) error = %v", err)
	}
}

func TestAuthRejectsMissingToken(t *testing.T) {
	auth := testAuthenticator(t)
	if _, err := auth.Authenticate(bearerRequest("")); err == nil {
		t.Fatal("Authenticate() without token succeeded, want unauthorized")
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp/portfolio", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	if _, err := auth.Authenticate(req); err == nil {
		t.Fatal("Authenticate() with non-bearer scheme succeeded, want unauthorized")
	}
}

func TestAuthRejectsInvalidSignature(t *testing.T) {
	auth := testAuthenticator(t)
	token, _, err := auth.Issue("agent:hermes-1", "internal", []string{ScopePortfolioRead}, time.Hour)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		t.Fatalf("token has %d parts, want 2", len(parts))
	}
	tampered := parts[0] + ".AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	if _, err := auth.Authenticate(bearerRequest(tampered)); err == nil {
		t.Fatal("Authenticate() with forged signature succeeded, want unauthorized")
	}
}

func TestAuthRejectsExpiredToken(t *testing.T) {
	auth := testAuthenticator(t)
	claims := &Claims{
		TokenID:   "tid-expired",
		ActorID:   "agent:hermes-1",
		ProfileID: "internal",
		Scopes:    []string{ScopePortfolioRead},
		IssuedAt:  time.Now().Add(-2 * time.Hour).UTC(),
		ExpiresAt: time.Now().Add(-time.Hour).UTC(),
	}
	token, err := auth.signClaims(claims)
	if err != nil {
		t.Fatalf("signClaims() error = %v", err)
	}
	if _, err := auth.Authenticate(bearerRequest(token)); err == nil {
		t.Fatal("Authenticate() with expired token succeeded, want unauthorized")
	}
}

func TestAuthRejectsRevokedToken(t *testing.T) {
	auth := testAuthenticator(t)
	token, claims, err := auth.Issue("agent:hermes-1", "internal", []string{ScopePortfolioRead}, time.Hour)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	auth.Revoke(claims.TokenID)
	if _, err := auth.Authenticate(bearerRequest(token)); err == nil {
		t.Fatal("Authenticate() with revoked token succeeded, want unauthorized")
	}
}

func TestIssueRejectsForbiddenScopes(t *testing.T) {
	auth := testAuthenticator(t)
	forbidden := []string{
		"lab:approve",
		"lab:publish",
		"admin:read",
		"admin:approve",
		"secret:read",
		"shell:exec",
		"filesystem:read",
		"sql:query",
	}
	for _, scope := range forbidden {
		if _, _, err := auth.Issue("agent:hermes-1", "internal", []string{scope}, time.Hour); err == nil {
			t.Errorf("Issue(scope=%q) succeeded, want rejection", scope)
		}
	}
}

func TestAuthRejectsTokenCarryingForbiddenScope(t *testing.T) {
	auth := testAuthenticator(t)
	claims := &Claims{
		TokenID:   "tid-evil",
		ActorID:   "agent:hermes-1",
		ProfileID: "internal",
		Scopes:    []string{ScopePortfolioRead, "lab:publish"},
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().Add(time.Hour).UTC(),
	}
	token, err := auth.signClaims(claims)
	if err != nil {
		t.Fatalf("signClaims() error = %v", err)
	}
	if _, err := auth.Authenticate(bearerRequest(token)); err == nil {
		t.Fatal("Authenticate() with lab:publish scope succeeded, want unauthorized")
	}
}

func TestRequireScopeRejectsMissingScope(t *testing.T) {
	auth := testAuthenticator(t)
	token, _, err := auth.Issue("agent:hermes-1", "internal", []string{ScopePortfolioRead}, time.Hour)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	id, err := auth.Authenticate(bearerRequest(token))
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if err := RequireScope(id, ScopeLabDraftCreate); err == nil {
		t.Fatal("RequireScope(missing) succeeded, want forbidden")
	}
}

func TestIssueRejectsBadInput(t *testing.T) {
	auth := testAuthenticator(t)
	if _, _, err := auth.Issue("", "internal", []string{ScopePortfolioRead}, time.Hour); err == nil {
		t.Error("Issue(empty actor) succeeded, want error")
	}
	if _, _, err := auth.Issue("agent:x", "", []string{ScopePortfolioRead}, time.Hour); err == nil {
		t.Error("Issue(empty profile) succeeded, want error")
	}
	if _, _, err := auth.Issue("agent:x", "internal", nil, time.Hour); err == nil {
		t.Error("Issue(no scopes) succeeded, want error")
	}
	if _, _, err := auth.Issue("agent:x", "internal", []string{"unknown:scope"}, time.Hour); err == nil {
		t.Error("Issue(unknown scope) succeeded, want error")
	}
	if _, _, err := auth.Issue("agent:x", "internal", []string{ScopePortfolioRead}, 48*time.Hour); err == nil {
		t.Error("Issue(excessive TTL) succeeded, want error")
	}
}

func TestNewAuthenticatorRejectsShortSecret(t *testing.T) {
	if _, err := NewAuthenticator([]byte("short"), AuthOptions{}); err == nil {
		t.Fatal("NewAuthenticator(short secret) succeeded, want error")
	}
}
