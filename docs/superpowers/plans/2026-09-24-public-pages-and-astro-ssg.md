# Public Pages Expansion + Astro SSG Migration — Implementation Plan

**Date:** 2026-09-24 (rev 2, post-research)
**Status:** Decisions resolved; implementation proceeding
**Supersedes (partially):** the Vite-only frontend decision in `CONTEXT.md` §2 Seam C

> I'm using the writing-plans skill to create this implementation plan.

---

## 1. Goal

Ship five real, independently addressable public pages with their own metadata and
an indexable surface:

| Page | Route | Content source |
|---|---|---|
| Home | `/` | positioning + sections, linking to sub-pages |
| About | `/about` | profile entity |
| Work | `/work` | 6 project entities |
| Work case study | `/work/:id` | one project entity |
| Services | `/services` | 4 skill categories (capabilities, not priced offers) |
| Contact | `/contact` | contact submission endpoint |

Plus the SEO surface that does not exist today: `robots.txt`, `sitemap.xml`,
per-page `<title>`/description, canonical URLs, Open Graph/Twitter cards, and
JSON-LD structured data.

Each page must be static HTML at build time — real metadata and real content in
the initial response, before any JavaScript runs.

## 2. Non-goals

- **No blog** (decided). The benchmarks carry 156–824 articles; that is a content
  operation, not a page change. The sitemap is built so blog entries can be added
  later without restructuring.
- No programmatic per-city service pages (§3.4).
- No new backend infrastructure, ORM, or external service (backend scope freeze).
- No change to publishing/Lab/Admin domain behaviour.
- No priced service offerings on `/services` (§9).
- No TypeScript 7 upgrade (blocked by tooling, §3.5).

## 3. Findings that shape the design

All measured against the live repo and the npm registry on 2026-09-24.

### 3.1 The Go server is already almost SSG-ready

`internal/app/app.go` `spaHandler.ServeHTTP` currently does:

```
/api/*            -> apiServer
/mcp              -> mcpServer
/api/admin/auth/* -> authMux
os.Stat(dist+path) is a real file -> http.ServeFile
else              -> http.ServeFile(dist/index.html)   <-- blanket SPA fallback
```

Astro emits `dist/about/index.html`, i.e. a **directory** request, but the handler
only serves a path when `!fi.IsDir()`. So `/about` currently falls through to the
homepage. **Two small additions — directory index, and an `/admin/*` prefix
fallback — make the existing Go server host a prerendered multi-page site.**

### 3.2 The existing "renders without backend" invariant forces build-time data

Recorded project invariant: *"Static assets must render cleanly with empty-state
fallbacks independently of backend `/api/*` endpoint availability."*

If Astro fetched `/api/*` during build, every build would require a running
server. Therefore page content is **exported from the Go source of truth** at
build time, not fetched over HTTP.

### 3.3 One source of truth, no drift

Seven portfolio entities are hard-coded inside `pkg/api/server.go`. Two copies
would rot, so the build **derives** the content file from Go:

```
go run ./tools/contentexport > frontend/src/content/portfolio.json
```

One authored copy; the exporter is the only reader of the literals.

### 3.4 Programmatic city pages stay out of scope

acidopal.id (~200 indexed URLs) and ramadigital.id (~1 011) are client-acquisition
engines built on templated per-city service pages. That conflicts with this
project's recorded decisions (hard-coded content, backend scope freeze, explicit
non-goals) and trades on volume rather than engineering evidence. It is a
positioning change needing its own decision and plan.

### 3.5 Verified version matrix

Measured with `npm view` on 2026-09-24. Node on this machine is **v26.8.1**.

| Package | Latest | Use | Why |
|---|---|---|---|
| `astro` | 7.3.4 | add | engines `node >=22.12.0` ✓. **Astro 7 bundles Vite 8**, the same major the repo already runs. |
| `@astrojs/react` | 7.0.0 | add | peers: react ^17‖^18‖^19 ✓ |
| `oxc-transform-react` | 0.151.0 | add | **new required peer** of `@astrojs/react` 7 (`^0.145.0`) |
| `@astrojs/check` | 0.9.10 | add (dev) | `astro check`; peer `typescript ^5 ‖ ^6` |
| `@astrojs/sitemap` | 3.7.4 | add | peer `vite ^5.2 ‖ ^6 ‖ ^7 ‖ ^8` ✓ |
| `react` / `react-dom` | 19.3.0 | **keep 19.2.8** | already satisfies the peer range; upgrade is unrelated to this work |
| `vitest` | 5.0.1 | **keep 4.1.11** | installed vitest 4 already declares `vite ^8` support ✓ |
| `typescript` | 7.0.2 | **keep 6.0.3** | `@astrojs/check` 0.9.10 peer is `^5 ‖ ^6` — **TS 7 would break type-checking** |
| `tailwindcss` / `@tailwindcss/vite` | 4.3.3 | keep | peer `vite ^5.2 ‖ ^6 ‖ ^7 ‖ ^8` ✓ |

