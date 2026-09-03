package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	mux         *http.ServeMux
	submissions []ContactSubmission
	mu          sync.RWMutex
}

func NewServer() *Server {
	s := &Server{
		mux:         http.NewServeMux(),
		submissions: make([]ContactSubmission, 0),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS headers for Vite frontend integration
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

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
		Tagline:  "Architecting Resilient Backend Engines & Autonomous AI Systems",
		Title:    "Senior Full-Stack Engineer & AI Systems Architect",
		Bio:      "Passionate software engineer specializing in high-concurrency Go microservices, reactive modern web architectures (Vite/React/TypeScript), and autonomous multi-agent systems with Hermes Agent & LLM orchestration.",
		Location: "Indonesia",
		Status:   "Available for High-Impact Projects",
		Email:    "regio@zoo.com",
		Phone:    "+62 851-5643-9303",
		Avatar:   "https://images.unsplash.com/photo-1534528741775-53994a69daeb?auto=format&fit=crop&w=600&q=80",
		Stats: []StatMetric{
			{Label: "Experience", Value: "5+ Years", Sub: "Full-Stack & Systems"},
			{Label: "Shipped Projects", Value: "45+", Sub: "Production Grade"},
			{Label: "Uptime & Quality", Value: "99.98%", Sub: "Zero-Downtime Releases"},
			{Label: "Code Coverage", Value: "94%+", Sub: "Disciplined TDD"},
		},
		SocialLinks: map[string]string{
			"github":   "https://github.com/gio0z",
			"linkedin": "https://linkedin.com/in/regiodani",
			"telegram": "https://t.me/Ingouk_bot",
			"whatsapp": "https://wa.me/6285156439303",
		},
		Highlights: []string{
			"Engineered distributed AI agent pipelines capable of multi-channel relay & autonomous execution",
			"Core advocate of deep modular design, strict test-first development, and clean architecture",
			"Proven track record building enterprise government booking systems, UMKM POS, and real-time gateways",
		},
	}
	jsonResponse(w, http.StatusOK, profile)
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	allProjects := []Project{
		{
			ID:          "jam-nguar",
			Title:       "Jam Nguar: Government Room Booking System",
			Tagline:     "4-tier RBAC room reservation & management engine for Blitar Regency ASN/PNS.",
			Description: "High-integrity municipal reservation platform handling strict booking deadlines, VIP room gating, automatic conflicts resolution, and cryptographic QR code check-in.",
			Category:    "Full-Stack",
			Tags:        []string{"Rust", "Actix-Web", "Next.js", "PostgreSQL", "Tailwind CSS"},
			Featured:    true,
			GithubURL:   "https://github.com/gio0z/jam-nguar",
			DemoURL:     "https://jam-nguar.blitar.go.id",
			Image:       "https://images.unsplash.com/photo-1497366216548-37526070297c?auto=format&fit=crop&w=1200&q=80",
			Metrics:     "15k+ Monthly Bookings, Zero Double-Booking Guarantee",
		},
		{
			ID:          "pos-kala",
			Title:       "POS Kala: Unified UMKM Retail Engine",
			Tagline:     "Point of sale, multi-warehouse inventory, and financial ledger platform for Indonesian SMEs.",
			Description: "Full-featured retail operations system with offline-first transaction sync, barcode scanning, thermal receipt printing, and comprehensive margin analysis.",
			Category:    "Full-Stack",
			Tags:        []string{"Go", "Vite", "React", "SQLite / Postgres", "Tailwind"},
			Featured:    true,
			GithubURL:   "https://github.com/gio0z/pos-kala",
			DemoURL:     "https://poskala.id",
			Image:       "https://images.unsplash.com/photo-1556742049-0a67e557224f?auto=format&fit=crop&w=1200&q=80",
			Metrics:     "Processed >$500k in GMV with sub-50ms checkout latency",
		},
		{
			ID:          "hermes-agent-systems",
			Title:       "Hermes Multi-Platform Gateway & Agent Mesh",
			Tagline:     "Enterprise autonomous AI agent orchestration with multi-channel routing.",
			Description: "Architected bidirectional communication pipelines across WhatsApp Baileys, Telegram, Discord, and Slack with localized sandboxing, rate limiting, and long-term memory integration.",
			Category:    "AI & Agents",
			Tags:        []string{"Python", "Go", "Docker", "Node.js", "Hermes Agent", "Hindsight"},
			Featured:    true,
			GithubURL:   "https://github.com/gio0z/hermes-cs",
			DemoURL:     "https://hermes-agent.nousresearch.com",
			Image:       "https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?auto=format&fit=crop&w=1200&q=80",
			Metrics:     "Active 24/7 autonomous triage, 10,000+ daily events handled",
		},
		{
			ID:          "thundercrawl",
			Title:       "ThunderCrawl: Distributed Web Scraping & Indexer",
			Tagline:     "Ultra-fast asynchronous crawler with headless stealth browser cluster.",
			Description: "High-throughput content extraction engine designed for AI data ingestion, bypassing bot detections, parsing structured Markdown, and indexing knowledge graphs.",
			Category:    "Systems",
			Tags:        []string{"Go", "Chromium", "Redis", "ElasticSearch", "Vite"},
			Featured:    false,
			GithubURL:   "https://github.com/gio0z/thundercrawl",
			DemoURL:     "https://thundercrawl.dev",
			Image:       "https://images.unsplash.com/photo-1558494949-ef010cbdcc31?auto=format&fit=crop&w=1200&q=80",
			Metrics:     "2,500 pages/minute crawling speed with <0.1% block rate",
		},
		{
			ID:          "open-design-studio",
			Title:       "Open Design Visual Artifact Engine",
			Tagline:     "Live design-system generative renderer for agentic design systems.",
			Description: "Integrated with Open Design MCP daemon to build dynamic design systems, landing pages, and interactive prototypes directly from conversational agent instructions.",
			Category:    "Frontend",
			Tags:        []string{"TypeScript", "Vite", "Tailwind CSS", "MCP", "Canvas"},
			Featured:    true,
			GithubURL:   "https://github.com/gio0z/open-design",
			DemoURL:     "https://opendesign.studio",
			Image:       "https://images.unsplash.com/photo-1507238691740-187a5b1d37b8?auto=format&fit=crop&w=1200&q=80",
			Metrics:     "Sub-second live preview compilation across 50+ component variants",
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
