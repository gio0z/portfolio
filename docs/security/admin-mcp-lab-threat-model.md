# Admin / MCP / Design Lab Publishing — Threat Model

Date: 2026-09-13. Scope: the publishing system as implemented in
`internal/adminauth`, `internal/adminapi`, `internal/publishing`,
`internal/publicapi`, and `internal/preview`. The MCP adapter is under
construction in `internal/mcppublisher` (contract pinned by its tests);
the server-side guarantees below hold regardless of MCP behavior.

Binding spec: `docs/superpowers/specs/2026-09-03-admin-mcp-lab-publishing-design.md`.

Public-label invariant: every third-party redesign shown publicly carries
`Independent redesign concept. Not affiliated with or endorsed by the original company.`
Enforced in code by `publishing.MandatoryDisclaimer`,
`publishingService.ApproveAndPublish` (rejects projects whose disclaimer
omits it), and `publicapi.cleanPublicProject` (prepends it when missing).

## 1. Assets

| Asset | Invariant | Enforcing symbols |
|---|---|---|
| Artifact bytes | Content-addressed by SHA-256; bytes immutable after `Put`; reads re-hash and fail closed on mismatch | `publishing.LocalArtifactStore.Put/Open/Promote`, `publishing.ValidateAndHash`, `publishing.ErrArtifactCorrupt`, `Open` re-hash compare |
| Reviewed hash binding | Approval is bound to `submission_revision + artifact_sha256`; any content change invalidates review | `publishingService.ApproveAndPublish` hash-equality check, `publishing.NewArtifactMismatchError`, `approvals` table FK on `(submission_id, revision, artifact_sha256)` |
| Admin session | Server-side, HMAC-signed, owner-allowlisted, revocable | `adminauth.Service.CreateSession`, `signSessionToken/verifySessionToken`, `MemorySessionStore`, `OwnerMiddleware`, `HandleLogout` |
| CSRF token | Per-session HMAC, constant-time validated on all mutations | `adminauth.Service.IssueCSRFToken/ValidateCSRFToken`, `CSRFMiddleware`, `IsStateChangingMethod` |
| OAuth flow state | Single-use state + PKCE S256, constant-time comparison | `adminauth.GeneratePKCE`, `HandleLogin/HandleCallback`, `OAuthStateCookieName`, `OAuthPKCECookieName` |
| Step-up proof | Second factor required for approve-and-publish; dev bypass refuses production | `adminauth.ApprovalVerifier`, `RequireStepUp`, `DevApprovalVerifier` (+ `ErrDevVerifierInProduction`), `PasskeyApprovalVerifier` (`X-Admin-Passkey-Assertion`) |
| Audit trail | Append-only; updates/deletes aborted at the DB layer | `audit_events_no_update` / `audit_events_no_delete` triggers in `migrations/001_registry.sql`, `SQLiteRepository.AppendAudit/WithTx` |
| MCP credentials | Short-lived, scoped, revocable; forbidden scopes unissuable | `internal/mcppublisher` auth tests (`TestIssueRejectsForbiddenScopes`, `TestAuthRejectsTokenCarryingForbiddenScope`, `TestAuthRejectsRevokedToken`, `TestAuthRejectsExpiredToken`) |
| Public surface | Sanitized read-only projection; no identities, preview URLs, or build internals | `publicapi.PublicLabProject`, `cleanPublicProject`, `RepositoryReader` (published-only queries) |

## 2. Trust boundaries

1. **Agent/MCP stops at `IN_REVIEW`.** `publishing.Transition` permits an
   `ActorAgent` only up to `{PreviewReady → InReview}`; every transition out
   of `InReview` requires `ActorOwner` (`Approved`, `ChangesRequested`,
   `Rejected`), and `{Approved → Published}` requires `ActorSystem`. The
   MCP tool catalog excludes approve/publish/permission/token/deletion tools
   (`TestToolCatalogExcludesForbiddenTools`, `TestApprovePublishAbsentForAgentIdentity`,
   `TestRequestReviewStopsAtInReview`).
2. **Admin-only approve/publish.** `adminapi.RegisterRoutes` wraps
   `POST /api/admin/reviews/{id}/approve-and-publish` in
   `OwnerMiddleware → CSRFMiddleware → RequireStepUp`
   (`wrapStepUp`); all other mutations in Owner + CSRF (`wrapMutation`);
   reads in Owner (`wrapOwner`). A nil `authSvc` fails closed to 401.
3. **Publishing service is the sole authority.** State transitions, hash
   verification, idempotency, atomic publication, and audit recording all
   live in `publishingService`; neither the MCP adapter nor the Admin UI
   writes the registry, object storage, or public data directly.
