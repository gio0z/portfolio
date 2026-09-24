package api

// Content is every piece of portfolio material the public site renders: the
// profile, the project list, and the skill catalogue.
//
// This is the single authored copy. The public handlers serve it, and the
// static build derives its content file from it via tools/contentexport, so the
// prerendered pages and the live API cannot drift apart.
// The field tags define the JSON contract of the generated frontend content
// file. No endpoint serialises Content directly, so these tags affect only
// tools/contentexport and the build-time file it writes.
type Content struct {
	Profile         Profile         `json:"profile"`
	Projects        []Project       `json:"projects"`
	SkillCategories []SkillCategory `json:"skill_categories"`
}

// PortfolioContent returns the authored portfolio content.
func PortfolioContent() Content {
	return Content{
		Profile:         portfolioProfile,
		Projects:        portfolioProjects,
		SkillCategories: portfolioSkillCategories,
	}
}

// portfolioProfile backs the public profile endpoint and the /about page.
var portfolioProfile = Profile{
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

// portfolioProjects backs the public project list, the /work index, and every
// /work/<id> case study. Project IDs are URL-safe and act as the route segment.
var portfolioProjects = []Project{
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

// portfolioSkillCategories backs the public skills endpoint and the /services
// page, where each category is presented as a capability rather than an offer.
var portfolioSkillCategories = []SkillCategory{
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