Net: three new dependencies plus one peer. No forced major upgrade anywhere.

### 3.6 A client-side SPA inside Astro needs a fallback, not an adapter

The standard recipe for a React Router SPA in Astro is `prerender = false` plus an
SSR adapter (Node/Vercel/Netlify). **That is rejected here.** Adopting an adapter
would add a second server runtime and contradict the Go single-binary
architecture.

The correct approach for this codebase:

- React Router is configured with `basename="/admin"` and lives in one
  client-only island at `dist/admin/index.html`.
- **The Go `spaHandler` already owns fallback responsibility**; it learns one rule:
  `/admin/*` with no matching real file falls back to `/admin/index.html`.
- No adapter, no second runtime, no duplicated `index.html` per route.

This is strictly better than the adapter recipe for this repo: the fallback stays
in the server we already run and already control.

## 4. Architecture decision

**Astro SSG for public pages + React islands for interactivity + `/admin` as a
client-only React island, in one Astro build served by the existing Go binary.**

### Why this shape

- Public pages become prerendered HTML → real metadata, a meaningful sitemap, and
  working link previews for crawlers that do not execute JavaScript.
- Public pages ship **zero JavaScript by default** (`.astro` components compile
  away). Only genuine interactivity hydrates (coverflow, tabs, contact form).
- `/admin` remains a client-rendered SPA — correct for an authenticated,
  cookie/CSRF-protected surface — hosted as `<AdminApp client:only="react" />`
  with **no admin component modified**.
- The Go server keeps serving `FRONTEND_DIST`; the only change is fallback rules.

### Rejected alternatives

| Alternative | Why rejected |
|---|---|
| SPA + head managed in JS | Crawlers without JS still see one identical title; a sitemap would be misleading. Defeats the purpose. |
| Astro fetching `/api/*` at build | Violates §3.2; makes builds depend on a live server. |
| Astro SSR **adapter** for the admin SPA | Adds a second runtime and contradicts the Go single-binary architecture (§3.6). The Go handler already owns fallback. |
| Two builds (Vite for admin + Astro for public) | Two toolchains, two dev servers, two dist layouts to reconcile. |
| `go:embed` the Astro output into the binary | Unnecessary: `FRONTEND_DIST` already points at a directory. Optional later. |
| Upgrade to TS 7 / vitest 5 / React 19.3 while here | Unrelated scope, and TS 7 actively breaks `@astrojs/check` (§3.5). |

## 5. File structure (as built)

```
frontend/
  astro.config.mjs              # NEW  site, react + sitemap integrations, outDir=dist
  vitest.config.ts              # NEW  getViteConfig({ test: {...} }) preserving jsdom
  src/content/portfolio.json    # NEW  GENERATED by `npm run prebuild` (never hand-edited)
  src/data/portfolio.ts         # NEW  typed accessor over the generated JSON
  src/lib/seo.ts                # NEW  title/description/canonical/JSON-LD builders
  src/lib/seo.test.ts           # NEW  17 tests over that metadata layer
  src/layouts/BaseLayout.astro  # NEW  <head>, nav, footer, per-page meta
  src/pages/
    index.astro                 # NEW  home (hero + coverflow + links to sub-pages)
    about.astro                 # NEW
    work/index.astro            # NEW
    work/[id].astro             # NEW  getStaticPaths over the 6 project IDs
    services.astro              # NEW
    contact.astro               # NEW
    admin/index.astro           # NEW  IslandRouter client:only, noindex
    lab/index.astro             # NEW  IslandRouter client:only, noindex
    robots.txt.ts               # NEW  endpoint (disallows /admin, /api)
  src/islands/
    IslandRouter.tsx            # NEW  client router for /admin + /lab only
    IslandRouter.test.tsx       # MOV  was src/router.test.tsx
  src/components/**             # KEPT  reused as-is; see note below
  .oxlintrc.json                # MOD  ignores the generated .astro directory

tools/contentexport/main.go     # NEW  emits portfolio.json from pkg/api
internal/app/app.go             # MOD  directory index + /admin and /lab shell fallback
```

