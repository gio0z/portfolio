package publishing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// MandatoryDisclaimer is the required disclaimer for all lab projects.
const MandatoryDisclaimer = "Independent redesign concept. Not affiliated with or endorsed by the original company."

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// CreateDraftInput describes the initial metadata and proposed design lab project.
type CreateDraftInput struct {
	Slug              string            `json:"slug"`
	Title             string            `json:"title"`
	OriginalProduct   string            `json:"original_product"`
	Disclaimer        string            `json:"disclaimer,omitempty"`
	Focus             []string          `json:"focus,omitempty"`
	Platforms         []string          `json:"platforms,omitempty"`
	Featured          bool              `json:"featured,omitempty"`
	ArtifactSHA256    string            `json:"artifact_sha256,omitempty"`
	PortfolioMetadata PortfolioMetadata `json:"portfolio_metadata,omitempty"`
	RequestID         string            `json:"request_id,omitempty"`
	Profile           string            `json:"profile,omitempty"`
}

// UpdateDraftInput contains modifications to an existing submission.
type UpdateDraftInput struct {
	SubmissionID      string            `json:"submission_id"`
	Title             string            `json:"title,omitempty"`
	ArtifactSHA256    string            `json:"artifact_sha256,omitempty"`
	PreviewURL        string            `json:"preview_url,omitempty"`
	PortfolioMetadata PortfolioMetadata `json:"portfolio_metadata,omitempty"`
	RequestID         string            `json:"request_id,omitempty"`
	Profile           string            `json:"profile,omitempty"`
}

// AttachAssetInput supplies an asset stream and metadata to store immutably.
type AttachAssetInput struct {
	Metadata  AssetMetadata `json:"metadata"`
	Content   io.Reader     `json:"-"`
	RequestID string        `json:"request_id,omitempty"`
	Profile   string        `json:"profile,omitempty"`
}

// PreviewResult records the sandboxed build, test, and scan evidence.
type PreviewResult struct {
	SubmissionID       string `json:"submission_id"`
	ArtifactSHA256     string `json:"artifact_sha256,omitempty"`
	PreviewURL         string `json:"preview_url"`
	BuildResult        string `json:"build_result"`
	TestResult         string `json:"test_result"`
	SecurityScanResult string `json:"security_scan_result"`
	DeploymentID       string `json:"deployment_id,omitempty"`
	RequestID          string `json:"request_id,omitempty"`
	Profile            string `json:"profile,omitempty"`
}

// ReviewDecision captures an owner review action with rationale.
type ReviewDecision struct {
	SubmissionID string `json:"submission_id"`
	Reason       string `json:"reason,omitempty"`
	RequestID    string `json:"request_id,omitempty"`
	Profile      string `json:"profile,omitempty"`
}

// ApproveInput provides parameters to approve and publish a reviewed submission.
type ApproveInput struct {
	SubmissionID   string `json:"submission_id"`
	ArtifactSHA256 string `json:"artifact_sha256"`
	IdempotencyKey string `json:"idempotency_key"`
	RequestID      string `json:"request_id,omitempty"`
	Profile        string `json:"profile,omitempty"`
}

// PublicationRequest describes the payload handed to an atomic publisher.
type PublicationRequest struct {
	Project    LabProject `json:"project"`
	Submission Submission `json:"submission"`
}

// PublicationOutput is the result of publishing to both Lab and portfolio destinations.
type PublicationOutput struct {
	LabURL          string   `json:"lab_url"`
	PortfolioURL    string   `json:"portfolio_url"`
	DeploymentIDs   []string `json:"deployment_ids,omitempty"`
	DestinationURLs []string `json:"destination_urls,omitempty"`
}

// PublicationResult represents a fully approved and published lab project revision.
type PublicationResult struct {
	SubmissionID    string    `json:"submission_id"`
	Revision        int64     `json:"revision"`
	ArtifactSHA256  string    `json:"artifact_sha256"`
	LabURL          string    `json:"lab_url"`
	PortfolioURL    string    `json:"portfolio_url"`
	DeploymentIDs   []string  `json:"deployment_ids,omitempty"`
	DestinationURLs []string  `json:"destination_urls,omitempty"`
	PublishedAt     time.Time `json:"published_at"`
}

