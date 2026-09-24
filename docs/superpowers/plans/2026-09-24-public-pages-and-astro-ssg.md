# Public Pages Expansion + Astro SSG Migration — Implementation Plan

**Date:** 2026-09-24
**Status:** Awaiting approval before implementation
**Supersedes (partially):** the Vite-only frontend decision in `CONTEXT.md` §2 Seam C

> I'm using the writing-plans skill to create this implementation plan.

---

## 1. Goal

Ship five real, independently addressable public pages with their own metadata and
an indexable surface:

| Page | Route | Content source |
|---|---|---|
| Home | `/` | existing sections, unchanged visually |
| About | `/about` | profile entity |
| Work | `/work` | 6 project entities |
| Work case study | `/work/:slug` | one project entity |
| Services | `/services` | skills catalog + profile |
| Contact | `/contact` | contact submission endpoint |

Each page must be static HTML at build time — real `<title>`, real
`<meta name="description">`, real content in the initial response — plus
`robots.txt` and `sitemap.xml`.

## 2. Non-goals

- No programmatic per-city service pages (explicit non-goal; see §3.4).
- No new backend infrastructure, ORM, or external service (backend scope freeze).
- No change to the publishing/Lab/Admin domain behaviour.
- No new copy invented for marketing purposes; pages re-present existing data.

## 3. Findings that shape the design

These were measured, not assumed.

### 3.1 The Go server is already almost SSG-ready

`internal/app/app.go` `spaHandler.ServeHTTP` currently does:

```
/api/*      -> apiServer
/mcp        -> mcpServer
/api/admin/auth/* -> authMux
os.Stat(dist + path) is a real file -> http.ServeFile
else        -> http.ServeFile(dist/index.html)   <-- blanket SPA fallback
```

Astro emits `dist/about/index.html`, i.e. a **directory** request. The current
code only serves a path when `!fi.IsDir()`, so `/about` would fall through to the
homepage `index.html`. **One small addition (directory index) makes the existing
Go server serve a prerendered multi-page site with no other change.**

### 3.2 The existing "renders without backend" invariant forces build-time data

`CONTEXT.md` and the recorded project decisions require: *"Static assets must
render cleanly with empty-state fallbacks independently of backend `/api/*`
endpoint availability."*

If Astro fetched `/api/*` during build, a build would require a running server,
and the invariant would be violated. Therefore **page content must be extracted at
build time from the Go source of truth**, not from a live HTTP call.

### 3.3 One source of truth, no drift

Seven portfolio entities are currently hard-coded inside `pkg/api/server.go`. Two
copies (Go literal + JS/JSON) would rot. So the build must **derive** the content
file from Go rather than duplicate it:

```
go run ./tools/contentexport > frontend/src/content/portfolio.json
```

This runs as an Astro `prebuild` step. There is exactly one authored copy.

### 3.4 Programmatic city pages are deliberately out of scope

The benchmark sites (acidopal.id ~200 URLs, ramadigital.id ~1 011 URLs) are
client-acquisition engines built on templated per-city service pages. That model
conflicts with this project's recorded decisions (hard-coded content, backend
scope freeze, explicit non-goals) and trades on volume rather than engineering
evidence. Adopting it is a positioning change, not a page change — it needs its
own decision and its own plan.

## 4. Architecture decision

**Astro SSG for public pages + React islands for interactivity + `/admin` as a
client-only React island, in a single Astro build.**

### Why this shape

- Public pages get real prerendered HTML → real metadata, real sitemap, and link
  previews / non-JS crawlers see actual content.
- `/admin` stays client-rendered (correct — it is an authenticated SPA behind
  cookies and CSRF), hosted as `<AdminApp client:only="react" />`. No rewrite.
- One build system replaces two; `vitest` keeps working because the admin and Lab
  React components and their tests are unchanged.

### Rejected alternatives

| Alternative | Why rejected |
|---|---|
| SPA + head managed in JS | Crawlers that do not execute JS still see one identical title; a sitemap would be misleading. Defeats the point of adding pages. |
| Astro fetching `/api/*` at build | Violates §3.2 invariant; makes builds depend on a live server. |
| Keep Vite for admin, add Astro for public (two builds) | Two toolchains, two dev servers, two dist layouts to reconcile in `spaHandler`. More moving parts for no benefit. |
| Build-go binary embedding Astro output via `go:embed` | Optional later; not needed now. `FRONTEND_DIST` already points at a directory. |

## 5. File structure