**Note on reuse.** The plan proposed moving the interactive components into
`src/islands/`. In practice Astro renders an imported React component to static
HTML when no `client:*` directive is given, so the existing components are
imported unmodified: static ones (`Hero`, `TrustBar`, `PhilosophySection`) ship
no JavaScript, and only `CoverflowSection`, `ExpertiseSection`, and
`ContactSection` hydrate. Nothing moved, and no component was edited.

The sitemap is produced by the official `@astrojs/sitemap` integration rather
than a hand-written endpoint, which is why there is no `sitemap.xml.ts`.

**Deleted** (made unreachable by the cutover, or already orphaned):
`vite.config.ts`, `index.html`, `src/main.tsx`, `src/App.tsx`, `src/App.css`,
`src/router.tsx`, `src/assets/`, `components/Navbar.tsx`, `components/Footer.tsx`,
`components/FilmAccordionSection.tsx`, `components/ProjectsSection.tsx`,
`components/SkillsSection.tsx`, plus the `three` and `@types/three`
dependencies that only `FilmAccordionSection` used.

## 6. Resolved decisions

These were open in rev 1 and are now settled by best practice.

1. **Canonical host** → `astro.config.mjs` reads `PORTFOLIO_PUBLIC_ORIGIN`, falling
   back to the value already used for `PortfolioOrigin` in
   `internal/app/config.go`. Canonical URLs and the sitemap are generated from
   `site`, so there is exactly one place to change. **Production origin must be
   confirmed before deploy** — until then the fallback is used and is documented.
2. **Home page shape** → keeps the hero and the coverflow (the visual
   differentiator) and condenses the rest into summaries that link to `/about`,
   `/work`, `/services`, `/contact`. Rationale: shipping the full text on both `/`
   and the sub-page creates duplicate content, which is the opposite of the goal.
3. **`/services`** → presents **capabilities** derived from the four skill
   categories already in the data, not priced offerings. The priced-offer model is
   the benchmark's positioning and would be a positioning change, not a page.
4. **No blog** → out of scope (§2); sitemap is structured so it can be added.
5. **TypeScript stays at 6** → TS 7 breaks `@astrojs/check` (§3.5).

## 7. Tasks

### Phase 0 — Go contracts (shared prerequisite; done first, inline)

**Task 0.1 — Export portfolio content from Go**
- **Files:** `pkg/api/models.go`, `pkg/api/server.go`, `tools/contentexport/main.go`, `pkg/api/content_test.go`
- **Test first:** `TestPortfolioContentIsExported` asserting 1 profile, 6 projects,
  4 skill categories, and that every project has `ID`, `Title`, `Tagline`,
  `Description`, `Tags`. Run → FAIL (undefined).
  > Verified: `Project` has **no `Slug` field**; its `ID` is URL-safe and is the
  > route segment. IDs: `jam-nguar`, `inven-kab-blitar`, `labtu-web`, `cs-portal`,
  > `tour-travel-web`, `nusantara-botanica`. Categories: `Backend & Systems`,
  > `Frontend Engineering`, `AI & Autonomous Agents`, `DevOps & Infrastructure`.
- Add `func PortfolioContent() Content`; refactor the three handlers to call it.
  **`pkg/api/server_test.go` must stay green — response bytes must not change.**
- `tools/contentexport` marshals with `SetEscapeHTML(false)` + indent.
- **Commit:** `feat(api): export portfolio content as the single source of truth`

**Task 0.2 — Go: directory index + `/admin/*` fallback**
- **Files:** `internal/app/app.go`, `internal/app/app_test.go`
- **Tests first:**
  - `GET /about` returns the body of `dist/about/index.html` (not the root index).
  - `GET /admin/reviews/abc` returns `dist/admin/index.html` when no real file matches.
  - `GET /admin/index.html` still serves the real file.
  - `GET /api/...` still routes to the API (fallback must not shadow it).
  - An unknown public path still falls back to the root `index.html`.
- **Commit:** `feat(app): serve directory index files and an admin spa fallback`