// AtomicPublisher deploys an approved artifact to Lab and Portfolio destinations atomically.
type AtomicPublisher interface {
	Publish(context.Context, PublicationRequest) (PublicationOutput, error)
}

// PublishingService defines the domain orchestration for design lab publishing.
type PublishingService interface {
	CreateDraft(context.Context, Actor, CreateDraftInput) (LabProject, Submission, error)
	UpdateDraft(context.Context, Actor, UpdateDraftInput) (Submission, error)
	AttachAsset(context.Context, Actor, AttachAssetInput) (ArtifactRef, error)
	MarkPreviewReady(context.Context, Actor, PreviewResult) (Submission, error)
	RequestReview(context.Context, Actor, string) (Submission, error)
	RequestChanges(context.Context, Actor, ReviewDecision) (Submission, error)
	Reject(context.Context, Actor, ReviewDecision) (Submission, error)
	ApproveAndPublish(context.Context, Actor, ApproveInput) (PublicationResult, error)
	Archive(context.Context, Actor, string) error
}

type publishingService struct {
	repo      Repository
	artifacts ArtifactStore
	publisher AtomicPublisher
}

// NewPublishingService creates a PublishingService backed by repository, artifact store, and publisher.
func NewPublishingService(repo Repository, artifacts ArtifactStore, publisher AtomicPublisher) PublishingService {
	return &publishingService{
		repo:      repo,
		artifacts: artifacts,
		publisher: publisher,
	}
}

func (s *publishingService) resolveProfile(profile string) string {
	if profile != "" {
		return profile
	}
	return "default"
}

func (s *publishingService) resolveRequestID(reqID string) string {
	if reqID != "" {
		return reqID
	}
	var b [8]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("req-%x", b)
}

func generateRandomID(prefix string) string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b[:]))
}

