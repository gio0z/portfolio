# Admin, MCP, and Design Lab Publishing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a secure draft-first publishing system where an internal AI agent prepares redesigns through MCP and the owner reviews one immutable preview in Admin before atomically publishing it to the Lab and portfolio Design Lab tab.

**Architecture:** Extend the existing Go modular monolith with focused `publishing`, `admin`, and `mcp` modules behind one Publishing Service interface. Persist registry and audit state through a repository interface, store immutable artifacts behind an artifact-store interface, and expose separate public, Admin, and MCP HTTP boundaries. Keep the existing Vite portfolio client and add route-aware Admin and Lab views without introducing a separate frontend framework.

**Tech Stack:** Go 1.23 `net/http`, SQLite via `modernc.org/sqlite`, Go OAuth2, Vite 8, React 19, TypeScript 6, Tailwind CSS 4, Vitest, React Testing Library, MCP Streamable HTTP, SHA-256, Playwright.

**Spec:** `docs/superpowers/specs/2026-09-03-admin-mcp-lab-publishing-design.md`

## Global Constraints

- Agent-accessible MCP operations stop at `IN_REVIEW`; approval and publication are Admin-only.
- **Approve & Publish** publishes one immutable `submission_revision + artifact_sha256` to Lab and portfolio atomically.
- Any post-review mutation invalidates approval and returns the submission to `DRAFT`.
- Private GitHub repositories are excluded from public output by default.
- Every third-party redesign displays: `Independent redesign concept. Not affiliated with or endorsed by the original company.`
- Admin cookies are host-only, `HttpOnly`, `Secure`, and `SameSite=Strict`.
- Every state mutation requires authorization, request ID, idempotency key, and append-only audit recording.
- Admin, preview, Lab, and public portfolio origins do not share privileged cookies.
- MCP sampling is disabled and MCP has no shell, arbitrary filesystem, raw SQL, secret, approval, or publish tools.
- Existing public portfolio behavior and `Featured Engineering Work` remain operational throughout implementation.

---

## Phase 1 — Publishing Domain and Durable Registry

### Task 1: Publishing Domain State Machine

**Files:**
- Create: `internal/publishing/model.go`
- Create: `internal/publishing/state.go`
- Test: `internal/publishing/state_test.go`

**Interfaces:**
- Produces: `type SubmissionState string`, `type LabProject`, `type Submission`, `type Approval`, `func Transition(current, next SubmissionState, actor ActorKind) error`, `func (s Submission) ReviewedArtifact() ArtifactRef`.
- Consumes: only Go standard library types.

- [ ] **Step 1: Write the failing state-transition table test**

```go
func TestTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    publishing.SubmissionState
		to      publishing.SubmissionState
		actor   publishing.ActorKind
		wantErr bool
	}{
		{"agent submits review", publishing.PreviewReady, publishing.InReview, publishing.ActorAgent, false},
		{"agent cannot approve", publishing.InReview, publishing.Approved, publishing.ActorAgent, true},
		{"owner approves", publishing.InReview, publishing.Approved, publishing.ActorOwner, false},
		{"approved publishes", publishing.Approved, publishing.Published, publishing.ActorSystem, false},
		{"review changes requested", publishing.InReview, publishing.ChangesRequested, publishing.ActorOwner, false},
		{"published archives", publishing.Published, publishing.Archived, publishing.ActorOwner, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := publishing.Transition(tt.from, tt.to, tt.actor)
			if (err != nil) != tt.wantErr { t.Fatalf("Transition() error = %v, wantErr %v", err, tt.wantErr) }
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/publishing -run TestTransition -v`  
Expected: FAIL because package/types do not exist.

- [ ] **Step 3: Implement the domain types and explicit transition table**

`Submission` must include `ID`, `LabProjectID`, `Revision`, `State`, `ArtifactSHA256`, `PreviewURL`, `BuildResult`, `TestResult`, `SecurityScanResult`, `PortfolioMetadata`, `SubmittedBy`, `SubmittedAt`, and `UpdatedAt`. `Approval` must bind submission ID, revision, hash, owner identity, timestamp, and idempotency key.

