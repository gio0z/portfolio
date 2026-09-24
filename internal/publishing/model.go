package publishing

import "time"

// ActorKind identifies the authority exercising a state transition.
type ActorKind string

const (
	ActorAgent  ActorKind = "AGENT"
	ActorOwner  ActorKind = "OWNER"
	ActorSystem ActorKind = "SYSTEM"
)

// Actor identifies an authenticated publishing actor.
type Actor struct {
	Kind     ActorKind `json:"kind"`
	Identity string    `json:"identity"`
}

// LabProject is the durable identity and public metadata anchor for a design-lab project.
type LabProject struct {
	ID              string `json:"id"`
	Slug            string `json:"slug"`
	Title           string `json:"title"`
	OriginalProduct string `json:"original_product"`
	Disclaimer      string `json:"disclaimer"`
	// SourceURLs are reference links to the follower-nominated source site.
	// They are stored as inert metadata: never fetched, never executed, and
	// only ever rendered as outbound links.
	SourceURLs []string  `json:"source_urls,omitempty"`
	Focus      []string  `json:"focus,omitempty"`
	Platforms  []string  `json:"platforms,omitempty"`
	Status     string    `json:"status"`
	Featured   bool      `json:"featured"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// PortfolioMetadata describes how an approved project appears publicly.
type PortfolioMetadata map[string]any

// ArtifactRef identifies immutable reviewed content.
type ArtifactRef struct {
	SHA256   string `json:"sha256"`
	MIMEType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

// Submission is one revision of a proposed Lab Project publication.
type Submission struct {
	ID                 string            `json:"id"`
	LabProjectID       string            `json:"lab_project_id"`
	Revision           int64             `json:"revision"`
	State              SubmissionState   `json:"state"`
	ArtifactSHA256     string            `json:"artifact_sha256"`
	PreviewURL         string            `json:"preview_url"`
	BuildResult        string            `json:"build_result"`
	TestResult         string            `json:"test_result"`
	SecurityScanResult string            `json:"security_scan_result"`
	PortfolioMetadata  PortfolioMetadata `json:"portfolio_metadata"`
	SubmittedBy        string            `json:"submitted_by"`
	SubmittedAt        time.Time         `json:"submitted_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// ReviewedArtifact returns the immutable artifact identity reviewed for this submission.
func (s Submission) ReviewedArtifact() ArtifactRef {
	return ArtifactRef{SHA256: s.ArtifactSHA256}
}

// Approval binds an owner decision to one immutable submission revision.
type Approval struct {
	SubmissionID   string    `json:"submission_id"`
	Revision       int64     `json:"revision"`
	ArtifactSHA256 string    `json:"artifact_sha256"`
	ApprovedBy     string    `json:"approved_by"`
	ApprovedAt     time.Time `json:"approved_at"`
	IdempotencyKey string    `json:"idempotency_key"`
}