func (s *publishingService) CreateDraft(ctx context.Context, actor Actor, input CreateDraftInput) (LabProject, Submission, error) {
	if actor.Kind != ActorAgent && actor.Kind != ActorOwner {
		return LabProject{}, Submission{}, NewForbiddenError("only agent or owner may create a draft")
	}
	if strings.TrimSpace(input.Slug) == "" {
		return LabProject{}, Submission{}, NewValidationError("slug is required", nil)
	}
	if !slugRegex.MatchString(input.Slug) {
		return LabProject{}, Submission{}, NewValidationError(fmt.Sprintf("invalid slug format %q: must match ^[a-z0-9]+(?:-[a-z0-9]+)*$", input.Slug), nil)
	}
	if strings.TrimSpace(input.Title) == "" {
		return LabProject{}, Submission{}, NewValidationError("title is required", nil)
	}
	if strings.TrimSpace(input.OriginalProduct) == "" {
		return LabProject{}, Submission{}, NewValidationError("original_product is required", nil)
	}

	disclaimer := input.Disclaimer
	if strings.TrimSpace(disclaimer) == "" {
		disclaimer = MandatoryDisclaimer
	} else if !strings.Contains(disclaimer, MandatoryDisclaimer) {
		disclaimer = strings.TrimSpace(disclaimer) + " " + MandatoryDisclaimer
	}

	now := time.Now().UTC()
	project := LabProject{
		ID:              generateRandomID("proj"),
		Slug:            strings.TrimSpace(input.Slug),
		Title:           strings.TrimSpace(input.Title),
		OriginalProduct: strings.TrimSpace(input.OriginalProduct),
		Disclaimer:      disclaimer,
		Focus:           input.Focus,
		Platforms:       input.Platforms,
		Status:          "draft",
		Featured:        input.Featured,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	submission := Submission{
		ID:                 generateRandomID("sub"),
		LabProjectID:       project.ID,
		Revision:           1,
		State:              Draft,
		ArtifactSHA256:     input.ArtifactSHA256,
		PortfolioMetadata:  input.PortfolioMetadata,
		SubmittedBy:        actor.Identity,
		SubmittedAt:        now,
		UpdatedAt:          now,
		BuildResult:        "",
		TestResult:         "",
		SecurityScanResult: "",
	}

	reqID := s.resolveRequestID(input.RequestID)
	profile := s.resolveProfile(input.Profile)

	err := s.repo.WithTx(ctx, func(txRepo Repository) error {
		if err := txRepo.CreateLabProject(ctx, project); err != nil {
			if errors.Is(err, ErrConflict) {
				return NewSlugConflictError(fmt.Sprintf("slug %q already exists", project.Slug), err)
			}
			return err
		}
		if err := txRepo.CreateSubmission(ctx, submission); err != nil {
			return err
		}
		return txRepo.AppendAudit(ctx, AuditEvent{
			Timestamp:      now,
			RequestID:      reqID,
			Actor:          actor,
			Profile:        profile,
			Action:         "create_draft",
			ProjectID:      project.ID,
			SubmissionID:   submission.ID,
			PreviousState:  "",
			NewState:       Draft,
			ArtifactSHA256: submission.ArtifactSHA256,
			Result:         "success",
		})
	})
	if err != nil {
		return LabProject{}, Submission{}, err
	}

	return project, submission, nil
}

func (s *publishingService) UpdateDraft(ctx context.Context, actor Actor, input UpdateDraftInput) (Submission, error) {
	if actor.Kind != ActorAgent && actor.Kind != ActorOwner {
		return Submission{}, NewForbiddenError("only agent or owner may update a draft")
	}
	if strings.TrimSpace(input.SubmissionID) == "" {
		return Submission{}, NewValidationError("submission_id is required", nil)
	}
	if input.Title != "" && strings.TrimSpace(input.Title) == "" {
		return Submission{}, NewValidationError("title cannot be empty", nil)
	}

	sub, err := s.repo.GetSubmission(ctx, input.SubmissionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Submission{}, NewNotFoundError("submission not found", err)
		}
		return Submission{}, err
	}

	if sub.State == Published || sub.State == Archived || sub.State == Rejected {
		return Submission{}, NewInvalidStateError(fmt.Sprintf("cannot update submission in %s state", sub.State), nil)
	}

	now := time.Now().UTC()
	var proj LabProject
	updateTitle := strings.TrimSpace(input.Title) != ""
	if updateTitle {
		p, err := s.repo.GetLabProject(ctx, sub.LabProjectID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return Submission{}, NewNotFoundError("project not found", err)
			}
			return Submission{}, err
		}
		proj = p
		proj.Title = strings.TrimSpace(input.Title)
		proj.UpdatedAt = now
	}

	newSub := sub
	newSub.Revision = sub.Revision + 1
	newSub.State = Draft
	newSub.UpdatedAt = now

	// Reset build/test/scan evidence on draft mutation; caller input cannot overwrite these
	newSub.BuildResult = ""
	newSub.TestResult = ""
	newSub.SecurityScanResult = ""

	if input.ArtifactSHA256 != "" {
		newSub.ArtifactSHA256 = input.ArtifactSHA256
	}
	if input.PreviewURL != "" {
		newSub.PreviewURL = input.PreviewURL
	}
	if input.PortfolioMetadata != nil {
		newSub.PortfolioMetadata = input.PortfolioMetadata
	}

	reqID := s.resolveRequestID(input.RequestID)
	profile := s.resolveProfile(input.Profile)

	err = s.repo.WithTx(ctx, func(txRepo Repository) error {
		if updateTitle {
			if err := txRepo.UpdateLabProject(ctx, proj); err != nil {
				return err
			}
		}
		if err := txRepo.UpdateSubmission(ctx, newSub, sub.Revision); err != nil {
			return err
		}
		return txRepo.AppendAudit(ctx, AuditEvent{
			Timestamp:      now,
			RequestID:      reqID,
			Actor:          actor,
			Profile:        profile,
			Action:         "update_draft",
			ProjectID:      sub.LabProjectID,
			SubmissionID:   sub.ID,
			PreviousState:  sub.State,
			NewState:       Draft,
			ArtifactSHA256: newSub.ArtifactSHA256,
			Result:         "success",
		})
	})
	if err != nil {
		return Submission{}, err
	}

	return newSub, nil
}