- [ ] **Step 4: Run package tests**

Run: `go test ./internal/publishing -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/publishing/model.go internal/publishing/state.go internal/publishing/state_test.go
git commit -m "feat(publishing): define review state machine"
```

### Task 2: Repository Contract and SQLite Registry

**Files:**
- Create: `internal/publishing/repository.go`
- Create: `internal/publishing/sqlite_repository.go`
- Create: `internal/publishing/migrations/001_registry.sql`
- Test: `internal/publishing/sqlite_repository_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Consumes: Task 1 domain types.
- Produces: `type Repository interface`, `func NewSQLiteRepository(db *sql.DB) *SQLiteRepository`, `func OpenRegistry(path string) (*sql.DB, error)`, and transaction-capable mutation methods.

```go
type Repository interface {
	CreateLabProject(context.Context, LabProject) error
	GetLabProject(context.Context, string) (LabProject, error)
	ListLabProjects(context.Context, ProjectFilter) ([]LabProject, error)
	CreateSubmission(context.Context, Submission) error
	GetSubmission(context.Context, string) (Submission, error)
	UpdateSubmission(context.Context, Submission, int64) error
	RecordApproval(context.Context, Approval) error
	GetIdempotencyResult(context.Context, string) (*OperationResult, error)
	RecordIdempotencyResult(context.Context, string, OperationResult) error
	AppendAudit(context.Context, AuditEvent) error
	WithTx(context.Context, func(Repository) error) error
}
```

- [ ] **Step 1: Write failing repository tests**

Cover project persistence, optimistic revision conflict, state persistence, idempotency lookup, approval hash binding, and append-only audit ordering using a temporary SQLite database.

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/publishing -run SQLite -v`  
Expected: FAIL because repository implementation is absent.

- [ ] **Step 3: Add SQLite dependency and migration**

Run: `go get modernc.org/sqlite@latest`.

Create tables: `lab_projects`, `submissions`, `approvals`, `idempotency_results`, `audit_events`, and `publications`. Add unique constraints on project slug, `(submission_id, revision)`, artifact hash where required, and idempotency key.

- [ ] **Step 4: Implement repository methods and transactions**

Use parameterized queries only. Store structured metadata as JSON. Return typed `ErrNotFound`, `ErrConflict`, and `ErrDuplicateIdempotencyKey` errors.

- [ ] **Step 5: Run tests and race detector**

Run: `go test -race ./internal/publishing -v`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/publishing
git commit -m "feat(publishing): add durable SQLite registry"
```

### Task 3: Immutable Artifact Store and Validation

**Files:**
- Create: `internal/publishing/artifacts.go`
- Create: `internal/publishing/local_artifact_store.go`
- Create: `internal/publishing/asset_validation.go`
- Test: `internal/publishing/artifacts_test.go`

**Interfaces:**
- Produces: `ArtifactStore`, `ArtifactRef`, `AssetPolicy`, `ValidateAndHash`, and safe local development storage.

```go
type ArtifactStore interface {
	Put(context.Context, io.Reader, AssetMetadata) (ArtifactRef, error)
	Open(context.Context, string) (io.ReadCloser, ArtifactRef, error)
	Promote(context.Context, ArtifactRef, string) (PublicationRef, error)
	DeletePreview(context.Context, string) error
}
```

- [ ] **Step 1: Write failing artifact tests**

Tests must prove SHA-256 content addressing, duplicate-content deduplication, maximum-size enforcement, MIME sniffing, executable rejection, `../` path rejection, symlink rejection, and SVG script rejection.

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/publishing -run Artifact -v`  
Expected: FAIL because artifact store is absent.

- [ ] **Step 3: Implement validation and immutable storage**

Development storage layout:

```text
var/portfolio/artifacts/sha256/<first-two>/<full-hash>
var/portfolio/publications/<project-id>/<revision>.json
```

Accept PNG, JPEG, WebP, MP4, JSON, HTML, CSS, JavaScript, and sanitized static archives. Write via temp file, fsync, hash verification, then atomic rename.

- [ ] **Step 4: Run tests**