### Phase 1 — Astro scaffold

**Task 1.1 — Add Astro (Vite kept temporarily)**
- **Files:** `frontend/package.json`, `frontend/astro.config.mjs`, `frontend/vitest.config.ts`
- Add exactly: `astro@^7.3.4`, `@astrojs/react@^7.0.0`, `oxc-transform-react`,
  `@astrojs/sitemap@^3.7.4`, `@astrojs/check` (dev).
- `vitest.config.ts` uses `getViteConfig` so existing tests keep running unchanged.
- **Verify:** `npx vitest run` → still 56 passing.
- **Commit:** `build(frontend): add astro 7 alongside vite`

**Task 1.2 — Prove the island pattern (coverflow)**
- **Files:** `src/layouts/BaseLayout.astro`, `src/pages/index.astro`, `src/islands/Coverflow.tsx`
- Mount coverflow with `client:visible` (below the fold; Three.js must not block paint).
- **Verify:** built `dist/index.html` contains hero markup as static HTML (grep a
  known heading) and the page works in a browser. Confirm Three.js is not in the
  initial JS payload.
- **Commit:** `feat(frontend): render home via astro with coverflow island`

### Phase 2 — Public pages (parallelisable after Phase 1)

**Task 2.1 — BaseLayout + metadata + structured data**
- **Files:** `src/layouts/BaseLayout.astro`, `src/lib/seo.ts`, `src/data/portfolio.ts`, `src/lib/seo.test.ts`
- `BaseLayout` props: `title`, `description`, `canonicalPath`, optional `jsonLd`.
- Emits `<title>`, `<meta name="description">`, `<link rel="canonical">`,
  Open Graph + Twitter card tags, and JSON-LD when supplied.
- Semantics: exactly one `<h1>` per page; `<main>`, `<nav aria-label>`, `<footer>`.
- **Test:** distinct non-empty title/description for each of the 5 routes.

**Task 2.2 — `/about`** — profile, bio, highlights, stats, social links. JSON-LD `Person`.
**Task 2.3 — `/work` + `/work/[slug]`** — `getStaticPaths` over the 6 IDs; each case study has unique title/description; JSON-LD `CreativeWork`. Back-link to `/work`.
**Task 2.4 — `/services`** — the 4 skill categories, each with a stable anchor id.
**Task 2.5 — `/contact`** — static copy + `ContactForm` island; WhatsApp behaviour preserved (render nothing on 404).

### Phase 3 — Lab + Admin (parallelisable after Phase 2.1)

**Task 3.1 — `/lab` + `/lab/[slug]`** — `getStaticPaths` fetches `/api/lab/projects`
at build time and **degrades to zero paths plus an empty state** on failure,
preserving §3.2 for genuinely dynamic data.
**Task 3.2 — `/admin`** — `AdminApp` island, `basename="/admin"`, wrapping the
existing router tree. **No admin component modified.**

### Phase 4 — SEO surface

**Task 4.1 — `robots.txt` + `sitemap.xml` endpoints**
- `robots.txt.ts`: allow public, disallow `/admin` and `/api/`, reference the sitemap.
- `sitemap.xml.ts`: every static page + one entry per project ID (+ Lab slugs when
  available at build).
- **Test:** sitemap contains `/about`, `/work`, all six `/work/<id>`, `/services`,
  `/contact`; contains **no** `/admin`.

### Phase 5 — Cutover

**Task 5.1 — Remove Vite** (config, scripts, `@vitejs/plugin-react`) once Astro owns build+test.
**Task 5.2 — Update docs and gates** — `CONTEXT.md` Seams B/C, and add to
`scripts/run-publishing-e2e.sh` an assertion that `dist/about/index.html` and
`dist/work/index.html` exist and each carries its own `<title>`.

## 8. Verification gates

```bash
go build ./... && go vet ./... && go test ./... -count=1 && go test -race ./... -count=1
cd frontend && npm run build && npx astro check && npx oxlint && npx vitest run
test -f frontend/dist/robots.txt && grep -q '<loc>' frontend/dist/sitemap.xml
grep -l '<title>About' frontend/dist/about/index.html
bash scripts/run-publishing-e2e.sh
```

Plus a **browser pass** over built output: `/`, `/about`, `/work`, `/work/<id>`,
`/services`, `/contact`, `/lab`, and `/admin` deep link — each must resolve to its
own document and `/admin` must still authenticate.