func (s *publishingService) AttachAsset(ctx context.Context, actor Actor, input AttachAssetInput) (ArtifactRef, error) {
	if actor.Kind != ActorAgent && actor.Kind != ActorOwner {
		return ArtifactRef{}, NewForbiddenError("only agent or owner may attach an asset")
	}
	if input.Content == nil {
		return ArtifactRef{}, NewValidationError("asset content reader is required", nil)
	}
	if strings.TrimSpace(input.Metadata.Name) == "" {
		return ArtifactRef{}, NewValidationError("asset name is required", nil)
	}

	ref, err := s.artifacts.Put(ctx, input.Content, input.Metadata)
	if err != nil {
		if errors.Is(err, ErrAssetTooLarge) || errors.Is(err, ErrAssetTypeNotAllowed) ||
			errors.Is(err, ErrMIMEMismatch) || errors.Is(err, ErrUnsafeArchive) ||
			errors.Is(err, ErrUnsafePath) || errors.Is(err, ErrArtifactCorrupt) {
			return ArtifactRef{}, NewValidationError(err.Error(), err)
		}
		return ArtifactRef{}, err
	}

	now := time.Now().UTC()
	reqID := s.resolveRequestID(input.RequestID)
	profile := s.resolveProfile(input.Profile)

	if err := s.repo.AppendAudit(ctx, AuditEvent{
		Timestamp:      now,
		RequestID:      reqID,
		Actor:          actor,
		Profile:        profile,
		Action:         "attach_asset",
		ArtifactSHA256: ref.SHA256,
		Result:         "success",
	}); err != nil {
		return ArtifactRef{}, err
	}

	return ref, nil
}

func (s *publishingService) MarkPreviewReady(ctx context.Context, actor Actor, result PreviewResult) (Submission, error) {
	if actor.Kind != ActorAgent && actor.Kind != ActorSystem && actor.Kind != ActorOwner {
		return Submission{}, NewForbiddenError("only agent, system, or owner may mark preview ready")
	}
	if strings.TrimSpace(result.SubmissionID) == "" {
		return Submission{}, NewValidationError("submission_id is required", nil)
	}

	sub, err := s.repo.GetSubmission(ctx, result.SubmissionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Submission{}, NewNotFoundError("submission not found", err)
		}
		return Submission{}, err
	}

	if sub.State == Published || sub.State == Archived || sub.State == Rejected {
		return Submission{}, NewInvalidStateError(fmt.Sprintf("cannot mark preview ready in %s state", sub.State), nil)
	}

	targetState := PreviewReady
	if result.BuildResult == "failed" {
		targetState = BuildFailed
	}

	if sub.State == Draft {
		if err := Transition(Draft, Building, ActorAgent); err != nil {
			return Submission{}, NewInvalidStateError(err.Error(), err)
		}
		if err := Transition(Building, targetState, ActorSystem); err != nil {
			return Submission{}, NewInvalidStateError(err.Error(), err)
		}
	} else if err := Transition(sub.State, targetState, actor.Kind); err != nil {
		return Submission{}, NewInvalidStateError(err.Error(), err)
	}

	now := time.Now().UTC()
	newSub := sub
	newSub.Revision = sub.Revision + 1
	newSub.State = targetState
	newSub.PreviewURL = result.PreviewURL
	newSub.BuildResult = result.BuildResult
	newSub.TestResult = result.TestResult
	newSub.SecurityScanResult = result.SecurityScanResult
	if result.ArtifactSHA256 != "" {
		newSub.ArtifactSHA256 = result.ArtifactSHA256
	}
	newSub.UpdatedAt = now

	reqID := s.resolveRequestID(result.RequestID)
	profile := s.resolveProfile(result.Profile)

	var deploymentIDs []string
	if result.DeploymentID != "" {
		deploymentIDs = []string{result.DeploymentID}
	}

	err = s.repo.WithTx(ctx, func(txRepo Repository) error {
		if err := txRepo.UpdateSubmission(ctx, newSub, sub.Revision); err != nil {
			return err
		}
		return txRepo.AppendAudit(ctx, AuditEvent{
			Timestamp:      now,
			RequestID:      reqID,
			Actor:          actor,
			Profile:        profile,
			Action:         "mark_preview_ready",
			ProjectID:      sub.LabProjectID,
			SubmissionID:   sub.ID,
			PreviousState:  sub.State,
			NewState:       targetState,
			ArtifactSHA256: newSub.ArtifactSHA256,
			DeploymentIDs:  deploymentIDs,
			Result:         "success",
		})
	})
	if err != nil {
		return Submission{}, err
	}

	return newSub, nil
}

