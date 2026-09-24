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
	profile := Profile{
		Name:     "Regio Dani Pangestu",
		Tagline:  "Software that keeps working after launch",
		Title:    "Full-Stack Engineer",
		Bio:      "I design and build the systems small and mid-sized businesses run on — ordering and inventory, bookings, customer support, and the automation in between. One engineer from first conversation to production, so nothing gets lost in handover.",
		Location: "Indonesia",
		Status:   "Available for new projects",
		Email:    "",
		Phone:    "",
		Avatar:   "https://images.unsplash.com/photo-1534528741775-53994a69daeb?auto=format&fit=crop&w=600&q=80",
		Stats:    GeneratedStats,
		SocialLinks: map[string]string{
			"github":   "https://github.com/gio0z",
			"linkedin": "https://linkedin.com/in/regiodani",
			"telegram": "https://t.me/Ingouk_bot",
		},
		Highlights: []string{
			"Every release ships with a rollback plan",
			"You own the code, the data, and the infrastructure — no lock-in",
		},
	}
	jsonResponse(w, http.StatusOK, profile)
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	allProjects := []Project{
		{
			ID:          "jam-nguar",
			Title:       "Jam Nguar",
			Tagline:     "Room booking for Blitar Regency government staff.",
			Description: "Staff book meeting rooms against real availability, with booking deadlines, VIP room rules, and conflicts caught before two people can claim the same slot. A QR code at the door confirms who actually turned up.",
			Category:    "Government",
			Tags:        []string{"Rust", "Actix-Web", "Next.js", "PostgreSQL"},
			Featured:    true,
			GithubURL:   "https://github.com/gio0z/jam-nguar",
			DemoURL:     "https://github.com/gio0z/jam-nguar",
			Image:       "https://images.unsplash.com/photo-1497366216548-37526070297c?auto=format&fit=crop&w=1200&q=80",
			Metrics:     "Handles booking conflicts so two people can never hold the same room",
		},
		{
			ID:          "inven-kab-blitar",
			Title:       "Inven Kabupaten Blitar",
			Tagline:     "Inventory and ledger for regional government stock.",
			Description: "Regional stock is reconciled against what actually moved, so the books match the shelves. Corrections are recorded instead of overwritten, and a negative balance shows up the moment it happens rather than at year end.",
			Category:    "Government",
			Tags:        []string{"PHP", "CodeIgniter", "MySQL", "Docker"},
			Featured:    false,
			GithubURL:   "https://github.com/gio0z/inven-kab-blitar",
			DemoURL:     "https://github.com/gio0z/inven-kab-blitar",
			Image:       "https://images.unsplash.com/photo-1556742049-0a67e557224f?auto=format&fit=crop&w=1200&q=80",
			Metrics:     "Every stock correction is written to an audit trail",
		},
		{
			ID:          "labtu-web",
			Title:       "Labtu",
			Tagline:     "Task and milestone tracking for government teams.",
			Description: "Daily checklists, duty catalogues, and recaps for a government division, so work in progress is visible without chasing anyone for a status update.",
			Category:    "Government",
			Tags:        []string{"Go", "Vite/React", "MySQL", "Docker"},
			Featured:    true,
			GithubURL:   "https://github.com/gio0z/labtu-web",
			DemoURL:     "https://github.com/gio0z/labtu-web",
			Image:       "https://images.unsplash.com/photo-1531403009284-440f080d1e12?auto=format&fit=crop&w=1200&q=80",
			Metrics:     "Replaced spreadsheet tracking for a whole division",
		},
		{
			ID:          "cs-portal",
			Title:       "CS Portal",
			Tagline:     "Multi-tenant customer service automation.",
			Description: "Customer support runs over WhatsApp: routine questions are answered from the business's own knowledge base, staff take over any conversation that needs a person, and plans are billed from the same place.",
			Category:    "AI & Agents",
			Tags:        []string{"Go", "PostgreSQL", "Docker", "WhatsApp", "Midtrans"},
			Featured:    true,
			GithubURL:   "https://github.com/gio0z/cs-portal",
			DemoURL:     "https://github.com/gio0z/cs-portal",
			Image:       "https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?auto=format&fit=crop&w=1200&q=80",
			Metrics:     "Support keeps running when staff are offline",
		},
		{
			ID:          "tour-travel-web",
			Title:       "Afsa Tour & Transport",
			Tagline:     "Tour packages and transport rental, Blitar.",
			Description: "A catalogue of tour packages and rental vehicles, with enquiries that reach the owner directly instead of piling up in a mailbox.",
			Category:    "Travel",
			Tags:        []string{"Astro", "TypeScript", "Express", "Docker"},
			Featured:    true,
			GithubURL:   "https://github.com/gio0z/tour-travel-web",
			DemoURL:     "https://github.com/gio0z/tour-travel-web",
			Image:       "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?auto=format&fit=crop&w=1200&q=80",
			Metrics:     "Enquiries land in WhatsApp instead of an inbox nobody reads",
		},
		{
			ID:          "nusantara-botanica",
			Title:       "Plantea",
			Tagline:     "Botanical export storefront.",
			Description: "A storefront for rare plants built to serve fast: pages are generated ahead of time rather than assembled on every visit, so a catalogue page opens without waiting on a server.",
			Category:    "Commerce",
			Tags:        []string{"Astro", "TypeScript", "Tailwind", "Vercel"},
			Featured:    true,
			GithubURL:   "https://github.com/gio0z/nusantara-botanica",
			DemoURL:     "https://github.com/gio0z/nusantara-botanica",
			Image:       "https://images.unsplash.com/photo-1558494949-ef010cbdcc31?auto=format&fit=crop&w=1200&q=80",
			Metrics:     "Pages load without a backend round-trip",
		},
	}

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
	skills := []SkillCategory{
		{
			Category: "Backend & Systems",
			Summary:  "High-performance concurrency, robust APIs, and clean domain design",
			Skills: []SkillItem{
				{Name: "Go (Golang)", Level: 95, Proficiency: "Expert", Icon: "Cpu", Description: "Goroutines, channels, microservices, net/http, standard library mastery"},
				{Name: "Rust", Level: 85, Proficiency: "Advanced", Icon: "Shield", Description: "Memory safety, zero-cost abstractions, Actix-Web, CLI tooling"},
				{Name: "PostgreSQL / SQLite", Level: 90, Proficiency: "Expert", Icon: "Database", Description: "Indexing, query optimization, migration management, schema design"},
				{Name: "RESTful & gRPC APIs", Level: 94, Proficiency: "Expert", Icon: "Network", Description: "Strict contract design, idempotency, rate limiting, OpenAPI"},
			},
		},
		{
			Category: "Frontend Engineering",
			Summary:  "Fluid, responsive, accessible, and reactive user interfaces",
			Skills: []SkillItem{
				{Name: "Vite Ecosystem", Level: 95, Proficiency: "Expert", Icon: "Zap", Description: "HMR, optimized roll-up bundling, plugin architecture"},
				{Name: "React & TypeScript", Level: 94, Proficiency: "Expert", Icon: "Code2", Description: "Custom hooks, state management, strict type checking, performance"},
				{Name: "Tailwind CSS", Level: 96, Proficiency: "Expert", Icon: "Palette", Description: "Custom design systems, responsive grids, dark/blue theme styling"},
				{Name: "UI/UX & Pinterest Aesthetics", Level: 90, Proficiency: "Advanced", Icon: "Layout", Description: "Modern glassmorphism, micro-interactions, clean typography"},
			},
		},
		{
			Category: "AI & Autonomous Agents",
			Summary:  "Agentic coding workflows, LLM orchestration, and multi-agent mesh",
			Skills: []SkillItem{
				{Name: "Hermes Agent Framework", Level: 95, Proficiency: "Expert", Icon: "Bot", Description: "Profiles, skills authoring, multi-platform gateway orchestration"},
				{Name: "Superpowers & Matt Pocock Flow", Level: 92, Proficiency: "Expert", Icon: "Sparkles", Description: "Disciplined TDD, grilling, spec-driven development, deep modules"},
				{Name: "MCP (Model Context Protocol)", Level: 90, Proficiency: "Expert", Icon: "Layers", Description: "Designing and integrating custom MCP tools and servers"},
				{Name: "Long-Term Memory Systems", Level: 88, Proficiency: "Advanced", Icon: "Brain", Description: "Hindsight integration, semantic graphs, entity retrieval"},
			},
		},
		{
			Category: "DevOps & Infrastructure",
			Summary:  "Reliable continuous delivery, sandboxing, and Linux environments",
			Skills: []SkillItem{
				{Name: "Docker & Containerization", Level: 90, Proficiency: "Advanced", Icon: "Container", Description: "Multi-stage builds, rootless containers, compose clusters"},
				{Name: "Linux & WSL Administration", Level: 92, Proficiency: "Expert", Icon: "Terminal", Description: "Shell automation, systemd services, process monitoring"},
				{Name: "CI/CD & Git Workflows", Level: 92, Proficiency: "Expert", Icon: "GitBranch", Description: "GitHub Actions, automated test suites, release tagging"},
			},
		},
	}
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