Run: `go test -race ./internal/publishing -run 'Artifact|Asset' -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/publishing/artifacts.go internal/publishing/local_artifact_store.go internal/publishing/asset_validation.go internal/publishing/artifacts_test.go
git commit -m "feat(publishing): add immutable artifact storage"
```

---

## Phase 2 — Publishing Service and HTTP API

### Task 4: Draft and Review Publishing Service

**Files:**
- Create: `internal/publishing/service.go`
- Create: `internal/publishing/service_errors.go`
- Test: `internal/publishing/service_test.go`

**Interfaces:**
- Consumes: `Repository` and `ArtifactStore` from Phase 1.
- Produces: one `PublishingService` consumed by MCP and Admin handlers.

```go
type PublishingService interface {
	CreateDraft(context.Context, Actor, CreateDraftInput) (LabProject, Submission, error)
	UpdateDraft(context.Context, Actor, UpdateDraftInput) (Submission, error)
	AttachAsset(context.Context, Actor, AttachAssetInput) (ArtifactRef, error)
	MarkPreviewReady(context.Context, Actor, PreviewResult) (Submission, error)
	RequestReview(context.Context, Actor, string) (Submission, error)
	RequestChanges(context.Context, Actor, ReviewDecision) (Submission, error)
	Reject(context.Context, Actor, ReviewDecision) (Submission, error)
	ApproveAndPublish(context.Context, Actor, ApproveInput) (PublicationResult, error)
	Archive(context.Context, Actor, string) error
}
```

- [ ] **Step 1: Write failing service tests**

Cover:

- Agent draft creation
- Required disclaimer injection
- Agent rejection from approval
- Review request only after build/test/scan pass
- Mutation after review creates a new revision and invalidates approval
- Owner approval bound to reviewed hash
- Repeated idempotency key returns identical result
- Atomic publish rollback when either public destination fails
- Complete audit event fields

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/publishing -run Service -v`  
Expected: FAIL because service is absent.

- [ ] **Step 3: Implement service orchestration**

Make authorization, state transition, validation, repository transaction, artifact promotion, dual-destination publication, and audit append one explicit sequence. Map failures to stable codes: `validation_failed`, `forbidden`, `invalid_state`, `artifact_mismatch`, `scan_failed`, `slug_conflict`, and `publication_failed`.

- [ ] **Step 4: Run tests and race detector**

Run: `go test -race ./internal/publishing -run Service -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/publishing/service.go internal/publishing/service_errors.go internal/publishing/service_test.go
git commit -m "feat(publishing): orchestrate drafts reviews and publication"
```

### Task 5: Public Lab and Design Lab API

**Files:**
- Create: `internal/publicapi/handler.go`
- Test: `internal/publicapi/handler_test.go`
- Modify: `pkg/api/server.go`

**Interfaces:**
- Produces:
  - `GET /api/lab/projects`
  - `GET /api/lab/projects/{slug}`
  - `GET /api/portfolio/design-lab`
- Consumes: read-only repository/publication queries.

- [ ] **Step 1: Write failing public-handler tests**

Assert only `PUBLISHED` records appear, mandatory disclaimer is present, private source metadata is omitted, missing slugs return 404, and JSON responses use cache headers without exposing Admin data.

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/publicapi -v`  
Expected: FAIL because handler is absent.

- [ ] **Step 3: Implement the public handlers and mount them**

Mount public routes in `pkg/api/server.go`. Keep existing profile, project, skills, contact, and health endpoints unchanged.

- [ ] **Step 4: Run API tests**

Run: `go test -race ./internal/publicapi ./pkg/api -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/publicapi pkg/api/server.go
git commit -m "feat(api): expose published Design Lab catalog"
```

### Task 6: Admin Authentication and Security Middleware

**Files:**
- Create: `internal/adminauth/session.go`
- Create: `internal/adminauth/github_oauth.go`
- Create: `internal/adminauth/csrf.go`
- Create: `internal/adminauth/middleware.go`
- Test: `internal/adminauth/auth_test.go`
- Modify: `main.go`