func (s *publishingService) RequestReview(ctx context.Context, actor Actor, submissionID string) (Submission, error) {
	if actor.Kind != ActorAgent {
		return Submission{}, NewForbiddenError("only agent may request review")
	}
	if strings.TrimSpace(submissionID) == "" {
		return Submission{}, NewValidationError("submission_id is required", nil)
	}

	sub, err := s.repo.GetSubmission(ctx, submissionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Submission{}, NewNotFoundError("submission not found", err)
		}
		return Submission{}, err
	}

	if sub.State != PreviewReady {
		return Submission{}, NewInvalidStateError(fmt.Sprintf("submission must be in PREVIEW_READY to request review, got %s", sub.State), nil)
	}

	if sub.SecurityScanResult != "passed" {
		return Submission{}, NewScanFailedError(fmt.Sprintf("security scan result is %q", sub.SecurityScanResult))
	}
	if sub.BuildResult != "passed" {
		return Submission{}, NewValidationError(fmt.Sprintf("build result is %q", sub.BuildResult), nil)
	}
	if sub.TestResult != "passed" {
		return Submission{}, NewValidationError(fmt.Sprintf("test result is %q", sub.TestResult), nil)
	}

	if err := Transition(sub.State, InReview, actor.Kind); err != nil {
		return Submission{}, NewForbiddenError(err.Error())
	}

	now := time.Now().UTC()
	newSub := sub
	newSub.Revision = sub.Revision + 1
	newSub.State = InReview
	newSub.UpdatedAt = now

	reqID := s.resolveRequestID("")
	profile := s.resolveProfile("")

	err = s.repo.WithTx(ctx, func(txRepo Repository) error {
		if err := txRepo.UpdateSubmission(ctx, newSub, sub.Revision); err != nil {
			return err
		}
		return txRepo.AppendAudit(ctx, AuditEvent{
			Timestamp:      now,
			RequestID:      reqID,
			Actor:          actor,
			Profile:        profile,
			Action:         "request_review",
			ProjectID:      sub.LabProjectID,
			SubmissionID:   sub.ID,
			PreviousState:  sub.State,
			NewState:       InReview,
			ArtifactSHA256: newSub.ArtifactSHA256,
			Result:         "success",
		})
	})
	if err != nil {
		return Submission{}, err
	}

	return newSub, nil
}

