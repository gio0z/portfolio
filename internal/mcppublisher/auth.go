// Package mcppublisher exposes a narrow, authenticated MCP adapter over the
// design-lab publishing workflow.
//
// The MCP surface is intentionally limited: agent credentials carry only the
// least-privilege scopes in AgentScopes, every tool forces the
// publishing.ActorAgent identity server-side, and no approval, publication,
// permission, token, or permanent-deletion capability exists anywhere in this
// package. An agent may progress a submission only as far as IN_REVIEW; the
// publishing service itself rejects any agent attempt to approve or publish.
package mcppublisher

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"portfolio/internal/publishing"
)

// Agent-accessible MCP scopes. Approval, publication, administration,
// secret-management, and arbitrary-execution scopes deliberately do not exist:
// the allowlist below is the entire universe of scopes an MCP token may carry.
const (
	ScopePortfolioRead    = "portfolio:read"
	ScopeLabDraftCreate   = "lab:draft:create"
	ScopeLabDraftUpdate   = "lab:draft:update"
	ScopeLabAssetUpload   = "lab:asset:upload"
	ScopeLabPreviewDeploy = "lab:preview:deploy"
	ScopeLabReviewRequest = "lab:review:request"
)

// DefaultMaxTokenTTL bounds agent credential lifetime. Short-lived,
// revocable credentials are required by the design spec.
const DefaultMaxTokenTTL = 24 * time.Hour

// Sentinel errors. Unauthorized conditions (missing, forged, expired, or
// revoked tokens) never disclose which check failed beyond the status.
var (
	ErrUnauthorized = errors.New("mcppublisher: unauthorized")
	ErrForbidden    = errors.New("mcppublisher: forbidden")
	ErrUnknownTool  = errors.New("mcppublisher: unknown tool")
)

var allowedScopes = map[string]struct{}{
	ScopePortfolioRead:    {},
	ScopeLabDraftCreate:   {},
	ScopeLabDraftUpdate:   {},
	ScopeLabAssetUpload:   {},
	ScopeLabPreviewDeploy: {},
	ScopeLabReviewRequest: {},
}

// AgentScopes returns every scope an agent token may hold, in stable order.
func AgentScopes() []string {
	return []string{
		ScopePortfolioRead,
		ScopeLabDraftCreate,
		ScopeLabDraftUpdate,
		ScopeLabAssetUpload,
		ScopeLabPreviewDeploy,
		ScopeLabReviewRequest,
	}
}

