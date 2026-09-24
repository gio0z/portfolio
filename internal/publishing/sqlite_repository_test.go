package publishing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func openSQLiteTestRepository(t *testing.T) (*sql.DB, *SQLiteRepository) {
	t.Helper()
	db, err := OpenRegistry(filepath.Join(t.TempDir(), "registry.db"))
	if err != nil {
		t.Fatalf("OpenRegistry() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, NewSQLiteRepository(db)
}

func testLabProject() LabProject {
	created := time.Date(2026, 9, 3, 8, 30, 0, 0, time.UTC)
	return LabProject{
		ID: "project-1", Slug: "mail-redesign", Title: "Mail Redesign",
		OriginalProduct: "Mail", Disclaimer: "Independent redesign concept.",
		Focus: []string{"accessibility", "speed"}, Platforms: []string{"web", "mobile"},
		Status: "draft", Featured: true, CreatedAt: created, UpdatedAt: created.Add(time.Minute),
	}
}

func testSubmission(revision int64, state SubmissionState, hash string) Submission {
	submitted := time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC)
	return Submission{
		ID: "submission-1", LabProjectID: "project-1", Revision: revision, State: state,
		ArtifactSHA256: hash, PreviewURL: "https://preview.example/submission-1",
		BuildResult: "passed", TestResult: "passed", SecurityScanResult: "passed",
		PortfolioMetadata: PortfolioMetadata{"summary": "A faster mail client", "score": float64(9)},
		SubmittedBy:       "agent:test", SubmittedAt: submitted, UpdatedAt: submitted.Add(time.Minute),
	}
}

func TestSQLiteProjectPersistenceAndFiltering(t *testing.T) {
	_, repo := openSQLiteTestRepository(t)
	ctx := context.Background()
	want := testLabProject()
	if err := repo.CreateLabProject(ctx, want); err != nil {
		t.Fatalf("CreateLabProject() error = %v", err)
	}

	got, err := repo.GetLabProject(ctx, want.ID)
	if err != nil {
		t.Fatalf("GetLabProject() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetLabProject() = %#v, want %#v", got, want)
	}
	projects, err := repo.ListLabProjects(ctx, ProjectFilter{Status: "draft", Featured: boolPointer(true)})
	if err != nil {
		t.Fatalf("ListLabProjects() error = %v", err)
	}
	if len(projects) != 1 || projects[0].ID != want.ID {
		t.Fatalf("ListLabProjects() = %#v, want project-1", projects)
	}
	if _, err := repo.GetLabProject(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetLabProject(missing) error = %v, want ErrNotFound", err)
	}

	// Update project
	want.Title = "Updated Title"
	want.UpdatedAt = want.UpdatedAt.Add(time.Minute)
	if err := repo.UpdateLabProject(ctx, want); err != nil {
		t.Fatalf("UpdateLabProject() error = %v", err)
	}
	gotUpdated, err := repo.GetLabProject(ctx, want.ID)
	if err != nil {
		t.Fatalf("GetLabProject() after update error = %v", err)
	}
	if gotUpdated.Title != "Updated Title" {
		t.Fatalf("gotUpdated.Title = %q, want %q", gotUpdated.Title, "Updated Title")
	}
	missingProj := want
	missingProj.ID = "non-existent"
	if err := repo.UpdateLabProject(ctx, missingProj); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateLabProject(non-existent) error = %v, want ErrNotFound", err)
	}
}

func TestSQLiteSubmissionRevisionConflictAndStatePersistence(t *testing.T) {
	_, repo := openSQLiteTestRepository(t)
	ctx := context.Background()
	if err := repo.CreateLabProject(ctx, testLabProject()); err != nil {
		t.Fatal(err)
	}
	first := testSubmission(1, Draft, "hash-one")
	if err := repo.CreateSubmission(ctx, first); err != nil {
		t.Fatalf("CreateSubmission() error = %v", err)
	}
	second := testSubmission(2, InReview, "hash-two")
	if err := repo.UpdateSubmission(ctx, second, 1); err != nil {
		t.Fatalf("UpdateSubmission() error = %v", err)
	}
	if err := repo.UpdateSubmission(ctx, testSubmission(3, Approved, "hash-three"), 1); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale UpdateSubmission() error = %v, want ErrConflict", err)
	}
	if err := repo.UpdateSubmission(ctx, testSubmission(4, Approved, "hash-four"), 2); !errors.Is(err, ErrConflict) {
		t.Fatalf("revision-skipping UpdateSubmission() error = %v, want ErrConflict", err)
	}
	got, err := repo.GetSubmission(ctx, first.ID)
	if err != nil {
		t.Fatalf("GetSubmission() error = %v", err)
	}
	if got.Revision != 2 || got.State != InReview || got.ArtifactSHA256 != "hash-two" {
		t.Fatalf("GetSubmission() = %#v, want revision 2 in review", got)
	}
}

