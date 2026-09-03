package adminauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	SessionCookieName    = "admin_session"
	DefaultSessionTTL    = 24 * time.Hour
)

var (
	ErrInvalidSessionToken = errors.New("invalid session token")
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionExpired      = errors.New("session expired")
)

// Session represents an authenticated admin session.
type Session struct {
	ID        string    `json:"id"`
	Login     string    `json:"login"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionStore manages active server-side sessions.
type SessionStore interface {
	Save(session *Session) error
	Get(id string) (*Session, error)
	Delete(id string) error
}

// MemorySessionStore is a thread-safe in-memory session store.
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemorySessionStore initializes an in-memory session store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

func (s *MemorySessionStore) Save(sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.ID] = sess
	return nil
}

func (s *MemorySessionStore) Get(id string) (*Session, error) {
	s.mu.RLock()
	sess, exists := s.sessions[id]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrSessionNotFound
	}

	if time.Now().After(sess.ExpiresAt) {
		s.mu.Lock()
		delete(s.sessions, id)
		s.mu.Unlock()
		return nil, ErrSessionExpired
	}

	return sess, nil
}

func (s *MemorySessionStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

// GenerateRandomString produces cryptographically secure hex-encoded random bytes.
func GenerateRandomString(numBytes int) (string, error) {
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// signSessionToken signs a session ID with HMAC-SHA256: <session_id>.<signature>
func signSessionToken(sessionID string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(sessionID))
	sig := hex.EncodeToString(mac.Sum(nil))
	return sessionID + "." + sig
}

// verifySessionToken verifies signature and returns the session ID.
func verifySessionToken(token string, secret []byte) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", ErrInvalidSessionToken
	}
	sessionID, sig := parts[0], parts[1]

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(sessionID))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(sig), []byte(expectedSig)) != 1 {
		return "", ErrInvalidSessionToken
	}
	return sessionID, nil
}

// SetSessionCookie sets a secure, host-only cookie for the admin session.
func SetSessionCookie(w http.ResponseWriter, signedToken string, expiresAt time.Time, secure bool) {
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    signedToken,
		Path:     "/",
		Domain:   "", // empty Domain ensures host-only cookie
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
}

// ClearSessionCookie clears the admin session cookie.
func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
}
