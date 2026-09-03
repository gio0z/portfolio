package adminapi

import (
	"time"

	"portfolio/internal/publishing"
)

// OverviewResponse contains summary statistics for the admin dashboard.
type OverviewResponse struct {
	PendingReviews    int            `json:"pending_reviews"`
	PublishedProjects int            `json:"published_projects"`
	BuildFailures     int            `json:"build_failures"`
	RecentAudits      []AuditSummary `json:"recent_audits"`
}

// ReviewQueueItem is an item in the admin review queue listing.
type ReviewQueueItem struct {
	ID                 string                   `json:"id"`
	ProjectID          string                   `json:"project_id"`
	ProjectSlug        string                   `json:"project_slug"`
	ProjectTitle       string                   `json:"project_title"`
	OriginalProduct    string                   `json:"original_product"`
	Revision           int64                    `json:"revision"`
	State              string                   `json:"state"`
	ArtifactSHA256     string                   `json:"artifact_sha256"`
	PreviewURL         string                   `json:"preview_url"`
	BuildResult        string                   `json:"build_result"`
	TestResult         string                   `json:"test_result"`
	SecurityScanResult string                   `json:"security_scan_result"`
	CanApprove         bool                     `json:"can_approve"`
	SubmittedBy        string                   `json:"submitted_by"`
	SubmittedAt        time.Time                `json:"submitted_at"`
	UpdatedAt          time.Time                `json:"updated_at"`
	PortfolioMetadata  publishing.PortfolioMetadata `json:"portfolio_metadata,omitempty"`
}

// ArtifactDetail represents reviewed artifact information.
type ArtifactDetail struct {
	SHA256 string `json:"sha256"`
}

// ReviewDetailResponse represents a detailed review item with project and submission info.
type ReviewDetailResponse struct {
	Submission publishing.Submission `json:"submission"`
	Project    publishing.LabProject `json:"project"`
	Artifact   ArtifactDetail        `json:"artifact"`
	CanApprove bool                  `json:"can_approve"`
}

// RequestChangesRequest is the request body for requesting changes.
type RequestChangesRequest struct {
	Reason string `json:"reason"`
}

// RejectRequest is the request body for rejecting a submission.
type RejectRequest struct {
	Reason string `json:"reason"`
}

// ApproveAndPublishRequest is the request body for approving and publishing.
type ApproveAndPublishRequest struct {
	ArtifactSHA256 string `json:"artifact_sha256"`
	IdempotencyKey string `json:"idempotency_key"`
}

// DecisionResponse is the response returned for review state transitions.
type DecisionResponse struct {
	Success    bool                  `json:"success"`
	Submission publishing.Submission `json:"submission"`
}

// PublicationResponse is the response returned on successful approve-and-publish.
type PublicationResponse struct {
	Success         bool      `json:"success"`
	SubmissionID    string    `json:"submission_id"`
	Revision        int64     `json:"revision"`
	ArtifactSHA256  string    `json:"artifact_sha256"`
	LabURL          string    `json:"lab_url"`
	PortfolioURL    string    `json:"portfolio_url"`
	DeploymentIDs   []string  `json:"deployment_ids,omitempty"`
	DestinationURLs []string  `json:"destination_urls,omitempty"`
	PublishedAt     time.Time `json:"published_at"`
}

// AuditSummary is a sanitized summary of an audit event.
type AuditSummary struct {
	Timestamp    time.Time `json:"timestamp"`
	RequestID    string    `json:"request_id"`
	ActorKind    string    `json:"actor_kind"`
	ActorID      string    `json:"actor_id"`
	Action       string    `json:"action"`
	ProjectID    string    `json:"project_id,omitempty"`
	SubmissionID string    `json:"submission_id,omitempty"`
	Result       string    `json:"result"`
}

// ErrorResponse is the standardized sanitized error response format.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}
