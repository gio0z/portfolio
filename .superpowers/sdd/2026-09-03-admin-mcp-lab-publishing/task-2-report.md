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