func (s *publishingService) RequestChanges(ctx context.Context, actor Actor, decision ReviewDecision) (Submission, error) {
	if actor.Kind != ActorOwner {
		return Submission{}, NewForbiddenError("only owner may request changes")
	}
	if strings.TrimSpace(decision.SubmissionID) == "" {
		return Submission{}, NewValidationError("submission_id is required", nil)
	}

	sub, err := s.repo.GetSubmission(ctx, decision.SubmissionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Submission{}, NewNotFoundError("submission not found", err)
		}
		return Submission{}, err
	}

	if sub.State != InReview {
		return Submission{}, NewInvalidStateError(fmt.Sprintf("submission must be in IN_REVIEW to request changes, got %s", sub.State), nil)
	}

	if err := Transition(sub.State, ChangesRequested, actor.Kind); err != nil {
		return Submission{}, NewForbiddenError(err.Error())
	}

	now := time.Now().UTC()
	newSub := sub
	newSub.Revision = sub.Revision + 1
	newSub.State = ChangesRequested
	newSub.UpdatedAt = now

	// Store review reason in portfolio metadata for downstream tools
	meta := make(PortfolioMetadata)
	for k, v := range sub.PortfolioMetadata {
		meta[k] = v
	}
	reason := strings.TrimSpace(decision.Reason)
	if reason != "" {
		meta["review_reason"] = reason
	}
	newSub.PortfolioMetadata = meta

	reqID := s.resolveRequestID(decision.RequestID)
	profile := s.resolveProfile(decision.Profile)

	err = s.repo.WithTx(ctx, func(txRepo Repository) error {
		if err := txRepo.UpdateSubmission(ctx, newSub, sub.Revision); err != nil {
			return err
		}
		return txRepo.AppendAudit(ctx, AuditEvent{
			Timestamp:      now,
			RequestID:      reqID,
			Actor:          actor,
			Profile:        profile,
			Action:         "request_changes",
			ProjectID:      sub.LabProjectID,
			SubmissionID:   sub.ID,
			PreviousState:  sub.State,
			NewState:       ChangesRequested,
			ArtifactSHA256: newSub.ArtifactSHA256,
			Reason:         reason,
			Result:         "success",
		})
	})
	if err != nil {
		return Submission{}, err
	}

	return newSub, nil
}

func (s *publishingService) Reject(ctx context.Context, actor Actor, decision ReviewDecision) (Submission, error) {
	if actor.Kind != ActorOwner {
		return Submission{}, NewForbiddenError("only owner may reject a submission")
	}
	if strings.TrimSpace(decision.SubmissionID) == "" {
		return Submission{}, NewValidationError("submission_id is required", nil)
	}

	sub, err := s.repo.GetSubmission(ctx, decision.SubmissionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Submission{}, NewNotFoundError("submission not found", err)
		}
		return Submission{}, err
	}

	if sub.State != InReview {
		return Submission{}, NewInvalidStateError(fmt.Sprintf("submission must be in IN_REVIEW to reject, got %s", sub.State), nil)
	}

	if err := Transition(sub.State, Rejected, actor.Kind); err != nil {
		return Submission{}, NewForbiddenError(err.Error())
	}

	now := time.Now().UTC()
	newSub := sub
	newSub.Revision = sub.Revision + 1
	newSub.State = Rejected
	newSub.UpdatedAt = now

	// Store review reason in portfolio metadata for downstream tools
	meta := make(PortfolioMetadata)
	for k, v := range sub.PortfolioMetadata {
		meta[k] = v
	}
	reason := strings.TrimSpace(decision.Reason)
	if reason != "" {
		meta["review_reason"] = reason
	}
	newSub.PortfolioMetadata = meta

	reqID := s.resolveRequestID(decision.RequestID)
	profile := s.resolveProfile(decision.Profile)

	err = s.repo.WithTx(ctx, func(txRepo Repository) error {
		if err := txRepo.UpdateSubmission(ctx, newSub, sub.Revision); err != nil {
			return err
		}
		return txRepo.AppendAudit(ctx, AuditEvent{
			Timestamp:      now,
			RequestID:      reqID,
			Actor:          actor,
			Profile:        profile,
			Action:         "reject",
			ProjectID:      sub.LabProjectID,
			SubmissionID:   sub.ID,
			PreviousState:  sub.State,
			NewState:       Rejected,
			ArtifactSHA256: newSub.ArtifactSHA256,
			Reason:         reason,
			Result:         "success",
		})
	})
	if err != nil {
		return Submission{}, err
	}

	return newSub, nil
}

