package publishing

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/001_registry.sql
var registryMigration string

type queryExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// SQLiteRepository stores publishing state in SQLite.
type SQLiteRepository struct {
	db queryExecutor
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

// OpenRegistry opens and migrates a SQLite registry at path.
func OpenRegistry(path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("publishing: registry path is empty")
	}
	if path != ":memory:" {
		path = filepath.Clean(path)
	}
	dsn := path
	if path != ":memory:" {
		dsn = "file:" + path
	}
	dsn += "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("publishing: open registry: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(registryMigration); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("publishing: migrate registry: %w", err)
	}
	return db, nil
}

func (r *SQLiteRepository) CreateLabProject(ctx context.Context, project LabProject) error {
	focus, err := json.Marshal(project.Focus)
	if err != nil {
		return fmt.Errorf("publishing: encode focus: %w", err)
	}
	platforms, err := json.Marshal(project.Platforms)
	if err != nil {
		return fmt.Errorf("publishing: encode platforms: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO lab_projects
		(id, slug, title, original_product, disclaimer, focus_json, platforms_json, status, featured, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, project.ID, project.Slug, project.Title,
		project.OriginalProduct, project.Disclaimer, string(focus), string(platforms), project.Status,
		project.Featured, encodeTime(project.CreatedAt), encodeTime(project.UpdatedAt))
	return classifyConflict(err)
}

func (r *SQLiteRepository) GetLabProject(ctx context.Context, id string) (LabProject, error) {
	return scanProject(r.db.QueryRowContext(ctx, `SELECT id, slug, title, original_product, disclaimer,
		focus_json, platforms_json, status, featured, created_at, updated_at FROM lab_projects WHERE id = ?`, id))
}

func (r *SQLiteRepository) ListLabProjects(ctx context.Context, filter ProjectFilter) ([]LabProject, error) {
	query := `SELECT id, slug, title, original_product, disclaimer, focus_json, platforms_json,
		status, featured, created_at, updated_at FROM lab_projects WHERE 1=1`
	var args []any
	if filter.Status != "" {
		query += ` AND status = ?`
		args = append(args, filter.Status)
	}
	if filter.Featured != nil {
		query += ` AND featured = ?`
		args = append(args, *filter.Featured)
	}
	query += ` ORDER BY created_at, id`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("publishing: list projects: %w", err)
	}
	defer rows.Close()
	var projects []LabProject
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("publishing: list projects: %w", err)
	}
	return projects, nil
}

type rowScanner interface{ Scan(...any) error }

func scanProject(row rowScanner) (LabProject, error) {
	var project LabProject
	var focus, platforms, created, updated string
	err := row.Scan(&project.ID, &project.Slug, &project.Title, &project.OriginalProduct, &project.Disclaimer,
		&focus, &platforms, &project.Status, &project.Featured, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return LabProject{}, ErrNotFound
	}
	if err != nil {
		return LabProject{}, fmt.Errorf("publishing: scan project: %w", err)
	}
	if err := json.Unmarshal([]byte(focus), &project.Focus); err != nil {
		return LabProject{}, fmt.Errorf("publishing: decode focus: %w", err)
	}
	if err := json.Unmarshal([]byte(platforms), &project.Platforms); err != nil {
		return LabProject{}, fmt.Errorf("publishing: decode platforms: %w", err)
	}
	project.CreatedAt, err = decodeTime(created)
	if err != nil {
		return LabProject{}, err
	}
	project.UpdatedAt, err = decodeTime(updated)
	if err != nil {
		return LabProject{}, err
	}
	return project, nil
}