4. **Origin isolation.** The admin session cookie is host-only, `HttpOnly`,
   `Secure`, `SameSite=Strict` (`SetSessionCookie`); the CSRF cookie is
   host-only `SameSite=Strict` and intentionally JS-readable so the Admin UI
   can attach `X-CSRF-Token`. OAuth cookies are `HttpOnly`,
   `SameSite=Lax`, 10-minute TTL, and cleared on callback. Admin cookies
   must never be forwarded to Lab/preview origins (edge responsibility;
   see deployment runbook).
5. **Preview sandbox.** Builds run under `preview.Policy` (non-root,
   read-only root FS, default-deny network, CPU/memory/PID/time/size caps,
   validated by `Policy.Validate`), static output allowlisted by
   `staticExtensions` (≤ `maxOutputFiles` files), published atomically under
   `var/portfolio/previews/<output-hash>/`; failures leave no partial
   output. Build logs are secret-redacted (`keyValuePattern`).

## 3. Enumerated threats → mitigations (traced to symbols)

1. **Artifact substitution (review one hash, publish another).** Mitigated:
   `ApproveAndPublish` requires `input.ArtifactSHA256 == sub.ArtifactSHA256`
   (`ErrCodeArtifactMismatch`); `Open` re-hashes bytes on read
   (`ErrArtifactCorrupt`); `Promote` refuses unknown hashes (`ErrNotFound`);
   `RecordApproval` FK-pins `(submission, revision, artifact_sha256)`.
   Tests: artifact-mismatch and corrupt-artifact cases in
   `internal/publishing/service_test.go`, `artifacts_test.go`.
2. **Confused deputy (agent drives approval).** Mitigated: actor-kind gates
   in `ApproveAndPublish` (`NewForbiddenError("only owner may approve and
   publish")`), the `Transition` table, and `actorFromContext` deriving
   `ActorOwner` only from a session placed in context by `OwnerMiddleware`.
   MCP scopes `lab:approve`, `lab:publish`, `admin:*` are unissuable and
   rejected at authentication even if forged into a token.
3. **Privilege escalation (non-owner session).** Mitigated: `OwnerMiddleware`
   verifies the HMAC signature (constant-time), requires a live server-side
   session, and enforces `strings.EqualFold(sess.Login, cfg.AllowedLogin)`
   (default `gio0z`); `HandleCallback` re-checks the allowlist after
   fetching the GitHub identity. Tests: `TestOAuth_AllowlistedUserAccepted`,
   `TestOAuth_NonAllowlistedUserRejected`.
4. **OAuth replay / code interception / CSRF-on-login.** Mitigated:
   128-bit `state` + PKCE S256 (`GeneratePKCE`, `GenerateRandomString`),
   constant-time state comparison, single-use cookies cleared before token
   exchange, `redirect_uri` pinned to `ADMIN_PUBLIC_ORIGIN`
   (`getCallbackURL`), 15 s callback timeout, 10-minute OAuth cookie TTL.
   Test: `TestOAuth_PKCEAndStateValidation`.
5. **Session replay after logout / expiry.** Mitigated: server-side store —
   `HandleLogout` (POST-only) deletes the session and clears the cookie;
   `MemorySessionStore.Get` lazily expires (`ErrSessionExpired`) and
   `OwnerMiddleware` rejects unknown sessions. Operational note: the store
   is in-process, so sessions do not survive restarts and are not shared
   across replicas — run single-replica or provide a shared
   `SessionStore` (see deployment runbook). Tests: `TestSession_Expiration`,
   `TestSession_StatusAndLogout`.
6. **CSRF against admin mutations.** Mitigated: `CSRFMiddleware` on every
   state-changing method (`IsStateChangingMethod`), token = HMAC
   `("csrf:"+sessionID)` compared in constant time; unauthenticated CSRF
   checks fail closed (`forbidden: unauthenticated csrf check`). The
   approve-and-publish path additionally requires step-up. Test:
   `TestCSRF_Validation`.
7. **Step-up bypass in production.** Mitigated: `NewService` refuses a nil
   verifier (`ErrMissingApprovalVerifier`) and refuses the dev verifier in
   production (`ErrDevVerifierInProduction`); `DevApprovalVerifier` requires
   `X-Admin-StepUp-Dev: true` and only outside production;
   `RequireStepUp` fails closed when no verifier is configured. Production
   uses `PasskeyApprovalVerifier` (`X-Admin-Passkey-Assertion`).
   Tests: `TestStepUp_ApprovalVerification`,
   `TestConfig_ProductionFailsClosedWithoutApprovalVerifier`.