func TestSQLiteIdempotencyLookupAndDuplicate(t *testing.T) {
	_, repo := openSQLiteTestRepository(t)
	ctx := context.Background()
	if got, err := repo.GetIdempotencyResult(ctx, "request-1"); err != nil || got != nil {
		t.Fatalf("missing GetIdempotencyResult() = (%#v, %v), want (nil, nil)", got, err)
	}
	want := OperationResult{Code: "published", ResourceID: "submission-1", Data: json.RawMessage(`{"lab_url":"https://lab.example/mail"}`)}
	if err := repo.RecordIdempotencyResult(ctx, "request-1", want); err != nil {
		t.Fatalf("RecordIdempotencyResult() error = %v", err)
	}
	got, err := repo.GetIdempotencyResult(ctx, "request-1")
	if err != nil || got == nil || !reflect.DeepEqual(*got, want) {
		t.Fatalf("GetIdempotencyResult() = (%#v, %v), want %#v", got, err, want)
	}
	if err := repo.RecordIdempotencyResult(ctx, "request-1", want); !errors.Is(err, ErrDuplicateIdempotencyKey) {
		t.Fatalf("duplicate RecordIdempotencyResult() error = %v, want ErrDuplicateIdempotencyKey", err)
	}
}

func TestSQLiteApprovalBindsSubmissionRevisionAndArtifactHash(t *testing.T) {
	db, repo := openSQLiteTestRepository(t)
	ctx := context.Background()
	if err := repo.CreateLabProject(ctx, testLabProject()); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSubmission(ctx, testSubmission(1, InReview, "reviewed-hash")); err != nil {
		t.Fatal(err)
	}
	approval := Approval{SubmissionID: "submission-1", Revision: 1, ArtifactSHA256: "reviewed-hash", ApprovedBy: "gio0z", ApprovedAt: time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC), IdempotencyKey: "approve-1"}
	if err := repo.RecordApproval(ctx, approval); err != nil {
		t.Fatalf("RecordApproval() error = %v", err)
	}
	var storedHash string
	if err := db.QueryRowContext(ctx, `SELECT artifact_sha256 FROM approvals WHERE submission_id = ? AND revision = ?`, approval.SubmissionID, approval.Revision).Scan(&storedHash); err != nil {
		t.Fatalf("read approval: %v", err)
	}
	if storedHash != "reviewed-hash" {
		t.Fatalf("stored approval hash = %q", storedHash)
	}
	approval.IdempotencyKey = "approve-2"
	approval.ArtifactSHA256 = "different-hash"
	if err := repo.RecordApproval(ctx, approval); !errors.Is(err, ErrConflict) {
		t.Fatalf("hash-mismatched RecordApproval() error = %v, want ErrConflict", err)
	}
}