func (s *publishingService) ApproveAndPublish(ctx context.Context, actor Actor, input ApproveInput) (PublicationResult, error) {
	if actor.Kind != ActorOwner {
		return PublicationResult{}, NewForbiddenError("only owner may approve and publish")
	}
	if strings.TrimSpace(input.SubmissionID) == "" {
		return PublicationResult{}, NewValidationError("submission_id is required", nil)
	}
	if strings.TrimSpace(input.ArtifactSHA256) == "" {
		return PublicationResult{}, NewValidationError("artifact_sha256 is required", nil)
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		return PublicationResult{}, NewValidationError("idempotency_key is required", nil)
	}

	// Idempotency check: if already processed, return stored result
	opRes, err := s.repo.GetIdempotencyResult(ctx, input.IdempotencyKey)
	if err != nil {
		return PublicationResult{}, err
	}
	if opRes != nil {
		var cached PublicationResult
		if err := json.Unmarshal(opRes.Data, &cached); err != nil {
			return PublicationResult{}, fmt.Errorf("decode cached publication: %w", err)
		}
		return cached, nil
	}

	sub, err := s.repo.GetSubmission(ctx, input.SubmissionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return PublicationResult{}, NewNotFoundError("submission not found", err)
		}
		return PublicationResult{}, err
	}

	if sub.State != InReview {
		return PublicationResult{}, NewInvalidStateError(fmt.Sprintf("submission must be in IN_REVIEW to approve, got %s", sub.State), nil)
	}

	if sub.ArtifactSHA256 != input.ArtifactSHA256 {
		return PublicationResult{}, NewArtifactMismatchError(fmt.Sprintf("submitted artifact hash %q does not match reviewed artifact hash %q", input.ArtifactSHA256, sub.ArtifactSHA256))
	}

	if sub.SecurityScanResult != "passed" {
		return PublicationResult{}, NewScanFailedError(fmt.Sprintf("security scan result is %q", sub.SecurityScanResult))
	}
	if sub.BuildResult != "passed" || sub.TestResult != "passed" {
		return PublicationResult{}, NewValidationError("build and test results must be passed", nil)
	}

	proj, err := s.repo.GetLabProject(ctx, sub.LabProjectID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return PublicationResult{}, NewNotFoundError("project not found", err)
		}
		return PublicationResult{}, err
	}

	if !strings.Contains(proj.Disclaimer, MandatoryDisclaimer) {
		return PublicationResult{}, NewValidationError("mandatory disclaimer missing from project", nil)
	}

	// Promote artifact in immutable store
	if s.artifacts != nil {
		ref := ArtifactRef{SHA256: sub.ArtifactSHA256}
		if _, err := s.artifacts.Promote(ctx, ref, proj.Slug); err != nil {
			return PublicationResult{}, NewPublicationFailedError(fmt.Sprintf("artifact promotion failed: %v", err), err)
		}
	}

	// Publish via atomic publisher
	if s.publisher == nil {
		return PublicationResult{}, NewPublicationFailedError("atomic publisher is not configured", nil)
	}

	pubReq := PublicationRequest{
		Project:    proj,
		Submission: sub,
	}
	pubOut, err := s.publisher.Publish(ctx, pubReq)
	if err != nil {
		// Record audit event of failed publication
		now := time.Now().UTC()
		_ = s.repo.AppendAudit(ctx, AuditEvent{
			Timestamp:      now,
			RequestID:      s.resolveRequestID(input.RequestID),
			Actor:          actor,
			Profile:        s.resolveProfile(input.Profile),
			Action:         "approve_and_publish",
			ProjectID:      proj.ID,
			SubmissionID:   sub.ID,
			PreviousState:  InReview,
			NewState:       InReview,
			ArtifactSHA256: sub.ArtifactSHA256,
			Result:         "failed",
			Error:          err.Error(),
		})
		return PublicationResult{}, NewPublicationFailedError(fmt.Sprintf("publisher failed: %v", err), err)
	}

	now := time.Now().UTC()
	newSub := sub
	newSub.Revision = sub.Revision + 1
	newSub.State = Published
	newSub.UpdatedAt = now

	approval := Approval{
		SubmissionID:   sub.ID,
		Revision:       sub.Revision,
		ArtifactSHA256: sub.ArtifactSHA256,
		ApprovedBy:     actor.Identity,
		ApprovedAt:     now,
		IdempotencyKey: input.IdempotencyKey,
	}

	pubResult := PublicationResult{
		SubmissionID:    sub.ID,
		Revision:        newSub.Revision,
		ArtifactSHA256:  sub.ArtifactSHA256,
		LabURL:          pubOut.LabURL,
		PortfolioURL:    pubOut.PortfolioURL,
		DeploymentIDs:   pubOut.DeploymentIDs,
		DestinationURLs: pubOut.DestinationURLs,
		PublishedAt:     now,
	}

	reqID := s.resolveRequestID(input.RequestID)
	profile := s.resolveProfile(input.Profile)

	err = s.repo.WithTx(ctx, func(txRepo Repository) error {
		if err := txRepo.RecordApproval(ctx, approval); err != nil {
			return err
		}
		if err := txRepo.UpdateSubmission(ctx, newSub, sub.Revision); err != nil {
			return err
		}
		raw, err := json.Marshal(pubResult)
		if err != nil {
			return err
		}
		if err := txRepo.RecordIdempotencyResult(ctx, input.IdempotencyKey, OperationResult{
			Code:       "published",
			ResourceID: sub.ID,
			Data:       raw,
		}); err != nil {
			return err
		}
		return txRepo.AppendAudit(ctx, AuditEvent{
			Timestamp:        now,
			RequestID:        reqID,
			Actor:            actor,
			Profile:          profile,
			Action:           "approve_and_publish",
			ProjectID:        proj.ID,
			SubmissionID:     sub.ID,
			PreviousState:    InReview,
			NewState:         Published,
			ArtifactSHA256:   sub.ArtifactSHA256,
			ApprovalIdentity: actor.Identity,
			DeploymentIDs:    pubOut.DeploymentIDs,
			DestinationURLs:  pubOut.DestinationURLs,
			Result:           "success",
		})
	})
	if err != nil {
		return PublicationResult{}, NewPublicationFailedError(fmt.Sprintf("atomic database update failed: %v", err), err)
	}

	return pubResult, nil
}

