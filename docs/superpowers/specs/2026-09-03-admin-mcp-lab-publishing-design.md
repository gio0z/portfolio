# Admin, MCP, and Design Lab Publishing Architecture

**Date:** 2026-09-03  
**Status:** Proposed for implementation review

## Purpose

Allow an internal AI agent to create or redesign an application, deploy a safe preview, and submit it for review. The owner reviews the exact immutable artifact in an Admin page. One approval publishes that artifact to both the Lab and the public portfolio's Design Lab tab.

The system must keep publishing authority with the owner while allowing the agent to automate preparation, validation, preview deployment, metadata entry, and revision handling.

## Product Domains

### Engineering Work

Production engineering projects sourced from repositories owned by `gio0z`. Repository visibility does not imply publication permission. Private repositories are excluded by default unless the owner explicitly enables metadata-only publication.

### Lab Project

An independent redesign, prototype, or interface experiment. Every third-party redesign must carry the public label:

> Independent redesign concept. Not affiliated with or endorsed by the original company.

### Submission

A proposed Lab Project revision prepared by an agent. A submission references one immutable artifact hash and progresses through the publishing state machine.

### Preview

A sandboxed deployment of the exact artifact under review. A preview is not publicly listed in the Lab or portfolio.

### Approval

An owner decision bound to one submission revision and artifact hash. Any content or artifact change invalidates the approval.

### Publication

The atomic operation that exposes an approved artifact below the configured `LAB_PUBLIC_ORIGIN` and adds its metadata to the public portfolio Design Lab tab.

## Architecture

```text
Internal AI Agent
      |
      | scoped MCP tools
      v
Portfolio Publisher MCP
      |
      | authenticated Publishing API calls
      v
Publishing API <-------------------- Admin Web App
      |                                  |
      | validates submission             | review / request changes /
      | and controls state                | reject / approve
      v                                  |
Draft Registry + Artifact Store --------+
      |
      | sandboxed build and preview
      v
Preview Runtime
      |
      | approved artifact hash only
      v
Atomic Publication
      |---------------------> LAB_PUBLIC_ORIGIN/<slug>
      +---------------------> Portfolio: Design Lab tab
```

The Admin app and MCP server are separate clients of one Publishing API. Neither client writes directly to the registry, object storage, deployment system, or public portfolio data.

## Components

### Publishing API

The sole authority for:

- Submission schema validation
- State transitions
- Ownership and scope checks
- Slug reservation
- Artifact hash verification
- Approval validation
- Publication and rollback orchestration
- Audit recording

All mutating requests require authentication, authorization, an idempotency key, and a request ID.

### Portfolio Publisher MCP

A narrow adapter for internal agents. It provides domain-specific tools and has no shell, arbitrary filesystem, raw SQL, secret management, or unrestricted deployment tools.

Agent-accessible tools:

```text
list_github_projects
get_github_project
list_lab_projects
get_lab_project
create_lab_draft
update_lab_draft
upload_lab_asset
deploy_lab_preview
get_deployment_status
request_review
get_review_feedback
archive_lab_draft
```

The MCP server does not expose approval, publication, permission, token, or permanent-deletion tools.

Server-initiated MCP sampling is disabled.

### Admin Web App

Routes:

```text
/admin
/admin/reviews
/admin/engineering
/admin/lab
/admin/deployments
/admin/media
/admin/audit
/admin/settings
```

The Admin app is the only user-facing client allowed to approve publication.

### Review Queue

The primary Admin workflow. Each review displays:

- Lab Project title and original product
- Required independent-concept disclaimer
- Sandboxed interactive preview
- Desktop and mobile viewport controls
- Before/after screenshots
- Submission summary and redesign goals
- Technology stack
- Build, test, and security-scan status
- Artifact hash
- Portfolio card preview
- Lab detail-page preview
- Agent notes and prior review feedback

Available decisions:

- **Approve & Publish**
- **Request Changes**
- **Reject**
- **Archive**

### Preview and Lab Runtime

Preview deployments use an isolated origin and are not indexed. Published Lab projects use paths below the configured `LAB_PUBLIC_ORIGIN`; experiments that execute untrusted JavaScript receive stronger per-project origin isolation.

