package adminauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"time"
)

const (
	CSRFHeaderName    = "X-CSRF-Token"
	CSRFHeaderAltName = "X-XSRF-Token"
	CSRFCookieName    = "admin_csrf"
)

// IssueCSRFToken derives a cryptographically bound CSRF token for a given session.
func (s *Service) IssueCSRFToken(sessionID string) (string, error) {
	mac := hmac.New(sha256.New, []byte(s.cfg.SessionSecret))
	mac.Write([]byte("csrf:" + sessionID))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// ValidateCSRFToken checks if the provided token matches the expected token for the session in constant time.
func (s *Service) ValidateCSRFToken(sessionID, token string) bool {
	if token == "" || sessionID == "" {
		return false
	}
	expected, err := s.IssueCSRFToken(sessionID)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

// IsStateChangingMethod returns true for HTTP methods that alter state.
func IsStateChangingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// SetCSRFCookie sets the CSRF token cookie readable by JavaScript to attach to headers.
func SetCSRFCookie(w http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	cookie := &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		Domain:   "", // host-only
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: false, // Accessible by frontend JS
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
}

// ClearCSRFCookie clears the CSRF token cookie.
func ClearCSRFCookie(w http.ResponseWriter, secure bool) {
	cookie := &http.Cookie{
		Name:     CSRFCookieName,
		Value:    "",
		Path:     "/",
		Domain:   "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
}