8. **Missing-secrets startup.** Mitigated: `NewService` fails closed on
   missing `ADMIN_GITHUB_CLIENT_ID` / `ADMIN_GITHUB_CLIENT_SECRET` /
   `ADMIN_SESSION_SIGNING_KEY`; `main.go` calls `log.Fatalf` in production
   when auth construction fails. Test: `TestConfig_MissingSecretsFailsClosed`.
9. **Malicious dependencies / archive bombs / path traversal / MIME
   spoofing / scripted SVG.** Mitigated: `ValidateAndHash` enforces the
   `allowedMIMETypes` allowlist on *detected* (not declared) content,
   rejects MIME mismatches (`ErrMIMEMismatch`), executables, oversize
   payloads (`ErrAssetTooLarge`, 32 MiB default), unsafe zip entries
   (`validateStaticArchive`: traversal/symlink rejection), and scripted
   SVGs; `Promote` rejects `..`/absolute/empty paths (`ErrUnsafePath`);
   `decodeJSON` caps bodies at 1 MiB and rejects unknown fields.
10. **Publishing unsafe content.** Mitigated: `RequestReview` and
    `ApproveAndPublish` both require `SecurityScanResult == "passed"`
    (`NewScanFailedError`), plus build/test `"passed"` at review request;
    `HandleOverview` surfaces non-`passed` results as build failures.
11. **Double-publish / replayed approval.** Mitigated: `idempotency_key`
    required (body or `Idempotency-Key` header); `GetIdempotencyResult`
    returns the cached `PublicationResult` for repeats;
    `RecordIdempotencyResult` + `WithTx` make first-write-wins atomic
    (`ErrDuplicateIdempotencyKey` → `ErrCodeSlugConflict`).
12. **Partial publication (Lab visible, portfolio not, or vice versa).**
    Mitigated: `ApproveAndPublish` promotes the artifact, calls
    `AtomicPublisher.Publish`, then commits approval + submission update +
    idempotency record + audit in one `WithTx` transaction; publisher
    failure appends a `failed` audit event and returns
    `ErrCodePublicationFailed` with both destinations hidden. Retry uses the
    same idempotency key.
13. **Audit tampering.** Mitigated: SQLite triggers abort UPDATE/DELETE on
    `audit_events`; `WithTx` commits audit rows in the same transaction as
    the state change; `cleanAuditSummary`/`cleanErrorString` and
    `sanitizeServiceError` keep paths, tokens, secrets, and stack traces out
    of admin responses; preview logs are secret-redacted.
14. **XSS via published Lab content.** Mitigated in depth: scripted SVGs
    rejected at ingest; preview output restricted to static extensions;
    public projection sanitized (`cleanPublicProject`); executable Lab
    projects run in sandboxed iframes per spec (edge/CSP responsibility —
    see deployment runbook).
15. **Private-repository disclosure.** Design rule (spec §GitHub Rules):
    private sources excluded by default, metadata-only only with explicit
    owner configuration; MCP contract test
    `TestPrivateGitHubReposExcludedByDefault`; public API can only read
    `PUBLISHED` projections. No private-source handling exists in
    `internal/publishing` itself — disclosure control lives in the
    GitHub-discovery/MCP layer, so changes there must preserve the default.
16. **Prompt injection via agent-supplied metadata.** Contained:
    agent input is data, never instructions — drafts/updates pass through
    schema validation (`CreateDraft`/`UpdateDraft`), slug rules, mandatory
    disclaimer enforcement, and owner review of the exact artifact before
    anything becomes public; state-regression (`UpdateDraft` resets to
    `Draft`) prevents post-review mutation without re-review.

## 4. Residual risks (accepted, must be owned operationally)

- **Rate limiting** on login/review/publication endpoints is spec-mandated
  but has no in-code implementation — enforce at the edge (see deployment
  runbook) until an in-app limiter lands.
- **Malware scanning** beyond executable/MIME/archive/SVG heuristics is not
  implemented — treat `SecurityScanResult` provenance as trusted-runner-only
  and keep the preview sandbox as the containment boundary.
- **Multi-replica sessions** require a shared `SessionStore`; the default
  is in-memory (see §3.5).
- **CSP/frame-isolation/noindex on preview and Lab routes** are edge
  responsibilities (see deployment runbook); the backend provides the
  isolation primitives (separate origins, no credential sharing).

## Ruling: threat model traces to implemented symbols, residual risks named explicitly — silent gaps would be worse than documented ones — cost if wrong: operators assume protections that do not exist.
