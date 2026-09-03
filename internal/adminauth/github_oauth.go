package adminauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	OAuthStateCookieName = "admin_oauth_state"
	OAuthPKCECookieName  = "admin_oauth_pkce"
	OAuthCookieTTL       = 10 * time.Minute
)

var (
	ErrOAuthStateMismatch = errors.New("oauth state mismatch")
	ErrOAuthCodeMissing   = errors.New("oauth code missing")
	ErrUnauthorizedUser   = errors.New("github user not authorized")
)

// GeneratePKCE creates a code verifier and code challenge (S256).
func GeneratePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}

// HandleLogin initiates the GitHub OAuth flow with state and PKCE.
func (s *Service) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := GenerateRandomString(16)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Store state and verifier in secure cookies
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthStateCookieName,
		Value:    state,
		Path:     "/",
		Expires:  time.Now().Add(OAuthCookieTTL),
		MaxAge:   int(OAuthCookieTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode, // Lax for redirect flow
	})

	http.SetCookie(w, &http.Cookie{
		Name:     OAuthPKCECookieName,
		Value:    verifier,
		Path:     "/",
		Expires:  time.Now().Add(OAuthCookieTTL),
		MaxAge:   int(OAuthCookieTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	redirectURI := s.getCallbackURL()
	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&scope=read:user&state=%s&code_challenge=%s&code_challenge_method=S256",
		s.cfg.GitHubAuthURL,
		url.QueryEscape(s.cfg.ClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
		url.QueryEscape(challenge),
	)

	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *Service) getCallbackURL() string {
	origin := strings.TrimRight(s.cfg.PublicOrigin, "/")
	return fmt.Sprintf("%s/api/admin/auth/callback", origin)
}

// HandleCallback processes the GitHub OAuth callback.
func (s *Service) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		http.Error(w, "invalid oauth callback parameters", http.StatusBadRequest)
		return
	}

	stateCookie, err := r.Cookie(OAuthStateCookieName)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != state {
		http.Error(w, "invalid or missing oauth state", http.StatusBadRequest)
		return
	}

	pkceCookie, err := r.Cookie(OAuthPKCECookieName)
	if err != nil || pkceCookie.Value == "" {
		http.Error(w, "invalid or missing pkce verifier", http.StatusBadRequest)
		return
	}
	codeVerifier := pkceCookie.Value

	// Clear temporary OAuth cookies
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthStateCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthPKCECookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	// Exchange code for access token with PKCE code_verifier
	accessToken, err := s.exchangeCode(ctx, code, codeVerifier)
	if err != nil {
		http.Error(w, "oauth token exchange failed", http.StatusBadRequest)
		return
	}

	// Fetch GitHub user identity
	login, err := s.fetchGitHubUser(ctx, accessToken)
	if err != nil {
		http.Error(w, "failed to retrieve user identity", http.StatusBadRequest)
		return
	}

	// Allowlist verification: strictly enforce allowed login
	if !strings.EqualFold(login, s.cfg.AllowedLogin) {
		http.Error(w, "user not authorized for admin access", http.StatusForbidden)
		return
	}

	// Create new session
	sess, signedToken, err := s.CreateSession(login)
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	SetSessionCookie(w, signedToken, sess.ExpiresAt, s.cfg.SecureCookies)

	// Issue and set CSRF cookie
	csrfToken, err := s.IssueCSRFToken(sess.ID)
	if err == nil {
		SetCSRFCookie(w, csrfToken, sess.ExpiresAt, s.cfg.SecureCookies)
	}

	// Redirect to admin UI
	adminHome := "/admin"
	if s.cfg.PublicOrigin != "" {
		adminHome = strings.TrimRight(s.cfg.PublicOrigin, "/") + "/admin"
	}
	http.Redirect(w, r, adminHome, http.StatusFound)
}

func (s *Service) exchangeCode(ctx context.Context, code, codeVerifier string) (string, error) {
	data := url.Values{}
	data.Set("client_id", s.cfg.ClientID)
	data.Set("client_secret", s.cfg.ClientSecret)
	data.Set("code", code)
	data.Set("code_verifier", codeVerifier)
	data.Set("redirect_uri", s.getCallbackURL())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.GitHubTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("token exchange returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("oauth error: %s (%s)", tokenResp.Error, tokenResp.ErrorDesc)
	}
	if tokenResp.AccessToken == "" {
		return "", errors.New("no access token in response")
	}

	return tokenResp.AccessToken, nil
}

func (s *Service) fetchGitHubUser(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.GitHubUserAPIURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("user endpoint returned %d", resp.StatusCode)
	}

	var userResp struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return "", err
	}
	if userResp.Login == "" {
		return "", errors.New("empty login in user response")
	}
	return userResp.Login, nil
}