func (s *publishingService) Archive(ctx context.Context, actor Actor, targetID string) error {
	if actor.Kind != ActorOwner {
		return NewForbiddenError("only owner may archive")
	}
	if strings.TrimSpace(targetID) == "" {
		return NewValidationError("target ID is required", nil)
	}

	sub, err := s.repo.GetSubmission(ctx, targetID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return NewNotFoundError("submission not found", err)
		}
		return err
	}

	if err := Transition(sub.State, Archived, actor.Kind); err != nil {
		return NewInvalidStateError(err.Error(), err)
	}

	now := time.Now().UTC()
	newSub := sub
	newSub.Revision = sub.Revision + 1
	newSub.State = Archived
	newSub.UpdatedAt = now

	reqID := s.resolveRequestID("")
	profile := s.resolveProfile("")

	return s.repo.WithTx(ctx, func(txRepo Repository) error {
		if err := txRepo.UpdateSubmission(ctx, newSub, sub.Revision); err != nil {
			return err
		}
		return txRepo.AppendAudit(ctx, AuditEvent{
			Timestamp:      now,
			RequestID:      reqID,
			Actor:          actor,
			Profile:        profile,
			Action:         "archive",
			ProjectID:      sub.LabProjectID,
			SubmissionID:   sub.ID,
			PreviousState:  sub.State,
			NewState:       Archived,
			ArtifactSHA256: newSub.ArtifactSHA256,
			Result:         "success",
		})
	})
}