Portfolio Admin cookies are host-only and are never sent to Lab or preview origins.

## Publishing State Machine

```text
DRAFT
  -> BUILDING
  -> PREVIEW_READY
  -> IN_REVIEW
  -> APPROVED
  -> PUBLISHED
  -> ARCHIVED
```

Failure branches:

```text
BUILDING -> BUILD_FAILED -> DRAFT
PREVIEW_READY -> DRAFT       (agent revision)
IN_REVIEW -> CHANGES_REQUESTED -> DRAFT
IN_REVIEW -> REJECTED
PUBLISHED -> ARCHIVED
```

Rules:

1. An agent may progress a submission only as far as `IN_REVIEW`.
2. Only an authenticated owner session may transition `IN_REVIEW` to `APPROVED`.
3. Approval is bound to `submission_revision + artifact_sha256`.
4. Any mutation after review invalidates approval and returns the submission to `DRAFT`.
5. `Approve & Publish` validates and publishes the same artifact atomically.
6. Repeated approval requests with the same idempotency key return the first result and do not create duplicate deployments.
7. A failed atomic publication does not expose either destination; it records a recoverable failure for retry.

## Approve & Publish Transaction

Endpoint concept:

```text
POST /api/admin/reviews/{submission_id}/approve-and-publish
```

Required checks:

- Owner authentication and authorization pass
- CSRF token passes
- Submission is `IN_REVIEW`
- Required disclaimer is present
- Build, tests, and security scan pass
- Requested artifact hash equals the reviewed artifact hash
- Preview deployment serves that artifact hash
- Slug is reserved for the same Lab Project
- Idempotency key is valid

Atomic outcome:

1. Record approval identity, revision, and artifact hash.
2. Promote the immutable artifact to the published Lab route.
3. Publish the Design Lab metadata record consumed by the portfolio.
4. Record both destination URLs and deployment identifiers.
5. Mark submission `PUBLISHED`.
6. Append an immutable audit event.

If any step fails, neither public destination becomes visible.

## Public Portfolio Information Architecture

The existing section remains titled:

> Featured Engineering Work

It gains two views:

```text
[ Engineering Work ] [ Design Lab ]
```

### Engineering Work view

- Data source: explicitly selected GitHub repositories owned by `gio0z`
- Card metadata: repository, architecture, language/stack, impact, and repository link
- Existing layered coverflow presentation remains available

### Design Lab view

- Data source: published Lab registry only
- Card metadata: original product, redesign focus, target platform, status, cover image, disclaimer, and case-study link
- Homepage shows a curated subset plus **Explore Design Lab**
- Full catalog lives on the Lab subdomain

Switching views changes the content model, copy, and action labels; it does not misrepresent a redesign as commissioned or production engineering work.

## Data Model

### Lab Project

```json
{
  "id": "lab_...",
  "slug": "product-redesign",
  "title": "Product Redesign",
  "original_product": "Product Name",
  "disclaimer": "Independent redesign concept. Not affiliated with or endorsed by the original company.",
  "focus": ["navigation", "information architecture"],
  "platforms": ["web", "mobile"],
  "status": "draft",
  "featured": false,
  "created_at": "ISO-8601",
  "updated_at": "ISO-8601"
}
```

### Submission Revision

```json
{
  "id": "submission_...",
  "lab_project_id": "lab_...",
  "revision": 3,
  "state": "IN_REVIEW",
  "artifact_sha256": "hex-encoded SHA-256",
  "preview_url": "https://preview...",
  "build_result": "passed",
  "test_result": "passed",
  "security_scan_result": "passed",
  "portfolio_metadata": {},
  "submitted_by": "agent identity",
  "submitted_at": "ISO-8601"
}
```

### Approval

```json
{
  "submission_id": "submission_...",
  "revision": 3,
  "artifact_sha256": "hex-encoded SHA-256",
  "approved_by": "owner identity",
  "approved_at": "ISO-8601",
  "idempotency_key": "opaque unique value"
}
```

## Authentication and Authorization

### Admin

- GitHub OAuth identity allowlisted to the owner account `gio0z`
- Passkey or equivalent MFA required for **Approve & Publish**
- Session cookie is `HttpOnly`, `Secure`, `SameSite=Strict`, and host-only
- CSRF protection on every state-changing request
- Login, review decisions, and publication endpoints are rate-limited

