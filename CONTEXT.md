# Domain Model & System Architecture: Regio Dani Pangestu Portfolio

## 1. Domain Vocabulary
- **Profile**: The core identity entity representing the developer (Regio Dani Pangestu - Full-Stack Engineer & AI Systems Architect).
- **Project**: A curated showcase item with title, slug, summary, architecture details, tags, live URL, and repository URL.
- **SkillCategory**: A grouped set of technical capabilities (Backend & Systems, Frontend Engineering, AI & Autonomous Agents, DevOps & Cloud).
- **ContactSubmission**: An incoming client or collaborator inquiry validated by the Go backend.
- **Design System ("Electric Sapphire")**: A cohesive blue-centric visual language:
  - Deep Base: `#080D1A` (rich deep void)
  - Surface Card: `#0F172A` / `#131F37`
  - Border Accents: `#1E293B` and active glow `#2563EB`
  - Primary Electric Blue: `#3B82F6` (accent), `#60A5FA` (hover), `#93C5FD` (light text)
  - Cyan Highlights: `#38BDF8`

## 2. System Seams & Testing Boundaries (Matt Pocock TDD Seams)
- **Seam A (Go Backend API)**:
  - `GET /api/profile` -> Returns structured profile info, social links, metrics.
  - `GET /api/projects` -> Returns list of portfolio projects, supports category filtering.
  - `GET /api/skills` -> Returns technical competence catalog.
  - `POST /api/contact` -> Validates email format, name, message body; returns 201 Created or 400 Bad Request.
  - `GET /api/health` -> System health probe.
- **Seam B (Go Static Embed / File Server)**:
  - Serves static SPA from Vite build distribution (`frontend/dist`) with fallback routing.
- **Seam C (Vite Frontend SPA)**:
  - React + TypeScript + Tailwind CSS client with modular component architecture.
  - Asynchronous data fetching against Go API with graceful fallback.
  - Fully responsive design matching high-end Pinterest portfolio references.
