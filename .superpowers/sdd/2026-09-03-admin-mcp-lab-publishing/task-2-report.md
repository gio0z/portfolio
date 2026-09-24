# Task 2 Report — Repository Contract and SQLite Registry

## Status

Complete. Added the publishing repository contract, durable SQLite implementation, embedded migration, repository integration tests, and the `modernc.org/sqlite` dependency.

## Changes

- Added `internal/publishing/repository.go`:
  - Exact `Repository` interface required by the task.
  - `ProjectFilter`, `OperationResult`, and `AuditEvent` repository boundary types.
  - Sentinel errors `ErrNotFound`, `ErrConflict`, and `ErrDuplicateIdempotencyKey` for `errors.Is` matching.
- Added `internal/publishing/sqlite_repository.go`:
  - `OpenRegistry` opens and migrates a registry with foreign keys and busy timeout enabled.
  - `NewSQLiteRepository` constructor.
  - Parameterized create/get/list/update, approval, idempotency, audit, and transaction methods.
  - Optimistic submission revision checks require both the expected current revision and the next consecutive revision.
  - Structured project/submission/result/audit data is JSON encoded where appropriate.
- Added `internal/publishing/migrations/001_registry.sql`:
  - Tables: `lab_projects`, `submissions`, `approvals`, `idempotency_results`, `audit_events`, `publications`.
  - Unique project slug, submission revision, approval artifact-binding FK, approval/idempotency keys, and publication revision constraints.
  - Audit update/delete triggers enforce append-only storage.
- Added `internal/publishing/sqlite_repository_test.go` with real temporary SQLite databases covering persistence/filtering, not-found classification, optimistic conflicts, state/revision persistence, idempotency retrieval/duplicates, approval hash binding, audit ordering/immutability, and transaction rollback.
- Updated `go.mod` and `go.sum` using `go get modernc.org/sqlite@latest` followed by `go mod tidy`. Latest SQLite (`v1.58.0`) requires Go 1.25, so the module Go directive was upgraded from 1.23 to 1.25 by the Go tool.

## TDD Evidence

### Initial RED

Command:

```text
go test ./internal/publishing -run SQLite -v
```

Result: failed to compile as expected because `SQLiteRepository`, `OpenRegistry`, `NewSQLiteRepository`, repository types, and typed errors did not yet exist.

### Self-review RED/GREEN

During self-review, added a revision-gap assertion. It initially failed:

```text
revision-skipping UpdateSubmission() error = <nil>, want ErrConflict
FAIL
```

Changed optimistic update validation to require `submission.Revision == expectedRevision + 1`; focused and full tests then passed.

## Verification

- `go test ./internal/publishing -run SQLite -v` — PASS, 6 SQLite integration tests.
- `go test ./...` — PASS (`internal/publishing`, `pkg/api`; root has no tests).
- `go test -race ./internal/publishing -v` — PASS, including all SQLite and state-machine tests.
- `go vet ./...` — PASS.
- `git diff --check` — PASS.

## Self-review

Checked every brief requirement against the implementation. Queries use placeholders for values, Task 1 domain types are reused unchanged, all specified tables and constraints exist, errors support `errors.Is`, audit rows cannot be updated or deleted, and transaction rollback is exercised. No known functional gaps for Task 2.

## Concerns

- `modernc.org/sqlite@latest` currently forces Go 1.25. This follows the brief's exact dependency command but raises the repository's minimum Go version from 1.23.

## Fix Round — Go 1.23 Compatibility and Audit Test

### Changes

- Restored the module directive to `go 1.23.0`.
- Pinned `modernc.org/sqlite v1.38.2` and regenerated `go.sum` with its Go 1.23-compatible transitive dependency graph.
- Corrected the audit immutability test to update the real `request_id` column, ensuring the append-only trigger—not a nonexistent-column error—rejects the mutation.

### Verification

Installed and used Go 1.23.12 to verify the requested compatibility target.

```text
$ /home/gio/go/bin/go1.23.12 test ./internal/publishing -run SQLite -v
=== RUN   TestSQLiteProjectPersistenceAndFiltering
--- PASS: TestSQLiteProjectPersistenceAndFiltering (0.00s)
=== RUN   TestSQLiteSubmissionRevisionConflictAndStatePersistence
--- PASS: TestSQLiteSubmissionRevisionConflictAndStatePersistence (0.00s)
=== RUN   TestSQLiteIdempotencyLookupAndDuplicate
--- PASS: TestSQLiteIdempotencyLookupAndDuplicate (0.00s)
=== RUN   TestSQLiteApprovalBindsSubmissionRevisionAndArtifactHash
--- PASS: TestSQLiteApprovalBindsSubmissionRevisionAndArtifactHash (0.00s)
=== RUN   TestSQLiteAuditIsAppendOnlyAndOrdered
--- PASS: TestSQLiteAuditIsAppendOnlyAndOrdered (0.00s)
=== RUN   TestSQLiteWithTxRollsBackAllMutations
--- PASS: TestSQLiteWithTxRollsBackAllMutations (0.00s)
PASS
ok  portfolio/internal/publishing  0.014s

$ /home/gio/go/bin/go1.23.12 test -race ./internal/publishing
ok  portfolio/internal/publishing  1.116s

$ /home/gio/go/bin/go1.23.12 test ./...
?   portfolio  [no test files]
ok  portfolio/internal/publishing  0.015s
ok  portfolio/pkg/api  0.003s

$ /home/gio/go/bin/go1.23.12 vet ./...
[no output; exit 0]
```

### Changed Files

- `go.mod`
- `go.sum`
- `internal/publishing/sqlite_repository_test.go`
- `.superpowers/sdd/2026-09-03-admin-mcp-lab-publishing/task-2-report.md`

### Concerns

None. The selected SQLite release and every module in the resolved build list declare Go 1.23 or earlier.