```
frontend/
  astro.config.mjs              # NEW  site url, react integration, build.outDir=dist
  src/content/portfolio.json    # NEW  GENERATED (never hand-edited)
  src/data/portfolio.ts         # NEW  typed accessor over the generated JSON
  src/layouts/BaseLayout.astro  # NEW  <head>, nav, footer, per-page meta
  src/pages/
    index.astro                 # MOD  home (existing sections)
    about.astro                 # NEW
    work/index.astro            # NEW
    work/[slug].astro           # NEW  generateStaticPaths from portfolio.json
    services.astro              # NEW
    contact.astro               # NEW
    admin/index.astro           # NEW  <AdminApp client:only="react" />
    lab/index.astro             # NEW
    lab/[slug].astro            # NEW  generateStaticPaths from Lab API at build
  src/islands/
    Coverflow.tsx               # MOV  from components/, hydration directive at call site
    WorkTabs.tsx                # MOV
    FilmAccordion.tsx           # MOV
    ContactForm.tsx             # MOV
    AdminApp.tsx                # NEW  thin wrapper: router + AdminLayout tree
    LabApp.tsx                  # NEW
  public/robots.txt             # NEW
tools/contentexport/main.go     # NEW  emits portfolio.json from pkg/api data
internal/app/app.go             # MOD  directory-index serving
```

## 6. Tasks

### Phase 0 — Prepare (no behaviour change)

**Task 0.1 — Isolated workspace**
- Create branch `feat/public-pages-astro` (worktree optional; main is currently clean).
- Run the existing gate to record a green baseline: `bash scripts/run-publishing-e2e.sh`.
- **Commit:** none (baseline only).

**Task 0.2 — Extract the Go content exporter**

The seven entities currently live in unexported literals inside
`pkg/api/server.go`. Add an exported accessor so both the HTTP path and the
exporter read the same values.

- **Files:** `pkg/api/models.go`, `pkg/api/server.go`, `tools/contentexport/main.go`, `pkg/api/content_test.go`
- **Step 1 (test first):** write `TestPortfolioContentIsExported` asserting
  `api.PortfolioContent()` returns 1 profile, 6 projects, 4 skill categories,
  and that every project has `ID`, `Title`, `Tagline`, `Description`, `Tags`.
  Run `go test ./pkg/api/ -run TestPortfolioContentIsExported` → **FAIL** (undefined).

  > Verified 2026-09-24: `Project` has **no `Slug` field** — its `ID` is already
  > URL-safe and is used as the route segment. The six IDs are `jam-nguar`,
  > `inven-kab-blitar`, `labtu-web`, `cs-portal`, `tour-travel-web`,
  > `nusantara-botanica`. The four skill categories are `Backend & Systems`,
  > `Frontend Engineering`, `AI & Autonomous Agents`, `DevOps & Infrastructure`.
- **Step 2:** add `func PortfolioContent() Content` returning the same structs
  `handleProfile`/`handleProjects`/`handleSkills` already return. Refactor those
  handlers to call it (no JSON shape change — `server_test.go` must stay green).
- **Step 3:** `tools/contentexport/main.go` marshals `PortfolioContent()` with
  `SetEscapeHTML(false)` + `MarshalIndent` to stdout.
- **Step 4:** verify `go run ./tools/contentexport | head` emits valid JSON.
- **Step 5:** `go test ./... && go vet ./...`
- **Commit:** `feat(api): export portfolio content as the single source of truth`

> Guard: this task must not alter `/api/profile`, `/api/projects`, or
> `/api/skills` response bytes. `pkg/api/server_test.go` is the proof.

### Phase 1 — Astro scaffold alongside Vite

**Task 1.1 — Add Astro without removing Vite**
- **Files:** `frontend/package.json`, `frontend/astro.config.mjs`, `frontend/tsconfig.json`
- Add deps: `astro`, `@astrojs/react`. Keep all existing deps.
- `astro.config.mjs`: `output: 'static'`, `build.outDir: 'dist'`,
  `site: 'https://<PORTFOLIO_PUBLIC_ORIGIN>'`, `vite.plugins` reuse of the existing
  Tailwind v4 plugin.
- **Commit:** `build(frontend): add astro alongside vite without removing either`

**Task 1.2 — Move one component as a React island (prove the pattern)**
- **Files:** `frontend/src/layouts/BaseLayout.astro`, `frontend/src/pages/index.astro`, `frontend/src/islands/Coverflow.tsx`
- Move `CoverflowSection.tsx` → `islands/Coverflow.tsx`; mount in `index.astro`
  with `client:visible`. Three.js only loads when scrolled into view.
