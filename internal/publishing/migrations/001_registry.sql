CREATE TABLE IF NOT EXISTS lab_projects (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    original_product TEXT NOT NULL,
    disclaimer TEXT NOT NULL,
    focus_json TEXT NOT NULL,
    platforms_json TEXT NOT NULL,
    status TEXT NOT NULL,
    featured INTEGER NOT NULL CHECK (featured IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS submissions (
    id TEXT NOT NULL,
    lab_project_id TEXT NOT NULL REFERENCES lab_projects(id),
    revision INTEGER NOT NULL CHECK (revision > 0),
    state TEXT NOT NULL,
    artifact_sha256 TEXT NOT NULL,
    preview_url TEXT NOT NULL,
    build_result TEXT NOT NULL,
    test_result TEXT NOT NULL,
    security_scan_result TEXT NOT NULL,
    portfolio_metadata_json TEXT NOT NULL,
    submitted_by TEXT NOT NULL,
    submitted_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (id, revision)
);
CREATE UNIQUE INDEX IF NOT EXISTS submissions_id_artifact_revision
    ON submissions(id, revision, artifact_sha256);

CREATE TABLE IF NOT EXISTS approvals (
    submission_id TEXT NOT NULL,
    revision INTEGER NOT NULL,
    artifact_sha256 TEXT NOT NULL,
    approved_by TEXT NOT NULL,
    approved_at TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    PRIMARY KEY (submission_id, revision),
    FOREIGN KEY (submission_id, revision, artifact_sha256)
        REFERENCES submissions(id, revision, artifact_sha256)
);

CREATE TABLE IF NOT EXISTS idempotency_results (
    idempotency_key TEXT PRIMARY KEY,
    result_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_events (
    sequence INTEGER PRIMARY KEY AUTOINCREMENT,
    event_json TEXT NOT NULL,
    request_id TEXT NOT NULL,
    timestamp TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS audit_events_no_update
BEFORE UPDATE ON audit_events BEGIN
    SELECT RAISE(ABORT, 'audit events are append-only');
END;
CREATE TRIGGER IF NOT EXISTS audit_events_no_delete
BEFORE DELETE ON audit_events BEGIN
    SELECT RAISE(ABORT, 'audit events are append-only');
END;

CREATE TABLE IF NOT EXISTS publications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    submission_id TEXT NOT NULL,
    revision INTEGER NOT NULL,
    artifact_sha256 TEXT NOT NULL,
    lab_url TEXT NOT NULL,
    portfolio_url TEXT NOT NULL,
    deployment_metadata_json TEXT NOT NULL,
    published_at TEXT NOT NULL,
    UNIQUE (submission_id, revision),
    FOREIGN KEY (submission_id, revision, artifact_sha256)
        REFERENCES submissions(id, revision, artifact_sha256)
);