## 9. Risks

| Risk | Mitigation |
|---|---|
| Astro 7 Rust compiler is strict about malformed HTML | New `.astro` files are authored clean; build fails loudly rather than silently mis-rendering. |
| Three.js/coverflow under hydration | Task 1.2 proves the pattern before the rest migrate; `client:visible` keeps it off the critical path. |
| Admin regression | Admin components untouched; the 6 admin test files plus `router.test.tsx` are the guard, plus the new Go fallback tests. |
| Build now needs the Go toolchain (content exporter) | Acceptable and documented: this repo is Go-first and the deploy path already builds Go. |
| Duplicate content between `/` and sub-pages | Home condenses and links; canonicals set per page (Task 2.1). |
| Production origin unknown | Single `site` value in `astro.config.mjs`; documented fallback until confirmed. |

## 10. Outcome

All six phases are implemented and verified. Measured, not assumed.

**Routes served by the Go binary** (`go run . -port 8099 -dist ./frontend/dist`,
exercised over real HTTP):

| Route | Status | Title |
|---|---|---|
| `/` | 200 | Regio Dani Pangestu — Full-Stack Engineer |
| `/about` | 200 | About — Regio Dani Pangestu |
| `/work` | 200 | Work — Regio Dani Pangestu |
| `/work/jam-nguar` | 200 | Jam Nguar — Regio Dani Pangestu |
| `/services` | 200 | Services — Regio Dani Pangestu |
| `/contact` | 200 | Contact — Regio Dani Pangestu |
| `/admin/reviews/abc-123` | 200 | Admin shell, deep link intact |
| `/lab/some-slug` | 200 | Design Lab shell |
| `/robots.txt`, `/sitemap-index.xml`, `/sitemap-0.xml` | 200 | text/plain, text/xml |
| `/api/health`, `/api/profile`, `/api/projects`, `/api/skills` | 200 | unchanged |

Thirteen documents are built: the six public pages, six project case studies,
and two application shells.

**JavaScript delivered per page.** External script references in the built HTML:

| Page | Scripts |
|---|---|
| `/about`, `/work`, `/work/<id>`, `/services` | 0 |
| `/contact` | 1 (the form) |
| `/` | 3 (coverflow, expertise tabs, runtime) |

`/about`, `/work`, `/work/<id>`, and `/services` ship no JavaScript at all.

**Gates.**

```bash
bash scripts/run-publishing-e2e.sh   # exit 0, "gate OK"
```

runs `go test -race ./...`, `go vet ./...`, `npx vitest run` (10 files, 73 tests),
`npm run build`, `npx astro check` (0 errors, 0 warnings, 0 hints), `npx tsc -b`,
`npx oxlint` (clean), the static output assertions, and
`go test ./e2e -run TestPublishingFlow`.

**Verified in a real browser** (Chromium, against the built output): every page
resolves to its own document with its own `<h1>`; the coverflow and expertise
tabs hydrate into working controls; the admin island mounts with its
navigation and sign-in state; the Design Lab renders its mandatory
non-affiliation disclaimer; the contact form mounts with its fields.

**Byte-identity of the refactor.** `/api/profile`, `/api/projects`, and
`/api/skills` produced identical sha256 digests before and after the content
was moved out of the handlers, and the generated `portfolio.json` compares equal
to the live API for all three.

**Corrections made while verifying** (things the plan or the codebase had
wrong, found by running the gates rather than the tests):

1. `Profile.phone` was declared required in `frontend/src/types.ts` while the Go
   API marks it `json:"-"` and never serialises it. The type was wrong; it is
   now optional.
2. `React.FormEvent` is deprecated in React 19 — the gate counts it as a
   warning. Two handlers now use `React.SubmitEvent`.
3. `oxlint` linted the generated `.astro/types.d.ts` and flagged its
   Astro-authored triple-slash reference. The generated directory is now
   ignored rather than hand-patched.
4. My own first gate script compared five page titles against the number six,
   and its JavaScript check matched the literal string "Next.js" in the project
   tech tags. Both are fixed; the gate now fails only on a real regression.
5. The Design Lab turned out to be client-routed (`useParams`, Router `Link`),
   so it needs a shell fallback just like the admin area. The Go handler was
   generalised from one hardcoded prefix to a list rather than special-casing a
   second one in place.