// Claims is the authenticated content of an MCP bearer token: actor ID,
// profile ID, scopes, issued-at, expiry, and token ID.
type Claims struct {
	TokenID   string    `json:"tid"`
	ActorID   string    `json:"actor"`
	ProfileID string    `json:"profile"`
	Scopes    []string  `json:"scopes"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

// Identity is the server-side view of an authenticated MCP caller. The actor
// kind is always publishing.ActorAgent: this package never mints or honors
// owner credentials, so approval and publication stay unreachable.
type Identity struct {
	Actor     publishing.Actor
	ProfileID string
	Scopes    []string
	TokenID   string
}

// HasScope reports whether the identity carries the given scope.
func (id Identity) HasScope(scope string) bool {
	for _, s := range id.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// RequireScope enforces that the identity carries the given scope.
func RequireScope(id Identity, scope string) error {
	if id.HasScope(scope) {
		return nil
	}
	return fmt.Errorf("%w: missing scope %q", ErrForbidden, scope)
}

// AuthOptions tunes the authenticator. A zero MaxTTL selects
// DefaultMaxTokenTTL.
type AuthOptions struct {
	MaxTTL time.Duration
}

// Authenticator issues and verifies HMAC-signed MCP bearer tokens and tracks
// revocations. It is safe for concurrent use.
type Authenticator struct {
	secret  []byte
	maxTTL  time.Duration
	mu      sync.RWMutex
	revoked map[string]time.Time
}

// NewAuthenticator constructs an authenticator around a caller-supplied
// secret of at least 32 bytes.
func NewAuthenticator(secret []byte, opts AuthOptions) (*Authenticator, error) {
	if len(secret) < 32 {
		return nil, errors.New("mcppublisher: token secret must be at least 32 bytes")
	}
	maxTTL := opts.MaxTTL
	if maxTTL <= 0 {
		maxTTL = DefaultMaxTokenTTL
	}
	cpy := make([]byte, len(secret))
	copy(cpy, secret)
	return &Authenticator{
		secret:  cpy,
		maxTTL:  maxTTL,
		revoked: make(map[string]time.Time),
	}, nil
}

// Issue mints a token for the given agent actor and profile with exactly the
// requested scopes. Any scope outside the agent allowlist — including
// approval, publication, administration, secret-management, shell, filesystem,
// or SQL scopes — is rejected, as are empty identities and excessive TTLs.
func (a *Authenticator) Issue(actorID, profileID string, scopes []string, ttl time.Duration) (string, *Claims, error) {
	if strings.TrimSpace(actorID) == "" {
		return "", nil, errors.New("mcppublisher: actor ID is required")
	}
	if strings.TrimSpace(profileID) == "" {
		return "", nil, errors.New("mcppublisher: profile ID is required")
	}
	if len(scopes) == 0 {
		return "", nil, errors.New("mcppublisher: at least one scope is required")
	}
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		if _, ok := allowedScopes[scope]; !ok {
			return "", nil, fmt.Errorf("mcppublisher: scope %q is not issuable to agents", scope)
		}
		if _, dup := seen[scope]; dup {
			return "", nil, fmt.Errorf("mcppublisher: duplicate scope %q", scope)
		}
		seen[scope] = struct{}{}
	}
	if ttl <= 0 || ttl > a.maxTTL {
		return "", nil, fmt.Errorf("mcppublisher: TTL must be within (0, %s]", a.maxTTL)
	}
	var tid [16]byte
	if _, err := rand.Read(tid[:]); err != nil {
		return "", nil, fmt.Errorf("mcppublisher: generate token ID: %w", err)
	}
	now := time.Now().UTC()
	claims := &Claims{
		TokenID:   fmt.Sprintf("%x", tid),
		ActorID:   actorID,
		ProfileID: profileID,
		Scopes:    append([]string(nil), scopes...),
		IssuedAt:  now,
		ExpiresAt: now.Add(ttl),
	}
	token, err := a.signClaims(claims)
	if err != nil {
		return "", nil, err
	}
	return token, claims, nil
}

// Revoke invalidates the token with the given token ID. Expiry still applies;
// revocation is an additional fail-closed check.
func (a *Authenticator) Revoke(tokenID string) {
	if tokenID == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pruneLocked()
	a.revoked[tokenID] = time.Now().UTC().Add(a.maxTTL)
}

// Authenticate verifies the bearer token on an incoming request and returns
// the caller identity. Missing, malformed, forged, expired, revoked, or
// over-scoped tokens are all rejected as unauthorized.
func (a *Authenticator) Authenticate(r *http.Request) (Identity, error) {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) || strings.TrimSpace(header[len(prefix):]) == "" {
		return Identity{}, ErrUnauthorized
	}
	claims, err := a.parseAndVerify(strings.TrimSpace(header[len(prefix):]))
	if err != nil {
		return Identity{}, ErrUnauthorized
	}
	if time.Now().UTC().After(claims.ExpiresAt) {
		return Identity{}, fmt.Errorf("%w: token expired", ErrUnauthorized)
	}
	if strings.TrimSpace(claims.ActorID) == "" || strings.TrimSpace(claims.ProfileID) == "" {
		return Identity{}, ErrUnauthorized
	}
	if len(claims.Scopes) == 0 {
		return Identity{}, ErrUnauthorized
	}
	for _, scope := range claims.Scopes {
		if _, ok := allowedScopes[scope]; !ok {
			return Identity{}, ErrUnauthorized
		}
	}
	a.mu.RLock()
	_, revoked := a.revoked[claims.TokenID]
	a.mu.RUnlock()
	if revoked {
		return Identity{}, fmt.Errorf("%w: token revoked", ErrUnauthorized)
	}
	return Identity{
		Actor:     publishing.Actor{Kind: publishing.ActorAgent, Identity: claims.ActorID},
		ProfileID: claims.ProfileID,
		Scopes:    append([]string(nil), claims.Scopes...),
		TokenID:   claims.TokenID,
	}, nil
}

// signClaims serializes and HMAC-signs claims as payload.signature.
func (a *Authenticator) signClaims(claims *Claims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("mcppublisher: encode claims: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(encoded))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + sig, nil
}

// parseAndVerify checks the token signature and returns its claims.
func (a *Authenticator) parseAndVerify(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, errors.New("mcppublisher: malformed token")
	}
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(parts[0]))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("mcppublisher: malformed token signature")
	}
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return nil, errors.New("mcppublisher: invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("mcppublisher: malformed token payload")
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, errors.New("mcppublisher: malformed token payload")
	}
	return &claims, nil
}

// pruneLocked drops revocation entries that have aged out. Callers must hold
// at least the read intent converted to a write lock; Revoke holds Lock.
func (a *Authenticator) pruneLocked() {
	now := time.Now().UTC()
	for tid, until := range a.revoked {
		if now.After(until) {
			delete(a.revoked, tid)
		}
	}
}