**Interfaces:**
- Produces: `OwnerMiddleware`, OAuth login/callback handlers, session issuance, CSRF issuance/validation, and step-up approval verification interface.

- [ ] **Step 1: Write failing authentication tests**

Test non-allowlisted GitHub identity rejection, allowlisted `gio0z` acceptance, secure cookie attributes, session expiration, CSRF failure, CSRF success, and missing step-up authentication rejection on approval.

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/adminauth -v`  
Expected: FAIL because auth package is absent.

- [ ] **Step 3: Implement OAuth and session middleware**

Read settings from environment:

```text
ADMIN_GITHUB_CLIENT_ID
ADMIN_GITHUB_CLIENT_SECRET
ADMIN_SESSION_SIGNING_KEY
ADMIN_ALLOWED_GITHUB_LOGIN=gio0z
ADMIN_PUBLIC_ORIGIN
```

Do not store settings in source or `config.yaml`. Use cryptographically random state, PKCE, signed/rotatable server-side sessions, and host-only secure cookies.

- [ ] **Step 4: Add step-up approval interface**

Define `ApprovalVerifier` with a development implementation guarded by environment and a production passkey/WebAuthn implementation point. Production startup fails closed if approval verification is not configured.

- [ ] **Step 5: Run tests**

Run: `go test -race ./internal/adminauth -v`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/adminauth main.go
git commit -m "feat(admin): add owner authentication and CSRF protection"
```

### Task 7: Admin Review HTTP API

**Files:**
- Create: `internal/adminapi/handler.go`
- Create: `internal/adminapi/dto.go`
- Test: `internal/adminapi/handler_test.go`
- Modify: `pkg/api/server.go`

**Interfaces:**
- Produces:
  - `GET /api/admin/overview`
  - `GET /api/admin/reviews`
  - `GET /api/admin/reviews/{id}`
  - `POST /api/admin/reviews/{id}/request-changes`
  - `POST /api/admin/reviews/{id}/reject`
  - `POST /api/admin/reviews/{id}/approve-and-publish`
  - `POST /api/admin/projects/{id}/archive`
  - `GET /api/admin/audit`
- Consumes: `PublishingService`, owner middleware, CSRF verifier, and approval verifier.

- [ ] **Step 1: Write failing handler tests**

Use `httptest` to verify authentication, authorization, CSRF, idempotency, required request ID, artifact mismatch, invalid state, successful atomic publication, sanitized errors, and response DTOs.

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/adminapi -v`  
Expected: FAIL because handler is absent.

- [ ] **Step 3: Implement route handlers**

Use explicit JSON decoders with size limits and unknown-field rejection. Never return internal filesystem paths, tokens, SQL errors, or stack traces.

- [ ] **Step 4: Run tests**

Run: `go test -race ./internal/adminapi ./pkg/api -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/adminapi pkg/api/server.go
git commit -m "feat(admin): add secure review and publishing API"
```

---

## Phase 3 — Admin and Public Frontends

### Task 8: Frontend Test Harness and Route Shell

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/package-lock.json`
- Modify: `frontend/src/App.tsx`
- Create: `frontend/src/router.tsx`
- Create: `frontend/src/test/setup.ts`
- Test: `frontend/src/router.test.tsx`

**Interfaces:**
- Produces: route-aware shells for `/`, `/admin/*`, and `/lab/*`.

- [ ] **Step 1: Add failing route test**

```tsx
it('renders the admin shell only for /admin routes', () => {
  window.history.pushState({}, '', '/admin/reviews');
  render(<AppRouter />);
  expect(screen.getByRole('heading', { name: /review queue/i })).toBeInTheDocument();
  expect(screen.queryByText(/featured engineering work/i)).not.toBeInTheDocument();
});
```

- [ ] **Step 2: Install test and router dependencies**

Run:

```bash
npm install react-router-dom
npm install -D vitest jsdom @testing-library/react @testing-library/jest-dom @testing-library/user-event
```

Add scripts: `test`, `test:watch`, and `test:coverage`.

- [ ] **Step 3: Run test to verify failure**