- **Verify:** `npm run build` in `frontend/` produces `dist/index.html` containing
  the hero markup as static HTML (grep for a known heading string), and the
  coverflow still renders in a browser at the served origin.
- **Commit:** `feat(frontend): render home via astro with coverflow island`

**Task 1.3 — Go server: directory index**

- **Files:** `internal/app/app.go`, `internal/app/app_test.go`
- **Step 1 (test first):** add a test that serves a temp dist containing
  `about/index.html` and asserts `GET /about` returns **that file's** body, and
  `GET /about/` likewise; and that an unknown path still falls back to the root
  `index.html`.
- **Step 2:** in `spaHandler.ServeHTTP`, before the SPA fallback, if
  `fi.IsDir()` then try `filepath.Join(path, "index.html")` and serve it when present.
- **Step 3:** `go test ./internal/app/ -count=1`
- **Commit:** `feat(app): serve directory index files so prerendered pages resolve`

### Phase 2 — The five public pages

Each page task follows the same shape: write the page, add a test proving the
prerendered HTML contains the page's own title/description and its content, then
verify in a browser.

**Task 2.1 — `<BaseLayout>` + per-page metadata (the SEO foundation)**

- **Files:** `frontend/src/layouts/BaseLayout.astro`, `frontend/src/data/portfolio.ts`, `frontend/src/lib/seo.ts`
- `BaseLayout` props: `title`, `description`, `canonicalPath`. Emits `<title>`,
  `<meta name="description">`, `<link rel="canonical">`, Open Graph + Twitter card
  tags, and the shared nav/footer.
- `src/data/portfolio.ts` imports `content/portfolio.json` and exposes typed
  helpers: `getProfile()`, `listProjects()`, `getProject(slug)`, `listSkillCategories()`.
- **Test:** `frontend/src/lib/seo.test.ts` — asserts `pageTitle()`/
  `pageDescription()` produce distinct values for each of the five routes and
  never emit an empty description.
- **Commit:** `feat(frontend): add base layout with per-page metadata`

**Task 2.2 — `/about`**
- **Files:** `frontend/src/pages/about.astro`, `frontend/src/pages/about.test.ts`
- Content: profile title, tagline, bio, highlights, stats, social links.
- **Acceptance:** built `dist/about/index.html` contains the bio text and the
  word "About" in `<title>`.
- **Commit:** `feat(frontend): add /about page`

**Task 2.3 — `/work` + `/work/[slug]`**
- **Files:** `frontend/src/pages/work/index.astro`, `frontend/src/pages/work/[slug].astro`, `frontend/src/pages/work/work.test.ts`
- `[slug].astro` uses `getStaticPaths()` over `listProjects()` keyed by project
  `ID` → 6 pages. `getProject(param)` matches on `ID`, not a `slug` field.
- Case study renders title, tagline, description, metrics, architecture, tags,
  GitHub + demo links, and a back-link to `/work`.
- **Test:** `getStaticPaths` returns exactly the 6 known slugs; each generated
  page's description is non-empty and unique.
- **Commit:** `feat(frontend): add /work catalog and project case studies`

**Task 2.4 — `/services`**
- **Files:** `frontend/src/pages/services.astro`, `frontend/src/pages/services.test.ts`
- Renders the 4 skill categories and their items; each category gets an anchor id.
- **Commit:** `feat(frontend): add /services page`

**Task 2.5 — `/contact`**
- **Files:** `frontend/src/pages/contact.astro`, `frontend/src/islands/ContactForm.tsx`
- Static page holds the copy; the form is an island (needs `POST /api/contact`).
- Must keep the existing WhatsApp behaviour (fetch `/api/contact/whatsapp`, render
  nothing on 404).
- **Test:** the island's existing render tests move to `ContactForm.test.tsx`;
  add a case asserting the form still renders when the WhatsApp fetch 404s.
- **Commit:** `feat(frontend): add /contact page with form island`

### Phase 3 — Lab + Admin on Astro

**Task 3.1 — `/lab` and `/lab/[slug]`**
- **Files:** `frontend/src/pages/lab/index.astro`, `frontend/src/pages/lab/[slug].astro`, `frontend/src/islands/LabCatalog.tsx`, `frontend/src/islands/LabCaseStudy.tsx`
- `getStaticPaths` fetches `/api/lab/projects` **at build time** and tolerates
  failure by producing zero paths plus an empty-state page (preserves §3.2 for
  the Lab, whose data is genuinely dynamic).
- **Commit:** `feat(frontend): move lab catalog and case study to astro`