### MCP

- Enabled only in an internal Hermes profile
- Disabled for public/customer-service/group profiles
- Separate token per agent/profile
- HTTPS remote transport or locally controlled stdio transport
- Short-lived, revocable credentials with least-privilege scopes
- Environment inheritance remains filtered; only the publisher credential is supplied

Agent scopes:

```text
portfolio:read
lab:draft:create
lab:draft:update
lab:asset:upload
lab:preview:deploy
lab:review:request
```

No `lab:approve`, `lab:publish`, `admin:*`, secret-management, or arbitrary-execution scopes.

## Artifact and Build Security

Builds run in disposable sandbox containers with:

- Non-root user
- Read-only base filesystem
- No host home mount
- No Docker socket
- No GitHub or deployment credentials
- CPU, memory, process, disk, and timeout limits
- Default-deny outbound network policy
- Explicit dependency-fetch phase when required
- Static build output as the only export

Asset controls:

- MIME validation based on content
- File count and size limits
- Path traversal and symlink rejection
- Archive inspection before extraction
- SVG sanitization or rejection of active SVG
- Malware scan
- SHA-256 hashing
- Content Security Policy on previews and published Lab projects

## Origin Isolation

- Admin, portfolio, preview, and executable Lab projects do not share privileged cookies.
- The Lab cannot make credentialed cross-origin requests to Admin APIs.
- Third-party redesign previews run inside a sandboxed iframe.
- `allow-same-origin` is withheld unless a reviewed project explicitly requires it.
- Preview routes use `noindex` and unguessable identifiers.

## GitHub Rules

- Repository discovery uses a read-only GitHub App or fine-grained token.
- Private repositories are excluded from public output by default.
- Private-source publication requires explicit owner configuration and defaults to metadata-only.
- Repository synchronization never grants the MCP publisher write access to arbitrary repositories.

## Audit Trail

Every mutation records:

```text
timestamp
request ID
actor and profile
tool or endpoint
target project/submission
previous state
new state
artifact hash
approval identity
deployment IDs and URLs
result or sanitized error
```

Audit events are append-only. Secrets and bearer tokens are redacted from errors and logs.

## Error Handling

- Validation errors are field-specific and do not mutate state.
- Duplicate idempotency keys return the original operation result.
- Build and preview failures retain logs without exposing secrets.
- Publishing failure leaves both public destinations hidden and creates a retryable Admin action.
- MCP clients receive stable machine-readable error codes suitable for agent correction.
- Admin review displays actionable failures without raw credentials, filesystem paths, or internal stack traces.

## Testing Strategy

### Publishing API

- State-transition table tests
- Authorization and scope tests
- Artifact-hash mismatch tests
- Idempotency tests
- Atomic publication rollback tests
- Private-repository disclosure tests
- Audit event completeness tests

### MCP Adapter

- Tool-schema contract tests
- Confirmation that forbidden tools are absent
- Scope enforcement tests
- Error-code translation tests
- Sampling-disabled verification

### Admin App

- Review queue rendering
- Preview sandbox attributes
- CSRF enforcement
- Owner allowlist enforcement
- Approval invalidation after mutation
- Approve-and-publish happy path and failure states

### End-to-End

1. Agent creates a draft.
2. Agent uploads assets and deploys preview.
3. Agent requests review.
4. Admin requests changes; agent submits a new revision.
5. Admin approves the new artifact.
6. The same artifact appears at the Lab route and Design Lab tab.
7. Audit records every transition.
8. Rollback removes public visibility consistently.

## Acceptance Criteria

- Agent can prepare and submit a redesign without receiving publish authority.
- Admin can review the exact artifact, evidence, public metadata, and disclaimer.
- One owner approval publishes the identical artifact to Lab and the Design Lab tab.
- Any post-review mutation invalidates approval.
- Publishing is idempotent and atomic across both public destinations.
- Private GitHub repositories are not disclosed by default.
- Lab/preview code cannot access Admin cookies or privileged APIs.
- Every state mutation and deployment is auditable.
- Public engineering work remains distinct from independent redesign experiments.
