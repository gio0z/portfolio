package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"portfolio/internal/adminapi"
	"portfolio/internal/adminauth"
	"portfolio/internal/publicapi"
	"portfolio/internal/publishing"
)

type ServerOption func(*Server)

type Server struct {
	mux           *http.ServeMux
	submissions   []ContactSubmission
	mu            sync.RWMutex
	publicHandler *publicapi.Handler
	adminHandler  *adminapi.Handler
	adminAuth     *adminauth.Service
	adminSubMux   http.Handler
}

// WithRepository configures the server with a publishing repository for public lab endpoints.
func WithRepository(repo publishing.Repository) ServerOption {
	return func(s *Server) {
		s.publicHandler = publicapi.NewHandlerWithRepo(repo)
	}
}

// WithPublicAPIReader configures the server with a publicapi.ProjectReader.
func WithPublicAPIReader(reader publicapi.ProjectReader) ServerOption {
	return func(s *Server) {
		s.publicHandler = publicapi.NewHandler(reader)
	}
}

// WithAdminAPI configures the server with an admin API handler and authentication service.
func WithAdminAPI(handler *adminapi.Handler, auth *adminauth.Service) ServerOption {
	return func(s *Server) {
		s.adminHandler = handler
		s.adminAuth = auth
		if handler != nil {
			s.adminSubMux = handler.Routes(auth)
		}
	}
}

// WithPublishing configures the server with full public and admin publishing services.
func WithPublishing(service publishing.PublishingService, repo publishing.Repository, auth *adminauth.Service) ServerOption {
	return func(s *Server) {
		s.publicHandler = publicapi.NewHandlerWithRepo(repo)
		s.adminHandler = adminapi.NewHandler(service, repo)
		s.adminAuth = auth
		if s.adminHandler != nil {
			s.adminSubMux = s.adminHandler.Routes(auth)
		}
	}
}

// SetAdminAPI dynamically updates the administrative API handler and authentication service.
func (s *Server) SetAdminAPI(handler *adminapi.Handler, auth *adminauth.Service) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.adminHandler = handler
	s.adminAuth = auth
	if handler != nil {
		s.adminSubMux = handler.Routes(auth)
	} else {
		s.adminSubMux = nil
	}
}

// SetRepository dynamically updates the publishing repository backing public lab endpoints.
func (s *Server) SetRepository(repo publishing.Repository) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publicHandler = publicapi.NewHandlerWithRepo(repo)
}

// SetPublicAPIReader dynamically updates the project reader backing public lab endpoints.
func (s *Server) SetPublicAPIReader(reader publicapi.ProjectReader) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publicHandler = publicapi.NewHandler(reader)
}

func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		mux:         http.NewServeMux(),
		submissions: make([]ContactSubmission, 0),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.publicHandler == nil {
		s.publicHandler = publicapi.NewHandler(nil)
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS headers for Vite frontend integration
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Request-ID, X-CSRF-Token, CSRF-Token, Idempotency-Key, X-Admin-StepUp-Dev, X-Admin-Passkey-Assertion")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/profile", s.handleProfile)
	s.mux.HandleFunc("/api/projects", s.handleProjects)
	s.mux.HandleFunc("/api/skills", s.handleSkills)
	s.mux.HandleFunc("/api/contact", s.handleContact)
	s.mux.HandleFunc("/api/contact/whatsapp", s.handleContactWhatsApp)

	// Public Lab & Design Lab catalog endpoints
	s.mux.HandleFunc("/api/lab/projects", func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		h := s.publicHandler
		s.mu.RUnlock()
		if h != nil {
			h.HandleListProjects(w, r)
		} else {
			publicapi.NewHandler(nil).HandleListProjects(w, r)
		}
	})
	s.mux.HandleFunc("/api/lab/projects/", func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		h := s.publicHandler
		s.mu.RUnlock()
		if h != nil {
			h.HandleProjectSubpath(w, r)
		} else {
			publicapi.NewHandler(nil).HandleProjectSubpath(w, r)
		}
	})
	s.mux.HandleFunc("/api/portfolio/design-lab", func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		h := s.publicHandler
		s.mu.RUnlock()
		if h != nil {
			h.HandlePortfolioDesignLab(w, r)
		} else {
			publicapi.NewHandler(nil).HandlePortfolioDesignLab(w, r)
		}
	})

	// Admin Review, Publishing, and Audit endpoints
	adminDelegate := func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		subMux := s.adminSubMux
		s.mu.RUnlock()

		if subMux == nil {
			http.Error(w, "unauthorized: admin service not configured", http.StatusUnauthorized)
			return
		}
		subMux.ServeHTTP(w, r)
	}

	s.mux.HandleFunc("/api/admin/overview", adminDelegate)
	s.mux.HandleFunc("/api/admin/reviews", adminDelegate)
	s.mux.HandleFunc("/api/admin/reviews/", adminDelegate)
	s.mux.HandleFunc("/api/admin/projects/", adminDelegate)
	s.mux.HandleFunc("/api/admin/audit", adminDelegate)
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"service":   "regio-portfolio-api",
		"framework": "go/http",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	profile := PortfolioContent().Profile
	jsonResponse(w, http.StatusOK, profile)
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	allProjects := PortfolioContent().Projects

	if category == "" || category == "All" {
		jsonResponse(w, http.StatusOK, allProjects)
		return
	}

	filtered := make([]Project, 0)
	for _, p := range allProjects {
		if p.Category == category {
			filtered = append(filtered, p)
		}
	}
	jsonResponse(w, http.StatusOK, filtered)
}

func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	skills := PortfolioContent().SkillCategories
	jsonResponse(w, http.StatusOK, skills)
}

func (s *Server) handleContact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	var req ContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid request body JSON",
		})
		return
	}

	if err := req.Validate(); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	s.mu.Lock()
	submission := ContactSubmission{
		ID:        fmt.Sprintf("sub_%d", time.Now().UnixNano()),
		Contact:   req,
		CreatedAt: time.Now().UTC(),
	}
	s.submissions = append(s.submissions, submission)
	s.mu.Unlock()

	jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Thank you, %s! Your message has been received. Regio will respond promptly.", req.Name),
		"id":      submission.ID,
	})
}

// handleContactWhatsApp exposes the owner's WhatsApp contact channel without
// ever embedding the number in the frontend bundle. The number lives only in
// the CONTACT_WHATSAPP environment variable on the server; the response
// carries a wa.me link built from it. The number is deliberately never
// logged and never appears in any other response.
func (s *Server) handleContactWhatsApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	digits := onlyDigits(os.Getenv("CONTACT_WHATSAPP"))
	if digits == "" {
		jsonResponse(w, http.StatusNotFound, map[string]string{
			"error": "contact channel not configured",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"url": "https://wa.me/" + digits,
	})
}

// onlyDigits strips formatting characters (spaces, dashes, a leading +) so the
// link is valid whether the operator stored the number as "+62 851-5643-9303"
// or as bare digits.
func onlyDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