**Task 3.2 — `/admin` as a client-only island**
- **Files:** `frontend/src/pages/admin/index.astro`, `frontend/src/pages/admin/[...path].astro`, `frontend/src/islands/AdminApp.tsx`
- `AdminApp` wraps the existing `router.tsx` tree (AdminLayout, Overview, Review
  Queue, ReviewDetail). **No admin component is modified.**
- Astro needs a catch-all so deep links like `/admin/reviews/:id` still serve the
  shell; the React router then takes over.
- **Verify:** all 9 existing `*.test.ts(x)` files (6 admin + 2 lab + `router.test.tsx`)
  still pass unchanged.
- **Commit:** `feat(frontend): host admin spa as a client-only astro island`

### Phase 4 — SEO surface

**Task 4.1 — `robots.txt` + `sitemap.xml`**
- **Files:** `frontend/public/robots.txt`, `frontend/src/pages/sitemap.xml.ts`
- `robots.txt` allows public pages, disallows `/admin` and `/api/`.
- `sitemap.xml.ts` is an Astro endpoint: emits every static page plus one entry
  per project slug and per published Lab slug (when available at build).
- **Test:** a unit test asserts the generated sitemap contains `/about`, `/work`,
  all six `/work/<slug>` entries, `/services`, `/contact`, and contains **no**
  `/admin` entry.
- **Commit:** `feat(frontend): add robots.txt and generated sitemap`

### Phase 5 — Cutover

**Task 5.1 — Remove Vite**
- Delete `vite.config.ts`, the Vite-only scripts, and any Vite-only deps that
  Astro does not use. Keep `vitest` and the test toolchain.
- `npm run build` → `astro build`; `npm run dev` → `astro dev`.
- **Commit:** `refactor(frontend): remove vite now that astro owns the build`

**Task 5.2 — Update project docs and gates**
- **Files:** `CONTEXT.md`, `scripts/run-publishing-e2e.sh`, `README`/runbooks as applicable
- Update Seam B/Seam C in `CONTEXT.md` to describe Astro SSG + islands.
- Add a build assertion to the gate: after `npm run build`, `dist/about/index.html`
  and `dist/work/index.html` exist and each contains its own `<title>`.
- **Commit:** `docs: record astro ssg architecture and page inventory`

## 7. Verification gates (must all pass at the end)

```bash
# Backend unchanged behaviour
go build ./... && go vet ./... && go test ./... -count=1 && go test -race ./... -count=1

# Frontend
cd frontend && npm run build       # astro build, must produce per-page index.html
npx tsc --noEmit                   # zero TS errors
npx oxlint                         # zero warnings
npx vitest run                     # all React + new page tests pass

# SEO surface
test -f frontend/dist/robots.txt
grep -q '<loc>' frontend/dist/sitemap.xml
grep -l '<title>About' frontend/dist/about/index.html

# End to end
bash scripts/run-publishing-e2e.sh
```

Plus a **browser check** on the built output at each page: `/`, `/about`, `/work`,
`/work/<slug>`, `/services`, `/contact`, `/lab`, `/admin` — confirming each
resolves to its own document (not the homepage fallback) and that `/admin` still
authenticates.

## 8. Risks

| Risk | Mitigation |
|---|---|
| Three.js/coverflow breaks under Astro hydration | Prove the pattern in Task 1.2 with one island before migrating the rest; use `client:visible`. |
| Admin regression | Admin components are not modified; the 6 existing admin test files are the guard. |
| Build now needs the Go toolchain (for `contentexport`) | Acceptable: this repo is Go-first and the deploy path already builds Go. Documented in Task 5.2. |
| Lab slugs unknown at build time | `getStaticPaths` degrades to an empty-state page rather than failing the build. |
| Duplicate content between `/` sections and `/work` | `/` keeps teasers; sub-pages hold detail. Add explicit canonical tags (Task 2.1). |
| Two toolchains during Phases 1–4 | Time-boxed; Phase 5 removes Vite. |

## 9. Open questions (need an answer before Phase 2)

1. **Canonical host:** what is the production origin for `<link rel="canonical">`
   and `sitemap.xml`? Plan reads `PORTFOLIO_PUBLIC_ORIGIN`; needs a real value.
2. **Home page shape:** does `/` keep all current sections (hero, philosophy,
   expertise, coverflow, contact teaser), or become a shorter landing page that
   links to the new sub-pages? This changes how much is moved, not the plan shape.
3. **`/services` audience:** the current positioning is engineering evidence, not
   service sales. Confirm whether `/services` presents capabilities or priced
   offerings — the latter is the benchmark model and a positioning change.