func (r *SQLiteRepository) CreateSubmission(ctx context.Context, submission Submission) error {
	metadata, err := json.Marshal(submission.PortfolioMetadata)
	if err != nil {
		return fmt.Errorf("publishing: encode portfolio metadata: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO submissions
		(id, lab_project_id, revision, state, artifact_sha256, preview_url, build_result, test_result,
		 security_scan_result, portfolio_metadata_json, submitted_by, submitted_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, submission.ID, submission.LabProjectID,
		submission.Revision, submission.State, submission.ArtifactSHA256, submission.PreviewURL,
		submission.BuildResult, submission.TestResult, submission.SecurityScanResult, string(metadata),
		submission.SubmittedBy, encodeTime(submission.SubmittedAt), encodeTime(submission.UpdatedAt))
	return classifyConflict(err)
}

func (r *SQLiteRepository) GetSubmission(ctx context.Context, id string) (Submission, error) {
	return scanSubmission(r.db.QueryRowContext(ctx, `SELECT id, lab_project_id, revision, state,
		artifact_sha256, preview_url, build_result, test_result, security_scan_result,
		portfolio_metadata_json, submitted_by, submitted_at, updated_at FROM submissions
		WHERE id = ? ORDER BY revision DESC LIMIT 1`, id))
}

func (r *SQLiteRepository) UpdateSubmission(ctx context.Context, submission Submission, expectedRevision int64) error {
	if submission.Revision != expectedRevision+1 {
		return ErrConflict
	}
	metadata, err := json.Marshal(submission.PortfolioMetadata)
	if err != nil {
		return fmt.Errorf("publishing: encode portfolio metadata: %w", err)
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO submissions
		(id, lab_project_id, revision, state, artifact_sha256, preview_url, build_result, test_result,
		 security_scan_result, portfolio_metadata_json, submitted_by, submitted_at, updated_at)
		SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		WHERE (SELECT MAX(revision) FROM submissions WHERE id = ?) = ?`, submission.ID, submission.LabProjectID,
		submission.Revision, submission.State, submission.ArtifactSHA256, submission.PreviewURL,
		submission.BuildResult, submission.TestResult, submission.SecurityScanResult, string(metadata),
		submission.SubmittedBy, encodeTime(submission.SubmittedAt), encodeTime(submission.UpdatedAt),
		submission.ID, expectedRevision)
	if err != nil {
		return classifyConflict(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("publishing: update submission: %w", err)
	}
	if affected != 1 {
		return ErrConflict
	}
	return nil
}

func scanSubmission(row rowScanner) (Submission, error) {
	var submission Submission
	var metadata, submitted, updated string
	err := row.Scan(&submission.ID, &submission.LabProjectID, &submission.Revision, &submission.State,
		&submission.ArtifactSHA256, &submission.PreviewURL, &submission.BuildResult, &submission.TestResult,
		&submission.SecurityScanResult, &metadata, &submission.SubmittedBy, &submitted, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Submission{}, ErrNotFound
	}
	if err != nil {
		return Submission{}, fmt.Errorf("publishing: scan submission: %w", err)
	}
	if err := json.Unmarshal([]byte(metadata), &submission.PortfolioMetadata); err != nil {
		return Submission{}, fmt.Errorf("publishing: decode portfolio metadata: %w", err)
	}
	var errTime error
	submission.SubmittedAt, errTime = decodeTime(submitted)
	if errTime != nil {
		return Submission{}, errTime
	}
	submission.UpdatedAt, errTime = decodeTime(updated)
	if errTime != nil {
		return Submission{}, errTime
	}
	return submission, nil
}

func (r *SQLiteRepository) RecordApproval(ctx context.Context, approval Approval) error {
	result, err := r.db.ExecContext(ctx, `INSERT INTO approvals
		(submission_id, revision, artifact_sha256, approved_by, approved_at, idempotency_key)
		SELECT id, revision, artifact_sha256, ?, ?, ? FROM submissions
		WHERE id = ? AND revision = ? AND artifact_sha256 = ?`, approval.ApprovedBy, encodeTime(approval.ApprovedAt),
		approval.IdempotencyKey, approval.SubmissionID, approval.Revision, approval.ArtifactSHA256)
	if err != nil {
		return classifyConflict(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("publishing: record approval: %w", err)
	}
	if affected != 1 {
		return ErrConflict
	}
	return nil
}

func (r *SQLiteRepository) GetIdempotencyResult(ctx context.Context, key string) (*OperationResult, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT result_json FROM idempotency_results WHERE idempotency_key = ?`, key).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("publishing: get idempotency result: %w", err)
	}
	var result OperationResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("publishing: decode operation result: %w", err)
	}
	return &result, nil
}

func (r *SQLiteRepository) RecordIdempotencyResult(ctx context.Context, key string, result OperationResult) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("publishing: encode operation result: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO idempotency_results (idempotency_key, result_json) VALUES (?, ?)`, key, string(raw))
	if isConstraintError(err) {
		return fmt.Errorf("%w: %s", ErrDuplicateIdempotencyKey, key)
	}
	if err != nil {
		return fmt.Errorf("publishing: record idempotency result: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) AppendAudit(ctx context.Context, event AuditEvent) error {
	raw, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("publishing: encode audit event: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO audit_events (event_json, request_id, timestamp) VALUES (?, ?, ?)`,
		string(raw), event.RequestID, encodeTime(event.Timestamp))
	if err != nil {
		return fmt.Errorf("publishing: append audit: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) WithTx(ctx context.Context, fn func(Repository) error) (err error) {
	db, ok := r.db.(*sql.DB)
	if !ok {
		return fmt.Errorf("publishing: nested transactions are not supported")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("publishing: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txRepo := &SQLiteRepository{db: tx}
	if err = fn(txRepo); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("publishing: commit transaction: %w", err)
	}
	return nil
}

func encodeTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
func decodeTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("publishing: decode time: %w", err)
	}
	return parsed, nil
}
func isConstraintError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "constraint")
}
func classifyConflict(err error) error {
	if err == nil {
		return nil
	}
	if isConstraintError(err) {
		return fmt.Errorf("%w: %v", ErrConflict, err)
	}
	return err
}
