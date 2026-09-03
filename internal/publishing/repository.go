package publishing

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNotFound                = errors.New("publishing: not found")
	ErrConflict                = errors.New("publishing: conflict")
	ErrDuplicateIdempotencyKey = errors.New("publishing: duplicate idempotency key")
)

// ProjectFilter limits projects returned by ListLabProjects. Nil fields are not filtered.
type ProjectFilter struct {
	Status   string
	Featured *bool
}

// OperationResult is the durable response associated with an idempotency key.
type OperationResult struct {
	Code       string          `json:"code"`
	ResourceID string          `json:"resource_id,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
}

// AuditEvent is an immutable record of a publishing mutation.
type AuditEvent struct {
	Timestamp        time.Time       `json:"timestamp"`
	RequestID        string          `json:"request_id"`
	Actor            Actor           `json:"actor"`
	Profile          string          `json:"profile"`
	Action           string          `json:"action"`
	ProjectID        string          `json:"project_id,omitempty"`
	SubmissionID     string          `json:"submission_id,omitempty"`
	PreviousState    SubmissionState `json:"previous_state,omitempty"`
	NewState         SubmissionState `json:"new_state,omitempty"`
	ArtifactSHA256   string          `json:"artifact_sha256,omitempty"`
	Reason           string          `json:"reason,omitempty"`
	ApprovalIdentity string          `json:"approval_identity,omitempty"`
	DeploymentIDs    []string        `json:"deployment_ids,omitempty"`
	DestinationURLs  []string        `json:"destination_urls,omitempty"`
	Result           string          `json:"result"`
	Error            string          `json:"error,omitempty"`
}

// Repository persists publishing workflow state and supports atomic units of work.
type Repository interface {
	CreateLabProject(context.Context, LabProject) error
	GetLabProject(context.Context, string) (LabProject, error)
	ListLabProjects(context.Context, ProjectFilter) ([]LabProject, error)
	UpdateLabProject(context.Context, LabProject) error
	CreateSubmission(context.Context, Submission) error
	GetSubmission(context.Context, string) (Submission, error)
	UpdateSubmission(context.Context, Submission, int64) error
	RecordApproval(context.Context, Approval) error
	GetIdempotencyResult(context.Context, string) (*OperationResult, error)
	RecordIdempotencyResult(context.Context, string, OperationResult) error
	AppendAudit(context.Context, AuditEvent) error
	WithTx(context.Context, func(Repository) error) error
}
