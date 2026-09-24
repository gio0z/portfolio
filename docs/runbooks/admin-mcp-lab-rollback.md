# Admin / MCP / Design Lab Publishing — Rollback Runbook

Date: 2026-09-13. Covers the two failure shapes of the atomic-publish design
as implemented in `internal/publishing/service.go` (`ApproveAndPublish`,
`Archive`), `internal/adminapi/handler.go` (`HandleApproveAndPublish`,
`HandleArchiveProject`), and `internal/publishing/migrations/001_registry.sql`.

Every public surface carries: `Independent redesign concept. Not affiliated with or endorsed by the original company.`

## 1. Rollback semantics (what the code guarantees)

- **Failed atomic publication is already rolled back.** If
  `AtomicPublisher.Publish` or `ArtifactStore.Promote` fails,
  `ApproveAndPublish` appends a `failed` audit event
  (`action: approve_and_publish`, `result: failed`, sanitized error) and
  returns `ErrCodePublicationFailed`. The submission stays `IN_REVIEW`,
  neither public destination becomes visible, and **no idempotency record is
  stored** — retrying with the *same* idempotency key re-executes cleanly.
- **Successful publication has no "unpublish" transition.** The state table
  (`publishing.Transition`) allows exactly one exit from `PUBLISHED`:
  `{Published → Archived, ActorOwner}`. Rollback of a bad publish = the
  `Archive` flow (§3), which appends history rather than rewriting it.
- **Re-posting approve never reverts.** A recorded idempotency key returns
  the cached `PublicationResult` (`GetIdempotencyResult`); rollback must go
  through `Archive`, never through a second approve call.
- **Registry and audit are append-only and survive every rollback.**
  `audit_events` has `audit_events_no_update` / `audit_events_no_delete`
  triggers; `submissions` keeps every revision (`PRIMARY KEY (id,
  revision)`); `Archive` bumps the revision and appends an `archive` audit
  event. Rollback adds rows; it never deletes or edits them.
- **Artifact bytes are immutable and are kept.** Published artifacts stay in
  content-addressed storage (`LocalArtifactStore`, path
  `var/portfolio/artifacts/sha256/<xx>/<sha256>`) as the permanent record
  of what was reviewed and published. Only ephemeral previews are ever
  deleted (`DeletePreview` removes `var/portfolio/previews/<hash>/`).

## 2. Case A — failed publication (submission still `IN_REVIEW`)

1. Inspect the failure (owner session cookie; reads need no `X-Request-ID`):
   ```bash
   BASE=https://<admin-host>
   curl -sSf -b admin_session.jar "$BASE/api/admin/audit?submission_id=<submission-id>&limit=5"
   curl -sSf -b admin_session.jar "$BASE/api/admin/reviews/<submission-id>"
   ```
   Expect the latest audit event with `result: failed` and a sanitized error.
2. Fix the underlying cause (publisher credentials, storage capacity,
   manifest permissions) without touching the submission.
3. Retry approve-and-publish with the **same** idempotency key and the
   **same** reviewed `artifact_sha256` (a mutated hash is correctly rejected
   with `artifact_mismatch`):
   ```bash
   curl -sSf -X POST -b admin_session.jar \
     -H "Content-Type: application/json" \
     -H "X-Request-ID: retry-1" -H "X-CSRF-Token: $CSRF" \
     -H "X-Admin-Passkey-Assertion: $ASSERTION" \
     -d '{"artifact_sha256":"<reviewed-sha256>","idempotency_key":"<original-key>"}' \
     "$BASE/api/admin/reviews/<submission-id>/approve-and-publish"
   ```
4. Confirm `state: PUBLISHED` and both destination URLs in the response, then
   run the deployment-runbook smoke checks (§8) against the new URLs.

## 3. Case B — bad publication (wrong content is live)

Only the owner may archive (`Archive` requires `ActorOwner`; the HTTP route
requires Owner + CSRF via `wrapMutation`).

1. Archive the project (accepts a project ID; the handler resolves it to the
   latest submission and archives that):
   ```bash
   curl -sSf -X POST -b admin_session.jar \
     -H "X-Request-ID: rollback-1" -H "X-CSRF-Token: $CSRF" \
     "$BASE/api/admin/projects/<project-id>/archive"
   ```
   This moves the submission `PUBLISHED → ARCHIVED` (revision bump + `archive`
   audit event) and sets the project `status` to `ARCHIVED`.
2. Confirm delisting — the public reader only serves `PUBLISHED` projects
   (`RepositoryReader.ListPublishedProjects` filters on status), so the
   archived project must vanish from all three endpoints:
   ```bash
   curl -sSf "$BASE/api/lab/projects" | grep -v "<project-slug>"
   curl -sSf "$BASE/api/lab/projects/<project-slug>"   # expect 404
   curl -sSf "$BASE/api/portfolio/design-lab" | grep -v "<project-slug>"
   ```
3. If the serving layer exposes static files from
   `var/portfolio/publications/<project-id>/` directly, remove that
   directory **after** step 1 confirms ARCHIVED. The serving layer must treat
   registry status as authoritative; a filesystem removal alone is not a
   rollback (it leaves the registry claiming PUBLISHED). Never touch
   `var/portfolio/artifacts/` — published bytes are history.
4. Clean up the ephemeral preview if it still exists (optional; previews are
   never publicly listed):
   via `LocalArtifactStore.DeletePreview(ctx, "<artifact-sha256>")` or
   `rm -rf var/portfolio/previews/<artifact-sha256>/`.
5. To re-publish corrected content, the agent prepares a new draft revision
   from `DRAFT`; the flawed revision stays `ARCHIVED` as the audit trail.
   The corrected publish uses a **new** idempotency key (the old key is
   bound to the archived revision's result).

## 4. What rollback preserves

- Full submission revision history (`submissions` rows, old and new).
- Every audit event, including the `failed` attempts and the `archive` event
  with actor identity, request ID, artifact hash, and deployment IDs/URLs.
- The `approvals` and `idempotency_results` rows of the rolled-back publish.
- The published artifact bytes under their SHA-256 address.

## 5. Last-resort registry restore (data corruption, not content rollback)

1. Stop the server (single writer; `MaxOpenConns(1)`).
2. Export current audit first: `sqlite3 <registry> "SELECT event_json FROM
   audit_events ORDER BY sequence;" > audit-salvage.jsonl` — restores lose
   every event written after the backup.
3. Copy the backup registry file over the live path and restart; the
   `001_registry.sql` migration is idempotent.
4. Re-verify smoke checks and confirm `audit_events` continuity from the
   salvage file. Prefer Case A/B above whenever the registry itself is
   healthy — file restore is for a corrupt database, not a bad publish.

## Ruling: rollback appends history instead of rewriting it — the triggers and revisioned rows make destructive rollback impossible by construction — cost if wrong: a "revert" that silently erases the evidence of what was live.