Run: `npm test -- --run src/router.test.tsx`  
Expected: FAIL because `AppRouter` is absent.

- [ ] **Step 4: Implement route shell**

Keep the existing portfolio at `/`; add lazy route boundaries for Admin and Lab to preserve public bundle size.

- [ ] **Step 5: Run tests and build**

Run:

```bash
npm test -- --run
npm run build
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/src/App.tsx frontend/src/router.tsx frontend/src/router.test.tsx frontend/src/test/setup.ts
git commit -m "feat(frontend): add portfolio admin and Lab route shells"
```

### Task 9: Admin Overview and Review Queue UI

**Files:**
- Create: `frontend/src/admin/AdminLayout.tsx`
- Create: `frontend/src/admin/AdminOverview.tsx`
- Create: `frontend/src/admin/ReviewQueue.tsx`
- Create: `frontend/src/admin/AdminApi.ts`
- Create: `frontend/src/admin/types.ts`
- Test: `frontend/src/admin/ReviewQueue.test.tsx`

**Interfaces:**
- Consumes: Task 7 Admin API.
- Produces: authenticated Admin navigation and review queue.

- [ ] **Step 1: Write failing review-queue test**

Render two submissions and assert draft title, original product, status, test/build/scan evidence, artifact hash, and review actions appear. Assert a failed scan disables approval.

- [ ] **Step 2: Run test to verify failure**

Run: `npm test -- --run src/admin/ReviewQueue.test.tsx`  
Expected: FAIL because Admin components are absent.

- [ ] **Step 3: Implement Admin API client and queue**

Use same-origin credentials, read CSRF token from the authenticated bootstrap response, attach request IDs and idempotency keys to mutations, and map stable API errors to actionable UI messages.

- [ ] **Step 4: Implement Overview cards**

Show review count, build failures, published projects, latest deployment, and recent MCP activity.

- [ ] **Step 5: Run tests and build**

Run:

```bash
npm test -- --run src/admin/ReviewQueue.test.tsx
npm run build
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/admin
git commit -m "feat(admin-ui): add overview and review queue"
```

### Task 10: Review Detail, Preview Sandbox, and Approval UI

**Files:**
- Create: `frontend/src/admin/ReviewDetail.tsx`
- Create: `frontend/src/admin/PreviewFrame.tsx`
- Create: `frontend/src/admin/EvidencePanel.tsx`
- Create: `frontend/src/admin/ApprovalDialog.tsx`
- Test: `frontend/src/admin/ReviewDetail.test.tsx`

**Interfaces:**
- Consumes: review DTO and Admin mutations from Tasks 7 and 9.
- Produces: complete owner review and **Approve & Publish** experience.

- [ ] **Step 1: Write failing security and interaction tests**

Assert preview iframe uses `sandbox="allow-scripts allow-forms"`, does not include `allow-same-origin`, supports desktop/mobile dimensions, displays before/after assets, blocks approval on hash/evidence failure, requires step-up verification, and sends reviewed artifact hash plus idempotency key.

- [ ] **Step 2: Run tests to verify failure**

Run: `npm test -- --run src/admin/ReviewDetail.test.tsx`  
Expected: FAIL because review detail is absent.

- [ ] **Step 3: Implement review detail**

Include tabs: Preview, Before/After, Evidence, Portfolio Card, and Lab Page. Show the exact artifact SHA-256 prominently.

- [ ] **Step 4: Implement decisions**

Provide **Approve & Publish**, **Request Changes**, **Reject**, and **Archive**. Approval dialog summarizes both publication destinations and requests step-up verification.

- [ ] **Step 5: Run tests and build**

Run:

```bash
npm test -- --run src/admin/ReviewDetail.test.tsx
npm run build
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/admin
git commit -m "feat(admin-ui): add secure artifact review and approval"
```

### Task 11: Portfolio Engineering Work / Design Lab Tabs

**Files:**
- Modify: `frontend/src/components/CoverflowSection.tsx`
- Create: `frontend/src/components/WorkTabs.tsx`
- Create: `frontend/src/components/DesignLabCard.tsx`
- Modify: `frontend/src/types.ts`
- Test: `frontend/src/components/WorkTabs.test.tsx`

