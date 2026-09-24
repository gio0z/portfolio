package adminauth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"portfolio/internal/adminauth"
)

// mockGitHubServer sets up a fake GitHub OAuth & user API server for testing.
type mockGitHubServer struct {
	server       *httptest.Server
	usersByToken map[string]string // token -> github login
	expectedPKCE string
}

func newMockGitHubServer(usersByToken map[string]string) *mockGitHubServer {
	m := &mockGitHubServer{usersByToken: usersByToken}
	mux := http.NewServeMux()

	// Token exchange endpoint: POST /login/oauth/access_token
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		code := r.FormValue("code")
		codeVerifier := r.FormValue("code_verifier")

		// Verify PKCE if expected
		if m.expectedPKCE != "" && codeVerifier != m.expectedPKCE {
			http.Error(w, "invalid code_verifier", http.StatusBadRequest)
			return
		}

		token := "token_for_" + code
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": token,
			"token_type":   "bearer",
			"scope":        "read:user",
		})
	})

	// User info endpoint: GET /user
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		login, ok := m.usersByToken[token]
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"login": login,
			"id":    12345,
		})
	})

	m.server = httptest.NewServer(mux)
	return m
}

func (m *mockGitHubServer) Close() {
	m.server.Close()
}

func testConfig(mockGH *mockGitHubServer) adminauth.Config {
	return adminauth.Config{
		ClientID:         "test_client_id",
		ClientSecret:     "test_client_secret",
		SessionSecret:    "super_secret_session_signing_key_32bytes!!",
		AllowedLogin:     "gio0z",
		PublicOrigin:     "https://regiodani.dev",
		Environment:      "development",
		GitHubAuthURL:    mockGH.server.URL + "/login/oauth/authorize",
		GitHubTokenURL:   mockGH.server.URL + "/login/oauth/access_token",
		GitHubUserAPIURL: mockGH.server.URL + "/user",
		ApprovalVerifier: adminauth.NewDevApprovalVerifier("development"),
		SecureCookies:    true,
	}
}

func TestOAuth_AllowlistedUserAccepted(t *testing.T) {
	mockGH := newMockGitHubServer(map[string]string{
		"token_for_code_gio0z": "gio0z",
	})
	defer mockGH.Close()

	cfg := testConfig(mockGH)
	svc, err := adminauth.NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// 1. Initiate login
	loginReq := httptest.NewRequest(http.MethodGet, "/api/admin/auth/login", nil)
	loginRec := httptest.NewRecorder()
	svc.HandleLogin(loginRec, loginReq)

	if loginRec.Code != http.StatusFound {
		t.Fatalf("expected redirect (302), got %d", loginRec.Code)
	}

	loc := loginRec.Header().Get("Location")
	parsedURL, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("invalid redirect location: %v", err)
	}
	state := parsedURL.Query().Get("state")
	if state == "" {
		t.Fatalf("state parameter missing from redirect URL: %s", loc)
	}
	challenge := parsedURL.Query().Get("code_challenge")
	if challenge == "" {
		t.Fatalf("code_challenge parameter missing from redirect URL: %s", loc)
	}

	// Extract cookies from login response
	loginCookies := loginRec.Result().Cookies()
	var stateCookie *http.Cookie
	var pkceCookie *http.Cookie
	for _, c := range loginCookies {
		if c.Name == adminauth.OAuthStateCookieName {
			stateCookie = c
		}
		if c.Name == adminauth.OAuthPKCECookieName {
			pkceCookie = c
		}
	}
	if stateCookie == nil {
		t.Fatalf("expected oauth state cookie %q not found", adminauth.OAuthStateCookieName)
	}
	if pkceCookie == nil {
		t.Fatalf("expected oauth pkce cookie %q not found", adminauth.OAuthPKCECookieName)
	}

	// 2. Simulate callback with code for allowlisted user gio0z
	callbackReq := httptest.NewRequest(http.MethodGet, "/api/admin/auth/callback?code=code_gio0z&state="+state, nil)
	callbackReq.AddCookie(stateCookie)
	callbackReq.AddCookie(pkceCookie)
	callbackRec := httptest.NewRecorder()

	svc.HandleCallback(callbackRec, callbackReq)

	if callbackRec.Code != http.StatusFound && callbackRec.Code != http.StatusOK {
		t.Fatalf("expected successful callback status (302 or 200), got %d (body: %s)", callbackRec.Code, callbackRec.Body.String())
	}

	// Verify session cookie was set
	cookies := callbackRec.Result().Cookies()
	var sessionCookie *http.Cookie
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == adminauth.SessionCookieName {
			sessionCookie = c
		}
		if c.Name == adminauth.CSRFCookieName {
			csrfCookie = c
		}
	}

	if sessionCookie == nil {
		t.Fatalf("session cookie %q was not set", adminauth.SessionCookieName)
	}
	if sessionCookie.Value == "" {
		t.Fatalf("session cookie value is empty")
	}
	if !sessionCookie.HttpOnly {
		t.Errorf("session cookie must be HttpOnly")
	}
	if !sessionCookie.Secure {
		t.Errorf("session cookie must be Secure")
	}
	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("session cookie SameSite must be Strict, got %v", sessionCookie.SameSite)
	}
	if sessionCookie.Domain != "" {
		t.Errorf("session cookie Domain must be empty for host-only cookie, got %q", sessionCookie.Domain)
	}
	if csrfCookie == nil || csrfCookie.Value == "" {
		t.Fatalf("csrf cookie %q was not set or empty", adminauth.CSRFCookieName)
	}
}