func TestSQLiteAuditIsAppendOnlyAndOrdered(t *testing.T) {
	db, repo := openSQLiteTestRepository(t)
	ctx := context.Background()
	first := AuditEvent{Timestamp: time.Date(2026, 9, 3, 11, 0, 0, 0, time.UTC), RequestID: "request-1", Actor: Actor{Kind: ActorAgent, Identity: "agent:test"}, Profile: "default", Action: "create_lab_draft", ProjectID: "project-1", NewState: Draft, ArtifactSHA256: "hash-one", Result: "ok"}
	second := first
	second.Timestamp = first.Timestamp.Add(time.Second)
	second.RequestID, second.Action, second.NewState = "request-2", "request_review", InReview
	if err := repo.AppendAudit(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := repo.AppendAudit(ctx, second); err != nil {
		t.Fatal(err)
	}
	rows, err := db.QueryContext(ctx, `SELECT request_id FROM audit_events ORDER BY sequence`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if !reflect.DeepEqual(ids, []string{"request-1", "request-2"}) {
		t.Fatalf("audit order = %#v", ids)
	}
	if _, err := db.ExecContext(ctx, `UPDATE audit_events SET request_id = 'changed' WHERE request_id = 'request-1'`); err == nil {
		t.Fatal("audit update succeeded, want append-only rejection")
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM audit_events WHERE request_id = 'request-1'`); err == nil {
		t.Fatal("audit delete succeeded, want append-only rejection")
	}
}

func TestSQLiteWithTxRollsBackAllMutations(t *testing.T) {
	_, repo := openSQLiteTestRepository(t)
	ctx := context.Background()
	rollback := errors.New("rollback")
	err := repo.WithTx(ctx, func(txRepo Repository) error {
		if err := txRepo.CreateLabProject(ctx, testLabProject()); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("WithTx() error = %v, want rollback", err)
	}
	if _, err := repo.GetLabProject(ctx, "project-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("project survived rollback: %v", err)
	}
}

func boolPointer(value bool) *bool { return &value }

// TestSQLiteSourceURLsPersistAcrossProjectWrites covers the new reference
// metadata round trip and the "absent means unchanged" contract at the
// repository boundary.
func TestSQLiteSourceURLsPersistAcrossProjectWrites(t *testing.T) {
	_, repo := openSQLiteTestRepository(t)
	ctx := context.Background()

	project := testLabProject()
	project.SourceURLs = []string{"https://example.com/one", "https://example.com/two"}
	if err := repo.CreateLabProject(ctx, project); err != nil {
		t.Fatalf("CreateLabProject() error = %v", err)
	}

	stored, err := repo.GetLabProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetLabProject() error = %v", err)
	}
	if !reflect.DeepEqual(stored.SourceURLs, project.SourceURLs) {
		t.Fatalf("SourceURLs = %#v, want %#v", stored.SourceURLs, project.SourceURLs)
	}

	listed, err := repo.ListLabProjects(ctx, ProjectFilter{})
	if err != nil {
		t.Fatalf("ListLabProjects() error = %v", err)
	}
	if len(listed) != 1 || !reflect.DeepEqual(listed[0].SourceURLs, project.SourceURLs) {
		t.Fatalf("ListLabProjects() source urls = %#v, want %#v", listed, project.SourceURLs)
	}

	stored.SourceURLs = []string{"https://example.com/replaced"}
	if err := repo.UpdateLabProject(ctx, stored); err != nil {
		t.Fatalf("UpdateLabProject() error = %v", err)
	}
	updated, err := repo.GetLabProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetLabProject() after update error = %v", err)
	}
	if !reflect.DeepEqual(updated.SourceURLs, []string{"https://example.com/replaced"}) {
		t.Fatalf("SourceURLs after update = %#v", updated.SourceURLs)
	}
}

// TestOpenRegistryBackfillsSourceURLsOnLegacyRegistry guards the upgrade path:
// the base migration is CREATE TABLE IF NOT EXISTS, so a registry file written
// before source_urls existed keeps its old column set. Opening it must add the
// column rather than failing every subsequent project read.
func TestOpenRegistryBackfillsSourceURLsOnLegacyRegistry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")

	legacy, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open legacy registry: %v", err)
	}
	if _, err := legacy.Exec(`CREATE TABLE lab_projects (
		id TEXT PRIMARY KEY, slug TEXT NOT NULL UNIQUE, title TEXT NOT NULL,
		original_product TEXT NOT NULL, disclaimer TEXT NOT NULL,
		focus_json TEXT NOT NULL, platforms_json TEXT NOT NULL,
		status TEXT NOT NULL, featured INTEGER NOT NULL,
		created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	if _, err := legacy.Exec(`INSERT INTO lab_projects VALUES
		('project-legacy', 'legacy-slug', 'Legacy', 'Legacy Product', 'Disclaimer',
		 '["a"]', '["web"]', 'PUBLISHED', 1, '2026-09-01T00:00:00Z', '2026-09-01T00:00:00Z')`); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("close legacy registry: %v", err)
	}

	db, err := OpenRegistry(path)
	if err != nil {
		t.Fatalf("OpenRegistry() on legacy registry error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := NewSQLiteRepository(db)

	project, err := repo.GetLabProject(context.Background(), "project-legacy")
	if err != nil {
		t.Fatalf("GetLabProject() on migrated legacy registry error = %v", err)
	}
	if len(project.SourceURLs) != 0 {
		t.Fatalf("legacy project SourceURLs = %#v, want empty", project.SourceURLs)
	}
	if project.Slug != "legacy-slug" || !project.Featured {
		t.Fatalf("migration altered legacy row: %#v", project)
	}
}