**Interfaces:**
- Consumes: existing engineering projects and `GET /api/portfolio/design-lab`.
- Produces: two semantically distinct views under `Featured Engineering Work`.

- [ ] **Step 1: Write failing tab test**

Assert `Engineering Work` is selected by default; switching to `Design Lab` requests published Lab records, shows the required disclaimer, changes card actions to `View Case Study`, and does not label Lab work as a GitHub engineering repository.

- [ ] **Step 2: Run test to verify failure**

Run: `npm test -- --run src/components/WorkTabs.test.tsx`  
Expected: FAIL because tabs are absent.

- [ ] **Step 3: Implement tabs without changing the approved heading**

Keep the heading exactly `Featured Engineering Work`. Preserve the existing layered coverflow for Engineering Work. Reuse the layout—not the metadata contract—for Design Lab.

- [ ] **Step 4: Add catalog link**

Design Lab view ends with `Explore Design Lab`, routed to the configured Lab origin.

- [ ] **Step 5: Run tests, build, and visual snapshot**

Run:

```bash
npm test -- --run src/components/WorkTabs.test.tsx
npm run build
node capture_sections.mjs
```

Expected: PASS and a new versioned screenshot showing both tabs.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components frontend/src/types.ts screenshots
git commit -m "feat(portfolio): add Engineering Work and Design Lab views"
```

### Task 12: Public Lab Catalog and Case Study Routes

**Files:**
- Create: `frontend/src/lab/LabCatalog.tsx`
- Create: `frontend/src/lab/LabCaseStudy.tsx`
- Create: `frontend/src/lab/LabApi.ts`
- Test: `frontend/src/lab/LabCatalog.test.tsx`
- Test: `frontend/src/lab/LabCaseStudy.test.tsx`

**Interfaces:**
- Consumes: Task 5 public Lab API.
- Produces: published Lab catalog and case-study presentation.

- [ ] **Step 1: Write failing public Lab tests**

Assert only published records render, disclaimer appears above the fold, missing slug renders not-found state, and interactive preview remains sandboxed.

- [ ] **Step 2: Run tests to verify failure**

Run: `npm test -- --run src/lab`  
Expected: FAIL because Lab routes are absent.

- [ ] **Step 3: Implement Lab catalog and case study**

Catalog cards show original product, redesign focus, platform, status, and cover. Case-study pages show rationale, before/after evidence, media, and sandboxed live preview.

- [ ] **Step 4: Run tests and build**

Run:

```bash
npm test -- --run src/lab
npm run build
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lab
git commit -m "feat(lab): add public redesign catalog and case studies"
```

---

## Phase 4 — MCP Publisher and Sandboxed Preview Pipeline

### Task 13: MCP Authentication and Tool Contracts

**Files:**
- Create: `internal/mcppublisher/auth.go`
- Create: `internal/mcppublisher/tools.go`
- Create: `internal/mcppublisher/server.go`
- Test: `internal/mcppublisher/server_test.go`

**Interfaces:**
- Consumes: `PublishingService` from Task 4.
- Produces: Streamable HTTP MCP endpoint `/mcp/portfolio` with read/draft/preview/review tools only.

- [ ] **Step 1: Write failing MCP contract test**

Assert expected tools are present and forbidden names containing `approve`, `publish`, `delete_permanently`, `secret`, `shell`, `filesystem`, or `sql` are absent. Assert sampling is not advertised.

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/mcppublisher -v`  
Expected: FAIL because MCP server is absent.

- [ ] **Step 3: Implement scoped token authentication**

Token claims include actor ID, profile ID, scopes, issued-at, expiry, and token ID. Require TLS at the edge and reject expired/revoked tokens.

- [ ] **Step 4: Implement JSON schemas for each tool**

Reject unknown fields and oversized payloads. Map service errors to stable MCP error data without secrets or internal paths.

- [ ] **Step 5: Run MCP tests**