func TestOAuth_NonAllowlistedUserRejected(t *testing.T) {
	mockGH := newMockGitHubServer(map[string]string{
		"token_for_code_attacker": "attacker123",
	})
	defer mockGH.Close()

	cfg := testConfig(mockGH)
	svc, err := adminauth.NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// 1. Initiate login
	loginReq := httptest.NewRequest(http.MethodGet, "/api/admin/auth/login", nil)
	loginRec := httptest.NewRecorder()
	svc.HandleLogin(loginRec, loginReq)

	loc := loginRec.Header().Get("Location")
	parsedURL, _ := url.Parse(loc)
	state := parsedURL.Query().Get("state")

	var stateCookie *http.Cookie
	var pkceCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == adminauth.OAuthStateCookieName {
			stateCookie = c
		}
		if c.Name == adminauth.OAuthPKCECookieName {
			pkceCookie = c
		}
	}

	// 2. Callback with non-allowlisted user
	callbackReq := httptest.NewRequest(http.MethodGet, "/api/admin/auth/callback?code=code_attacker&state="+state, nil)
	callbackReq.AddCookie(stateCookie)
	callbackReq.AddCookie(pkceCookie)
	callbackRec := httptest.NewRecorder()

	svc.HandleCallback(callbackRec, callbackReq)

	if callbackRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-allowlisted user, got %d", callbackRec.Code)
	}

	// Verify no session cookie set
	for _, c := range callbackRec.Result().Cookies() {
		if c.Name == adminauth.SessionCookieName && c.Value != "" && c.MaxAge >= 0 {
			t.Fatalf("session cookie should not be issued to non-allowlisted user")
		}
	}
}

func TestOAuth_PKCEAndStateValidation(t *testing.T) {
	mockGH := newMockGitHubServer(map[string]string{
		"token_for_code_gio0z": "gio0z",
	})
	defer mockGH.Close()

	cfg := testConfig(mockGH)
	svc, err := adminauth.NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// 1. Test missing state cookie
	req := httptest.NewRequest(http.MethodGet, "/api/admin/auth/callback?code=code_gio0z&state=fake_state", nil)
	rec := httptest.NewRecorder()
	svc.HandleCallback(rec, req)
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 400 or 401 for missing state cookie, got %d", rec.Code)
	}

	// 2. Test mismatched state
	loginReq := httptest.NewRequest(http.MethodGet, "/api/admin/auth/login", nil)
	loginRec := httptest.NewRecorder()
	svc.HandleLogin(loginRec, loginReq)

	var stateCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == adminauth.OAuthStateCookieName {
			stateCookie = c
			break
		}
	}

	mismatchReq := httptest.NewRequest(http.MethodGet, "/api/admin/auth/callback?code=code_gio0z&state=tampered_state", nil)
	mismatchReq.AddCookie(stateCookie)
	mismatchRec := httptest.NewRecorder()
	svc.HandleCallback(mismatchRec, mismatchReq)
	if mismatchRec.Code != http.StatusBadRequest && mismatchRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 400 or 401 for mismatched state, got %d", mismatchRec.Code)
	}
}