Run: `go test -race ./internal/mcppublisher -v`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/mcppublisher
git commit -m "feat(mcp): add scoped portfolio publisher tools"
```

### Task 14: Sandboxed Build and Preview Runner

**Files:**
- Create: `internal/preview/runner.go`
- Create: `internal/preview/policy.go`
- Create: `internal/preview/docker_runner.go`
- Test: `internal/preview/runner_test.go`
- Create: `deploy/preview/Dockerfile.runner`

**Interfaces:**
- Produces: `PreviewRunner.Build(context.Context, BuildRequest) (PreviewResult, error)`.
- Consumes: immutable source artifact; returns immutable static output artifact plus build/test/scan evidence.

- [ ] **Step 1: Write failing policy tests**

Assert non-root execution, no Docker socket, no host home, no secrets, read-only base, bounded CPU/memory/PIDs/disk/time, default-deny network, and output-only export.

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/preview -v`  
Expected: FAIL because preview runner is absent.

- [ ] **Step 3: Implement Docker runner command construction**

Build arguments must include:

```text
--read-only
--network=none
--cap-drop=ALL
--security-opt=no-new-privileges
--pids-limit
--memory
--cpus
--user=nonroot
```

Use a separate explicit dependency-fetch step when required; never mount deployment or GitHub credentials.

- [ ] **Step 4: Implement evidence capture**

Return sanitized build logs, test result, security scan result, source hash, output hash, and preview deployment identifier.

- [ ] **Step 5: Run tests and one fixture build**

Run:

```bash
go test -race ./internal/preview -v
docker build -f deploy/preview/Dockerfile.runner -t portfolio-preview-runner:test .
```

Expected: tests PASS and image builds.

- [ ] **Step 6: Commit**

```bash
git add internal/preview deploy/preview/Dockerfile.runner
git commit -m "feat(preview): add sandboxed static build runner"
```

### Task 15: Preview Deployment and Dual-Destination Publisher

**Files:**
- Create: `internal/deployment/preview.go`
- Create: `internal/deployment/publisher.go`
- Create: `internal/deployment/rollback.go`
- Test: `internal/deployment/publisher_test.go`

**Interfaces:**
- Produces: `PreviewDeployer`, `LabPublisher`, `PortfolioCatalogPublisher`, and `AtomicPublisher` used by Task 4.

- [ ] **Step 1: Write failing atomic publication tests**

Use fakes to prove Lab failure publishes neither destination, portfolio failure rolls back Lab visibility, retries are idempotent, and successful publication records both URLs and deployment IDs for the same hash.

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/deployment -v`  
Expected: FAIL because deployment package is absent.

- [ ] **Step 3: Implement preview and publication adapters**

Configuration:

```text
PREVIEW_PUBLIC_ORIGIN
LAB_PUBLIC_ORIGIN
PORTFOLIO_PUBLIC_ORIGIN
ARTIFACT_STORE_ROOT
```

Use immutable version paths plus a final pointer swap. Public visibility changes only after both destinations stage successfully.

- [ ] **Step 4: Run tests**

Run: `go test -race ./internal/deployment -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/deployment
git commit -m "feat(deploy): publish Lab and portfolio atomically"
```

---

## Phase 5 — Integration, Security Gates, and Operations

### Task 16: Application Composition and Configuration

**Files:**
- Create: `internal/app/config.go`
- Create: `internal/app/app.go`
- Test: `internal/app/config_test.go`
- Modify: `main.go`
- Modify: `.gitignore`
- Create: `.env.example`

**Interfaces:**
- Composes repository, artifact store, publishing service, Admin auth/API, public API, MCP server, preview runner, and deployment publisher.

- [ ] **Step 1: Write failing configuration tests**

Assert production fails closed when signing key, OAuth credentials, owner allowlist, approval verifier, or public origins are missing. Assert secrets are not logged.

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/app -v`  
Expected: FAIL because application composition is absent.

- [ ] **Step 3: Implement typed configuration and composition**

`main.go` should only parse flags/env, call `app.New`, start the server, and handle graceful shutdown. `.env.example` lists names without values.

- [ ] **Step 4: Run tests and build**

Run:

```bash
go test -race ./...
go vet ./...
go build ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app main.go .gitignore .env.example
git commit -m "refactor(app): compose publishing subsystems securely"
```

### Task 17: Full End-to-End Publication Flow

**Files:**
- Create: `e2e/publishing_flow_test.go`
- Create: `e2e/fixtures/lab-redesign/`
- Create: `scripts/run-publishing-e2e.sh`

**Interfaces:**
- Exercises all externally visible seams from MCP draft creation through Admin publication and public readback.

- [ ] **Step 1: Write failing end-to-end test**

Test sequence:

1. MCP creates draft.
2. MCP uploads fixture and deploys preview.
3. MCP requests review.
4. Admin requests changes.
5. MCP submits revision 2.
6. Admin approves revision 2 with matching hash.
7. Lab route and Design Lab API expose revision 2.
8. Revision 1 remains unavailable.
9. Audit contains every transition.
10. Repeated approval returns the original publication.

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./e2e -run TestPublishingFlow -v`  
Expected: FAIL until all composition wiring is complete.

- [ ] **Step 3: Complete wiring needed by the test**

Make only integration fixes necessary for this flow; do not add new features.

- [ ] **Step 4: Run complete verification**

Run:

```bash
go test -race ./...
go vet ./...
cd frontend && npm test -- --run && npm run build
cd .. && go test ./e2e -run TestPublishingFlow -v
```

Expected: all commands PASS.

- [ ] **Step 5: Commit**

```bash
git add e2e scripts/run-publishing-e2e.sh
git commit -m "test: verify MCP to Admin publication flow"
```

### Task 18: Security Review and Deployment Runbook

**Files:**
- Create: `docs/security/admin-mcp-lab-threat-model.md`
- Create: `docs/runbooks/admin-mcp-lab-deployment.md`
- Create: `docs/runbooks/admin-mcp-lab-rollback.md`
- Create: `deploy/Caddyfile.example`
- Create: `deploy/docker-compose.example.yml`

**Interfaces:**
- Documents operational boundaries and deployment procedures for the completed system.

- [ ] **Step 1: Write the threat model against implemented controls**

Cover prompt injection, malicious dependencies, artifact substitution, confused deputy, privilege escalation, replay, CSRF, XSS, cookie leakage, origin confusion, path traversal, archive bombs, private-repository disclosure, and audit tampering. Cite the exact implementing packages and tests.

- [ ] **Step 2: Write deployment and rollback runbooks**

Include OAuth registration, DNS/origin layout, secret provisioning, database migration, artifact storage ownership, MCP profile isolation, health checks, preview verification, publication smoke test, rollback, token rotation, and incident revocation.

- [ ] **Step 3: Add edge configuration examples**

Caddy must enforce HTTPS, origin-specific security headers, CSP, frame isolation, request-size limits, rate limits where supported, and must not forward Admin cookies to Lab/preview origins.

- [ ] **Step 4: Run documentation and configuration checks**

Run:

```bash
git diff --check
docker compose -f deploy/docker-compose.example.yml config
```

Expected: PASS.

- [ ] **Step 5: Final code review and verification**

Use `matt-pocock:code-review` against the specification, then run:

```bash
go test -race ./...
go vet ./...
cd frontend && npm test -- --run && npm run build
```

Expected: all PASS with no Critical or High review findings.

- [ ] **Step 6: Commit**

```bash
git add docs deploy
git commit -m "docs: add publishing security and operations runbooks"
```

## Plan Self-Review

- **Spec coverage:** Domain state, Admin review, MCP scope, immutable artifacts, atomic dual publication, Design Lab tab, public Lab, origin isolation, private repository handling, audit, error handling, testing, and operations each map to tasks above.
- **Placeholder scan:** No `TBD`, `TODO`, `FIXME`, or implied implementation steps remain.
- **Type consistency:** `Submission`, `Approval`, `ArtifactRef`, `Repository`, `PublishingService`, `PreviewRunner`, and publisher interfaces are defined before consumers.
- **Scope decomposition:** Five phases isolate durable registry, service/API, UI, MCP/deployment, and integration/operations. Each task ends in a testable commit.