func TestSession_Expiration(t *testing.T) {
	mockGH := newMockGitHubServer(nil)
	defer mockGH.Close()

	cfg := testConfig(mockGH)
	cfg.SessionTTL = 50 * time.Millisecond // very short TTL for test
	svc, err := adminauth.NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Issue a session
	session, cookieValue, err := svc.CreateSession("gio0z")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	if session.Login != "gio0z" {
		t.Fatalf("expected login gio0z, got %s", session.Login)
	}

	protectedHandler := svc.OwnerMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// Immediate request: valid
	req := httptest.NewRequest(http.MethodGet, "/api/admin/overview", nil)
	req.AddCookie(&http.Cookie{Name: adminauth.SessionCookieName, Value: cookieValue})
	rec := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK before expiration, got %d", rec.Code)
	}

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	// Post-expiration request: 401 Unauthorized
	expReq := httptest.NewRequest(http.MethodGet, "/api/admin/overview", nil)
	expReq.AddCookie(&http.Cookie{Name: adminauth.SessionCookieName, Value: cookieValue})
	expRec := httptest.NewRecorder()
	protectedHandler.ServeHTTP(expRec, expReq)

	if expRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for expired session, got %d", expRec.Code)
	}
}

func TestCSRF_Validation(t *testing.T) {
	mockGH := newMockGitHubServer(nil)
	defer mockGH.Close()

	cfg := testConfig(mockGH)
	svc, err := adminauth.NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	session, cookieVal, err := svc.CreateSession("gio0z")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	csrfToken, err := svc.IssueCSRFToken(session.ID)
	if err != nil {
		t.Fatalf("failed to issue CSRF token: %v", err)
	}

	protectedHandler := svc.OwnerMiddleware(svc.CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("action performed"))
	})))

	// 1. Safe method (GET) without CSRF token: should succeed
	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/overview", nil)
	getReq.AddCookie(&http.Cookie{Name: adminauth.SessionCookieName, Value: cookieVal})
	getRec := httptest.NewRecorder()
	protectedHandler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET should not require CSRF token, got %d", getRec.Code)
	}

	// 2. State-changing method (POST) without CSRF token: should fail (403 Forbidden)
	postReqNoCSRF := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/123/reject", strings.NewReader(`{}`))
	postReqNoCSRF.AddCookie(&http.Cookie{Name: adminauth.SessionCookieName, Value: cookieVal})
	postRecNoCSRF := httptest.NewRecorder()
	protectedHandler.ServeHTTP(postRecNoCSRF, postReqNoCSRF)
	if postRecNoCSRF.Code != http.StatusForbidden {
		t.Fatalf("POST without CSRF token should return 403, got %d", postRecNoCSRF.Code)
	}

	// 3. POST with invalid CSRF token: should fail (403 Forbidden)
	postReqBadCSRF := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/123/reject", strings.NewReader(`{}`))
	postReqBadCSRF.AddCookie(&http.Cookie{Name: adminauth.SessionCookieName, Value: cookieVal})
	postReqBadCSRF.Header.Set("X-CSRF-Token", "wrong_token_value")
	postRecBadCSRF := httptest.NewRecorder()
	protectedHandler.ServeHTTP(postRecBadCSRF, postReqBadCSRF)
	if postRecBadCSRF.Code != http.StatusForbidden {
		t.Fatalf("POST with invalid CSRF token should return 403, got %d", postRecBadCSRF.Code)
	}

	// 4. POST with valid CSRF token: should succeed (200 OK)
	postReqGoodCSRF := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/123/reject", strings.NewReader(`{}`))
	postReqGoodCSRF.AddCookie(&http.Cookie{Name: adminauth.SessionCookieName, Value: cookieVal})
	postReqGoodCSRF.Header.Set("X-CSRF-Token", csrfToken)
	postRecGoodCSRF := httptest.NewRecorder()
	protectedHandler.ServeHTTP(postRecGoodCSRF, postReqGoodCSRF)
	if postRecGoodCSRF.Code != http.StatusOK {
		t.Fatalf("POST with valid CSRF token should return 200, got %d", postRecGoodCSRF.Code)
	}

	// 5. PUT and DELETE also require CSRF
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/api/admin/resource", nil)
		req.AddCookie(&http.Cookie{Name: adminauth.SessionCookieName, Value: cookieVal})
		rec := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s without CSRF token should return 403, got %d", method, rec.Code)
		}
	}
}

func TestStepUp_ApprovalVerification(t *testing.T) {
	mockGH := newMockGitHubServer(nil)
	defer mockGH.Close()

	// Dev verifier in development environment
	devVerifier := adminauth.NewDevApprovalVerifier("development")
	cfgDev := testConfig(mockGH)
	cfgDev.ApprovalVerifier = devVerifier
	svcDev, err := adminauth.NewService(cfgDev)
	if err != nil {
		t.Fatalf("failed to create dev service: %v", err)
	}

	session, cookieVal, _ := svcDev.CreateSession("gio0z")
	csrfToken, _ := svcDev.IssueCSRFToken(session.ID)

	approvalHandler := svcDev.OwnerMiddleware(svcDev.CSRFMiddleware(svcDev.RequireStepUp(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"approved"}`))
	}))))

	// 1. Missing step-up header/evidence: 403 Forbidden
	reqNoStepUp := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/123/approve-and-publish", strings.NewReader(`{}`))
	reqNoStepUp.AddCookie(&http.Cookie{Name: adminauth.SessionCookieName, Value: cookieVal})
	reqNoStepUp.Header.Set("X-CSRF-Token", csrfToken)
	recNoStepUp := httptest.NewRecorder()
	approvalHandler.ServeHTTP(recNoStepUp, reqNoStepUp)
	if recNoStepUp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for missing step-up, got %d", recNoStepUp.Code)
	}

	// 2. Valid dev step-up evidence in development mode: 200 OK
	reqWithStepUp := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/123/approve-and-publish", strings.NewReader(`{}`))
	reqWithStepUp.AddCookie(&http.Cookie{Name: adminauth.SessionCookieName, Value: cookieVal})
	reqWithStepUp.Header.Set("X-CSRF-Token", csrfToken)
	reqWithStepUp.Header.Set(adminauth.DevStepUpHeader, "true")
	recWithStepUp := httptest.NewRecorder()
	approvalHandler.ServeHTTP(recWithStepUp, reqWithStepUp)
	if recWithStepUp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with dev step-up header, got %d (body: %s)", recWithStepUp.Code, recWithStepUp.Body.String())
	}
}

func TestConfig_ProductionFailsClosedWithoutApprovalVerifier(t *testing.T) {
	mockGH := newMockGitHubServer(nil)
	defer mockGH.Close()

	// 1. Production with no approval verifier fails closed
	prodCfgNoVerifier := testConfig(mockGH)
	prodCfgNoVerifier.Environment = "production"
	prodCfgNoVerifier.ApprovalVerifier = nil
	_, err := adminauth.NewService(prodCfgNoVerifier)
	if err == nil {
		t.Fatalf("expected production startup to fail closed when approval verifier is nil")
	}

	// 2. Production with DevApprovalVerifier fails closed
	prodCfgDevVerifier := testConfig(mockGH)
	prodCfgDevVerifier.Environment = "production"
	prodCfgDevVerifier.ApprovalVerifier = adminauth.NewDevApprovalVerifier("production")
	_, err = adminauth.NewService(prodCfgDevVerifier)
	if err == nil {
		t.Fatalf("expected production startup to fail closed when dev approval verifier is used in production")
	}

	// 3. Production with valid production ApprovalVerifier succeeds
	prodVerifier := adminauth.NewPasskeyApprovalVerifier()
	prodCfgValid := testConfig(mockGH)
	prodCfgValid.Environment = "production"
	prodCfgValid.ApprovalVerifier = prodVerifier
	svcProd, err := adminauth.NewService(prodCfgValid)
	if err != nil {
		t.Fatalf("expected production startup with production ApprovalVerifier to succeed, got %v", err)
	}
	if svcProd == nil {
		t.Fatalf("service is nil")
	}
}

func TestConfig_MissingSecretsFailsClosed(t *testing.T) {
	// Missing ClientSecret
	cfg1 := adminauth.Config{
		ClientID:      "id",
		SessionSecret: "secret",
		AllowedLogin:  "gio0z",
	}
	if _, err := adminauth.NewService(cfg1); err == nil {
		t.Fatalf("expected error when ClientSecret is missing")
	}

	// Missing SessionSecret
	cfg2 := adminauth.Config{
		ClientID:     "id",
		ClientSecret: "secret",
		AllowedLogin: "gio0z",
	}
	if _, err := adminauth.NewService(cfg2); err == nil {
		t.Fatalf("expected error when SessionSecret is missing")
	}
}

func TestSession_StatusAndLogout(t *testing.T) {
	mockGH := newMockGitHubServer(nil)
	defer mockGH.Close()

	cfg := testConfig(mockGH)
	svc, err := adminauth.NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	// 1. Session check when unauthenticated returns 401
	unauthReq := httptest.NewRequest(http.MethodGet, "/api/admin/auth/session", nil)
	unauthRec := httptest.NewRecorder()
	mux.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated session check, got %d", unauthRec.Code)
	}

	// 2. Issue session and check status -> returns 200 with login
	session, cookieVal, err := svc.CreateSession("gio0z")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	authReq := httptest.NewRequest(http.MethodGet, "/api/admin/auth/session", nil)
	authReq.AddCookie(&http.Cookie{Name: adminauth.SessionCookieName, Value: cookieVal})
	authRec := httptest.NewRecorder()
	mux.ServeHTTP(authRec, authReq)
	if authRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for authenticated session check, got %d", authRec.Code)
	}

	var sessionResp struct {
		Authenticated bool   `json:"authenticated"`
		Login         string `json:"login"`
	}
	if err := json.NewDecoder(authRec.Body).Decode(&sessionResp); err != nil {
		t.Fatalf("failed to decode session response: %v", err)
	}
	if !sessionResp.Authenticated || sessionResp.Login != "gio0z" {
		t.Fatalf("unexpected session response: %+v", sessionResp)
	}

	// 3. Logout clears session and CSRF cookies
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/admin/auth/logout", nil)
	logoutReq.AddCookie(&http.Cookie{Name: adminauth.SessionCookieName, Value: cookieVal})
	logoutRec := httptest.NewRecorder()
	mux.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for logout, got %d", logoutRec.Code)
	}

	// Verify session cookie was cleared
	var clearedSessionCookie, clearedCSRFCookie *http.Cookie
	for _, c := range logoutRec.Result().Cookies() {
		if c.Name == adminauth.SessionCookieName {
			clearedSessionCookie = c
		}
		if c.Name == adminauth.CSRFCookieName {
			clearedCSRFCookie = c
		}
	}
	if clearedSessionCookie == nil || clearedSessionCookie.MaxAge >= 0 {
		t.Fatalf("expected cleared session cookie with negative max age")
	}
	if clearedCSRFCookie == nil || clearedCSRFCookie.MaxAge >= 0 {
		t.Fatalf("expected cleared csrf cookie with negative max age")
	}

	// 4. Session check after logout returns 401
	postLogoutReq := httptest.NewRequest(http.MethodGet, "/api/admin/auth/session", nil)
	postLogoutReq.AddCookie(&http.Cookie{Name: adminauth.SessionCookieName, Value: cookieVal})
	postLogoutRec := httptest.NewRecorder()
	mux.ServeHTTP(postLogoutRec, postLogoutReq)
	if postLogoutRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for post-logout session check, got %d", postLogoutRec.Code)
	}
	_ = session
}
