package publishing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type fakeAtomicPublisher struct {
	publishFunc func(ctx context.Context, req PublicationRequest) (PublicationOutput, error)
	calls       int
}

func (f *fakeAtomicPublisher) Publish(ctx context.Context, req PublicationRequest) (PublicationOutput, error) {
	f.calls++
	if f.publishFunc != nil {
		return f.publishFunc(ctx, req)
	}
	return PublicationOutput{
		LabURL:          "https://lab.example.com/" + req.Project.Slug,
		PortfolioURL:    "https://portfolio.example.com/lab/" + req.Project.Slug,
		DeploymentIDs:   []string{"deploy-lab-1", "deploy-port-1"},
		DestinationURLs: []string{"https://lab.example.com/" + req.Project.Slug, "https://portfolio.example.com/lab/" + req.Project.Slug},
	}, nil
}

func setupServiceTest(t *testing.T) (*SQLiteRepository, *LocalArtifactStore, *fakeAtomicPublisher, PublishingService) {
	t.Helper()
	_, repo := openSQLiteTestRepository(t)
	root := t.TempDir()
	store := NewLocalArtifactStore(root, AssetPolicy{MaxBytes: 10 * 1024 * 1024})
	publisher := &fakeAtomicPublisher{}
	svc := NewPublishingService(repo, store, publisher)
	return repo, store, publisher, svc
}

func createTestArtifact(t *testing.T, store *LocalArtifactStore, content string) ArtifactRef {
	t.Helper()
	ref, err := store.Put(context.Background(), strings.NewReader(content), AssetMetadata{
		Name:     "index.html",
		MIMEType: "text/html",
	})
	if err != nil {
		t.Fatalf("createTestArtifact store.Put() error = %v", err)
	}
	return ref
}

func TestServiceAgentDraftCreation(t *testing.T) {
	ctx := context.Background()
	repo, _, _, svc := setupServiceTest(t)

	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}
	input := CreateDraftInput{
		Slug:            "mail-redesign",
		Title:           "Mail Redesign",
		OriginalProduct: "FastMail",
		Focus:           []string{"layout", "typography"},
		Platforms:       []string{"web"},
		Featured:        true,
		RequestID:       "req-create-1",
		Profile:         "hermes-agent",
	}

	project, submission, err := svc.CreateDraft(ctx, agent, input)
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	if project.ID == "" || project.Slug != "mail-redesign" {
		t.Fatalf("unexpected project: %+v", project)
	}
	if project.Disclaimer != MandatoryDisclaimer {
		t.Fatalf("project disclaimer = %q, want %q", project.Disclaimer, MandatoryDisclaimer)
	}
	if submission.ID == "" || submission.LabProjectID != project.ID {
		t.Fatalf("unexpected submission: %+v", submission)
	}
	if submission.Revision != 1 {
		t.Fatalf("submission revision = %d, want 1", submission.Revision)
	}
	if submission.State != Draft {
		t.Fatalf("submission state = %s, want %s", submission.State, Draft)
	}
	if submission.SubmittedBy != agent.Identity {
		t.Fatalf("submitted by = %q, want %q", submission.SubmittedBy, agent.Identity)
	}

	// Verify persistence
	gotProj, err := repo.GetLabProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetLabProject() error = %v", err)
	}
	if gotProj.Slug != project.Slug {
		t.Fatalf("persisted project slug = %q, want %q", gotProj.Slug, project.Slug)
	}

	gotSub, err := repo.GetSubmission(ctx, submission.ID)
	if err != nil {
		t.Fatalf("GetSubmission() error = %v", err)
	}
	if gotSub.Revision != 1 || gotSub.State != Draft {
		t.Fatalf("persisted submission = %+v", gotSub)
	}
}

func TestServiceRequiredDisclaimerInjection(t *testing.T) {
	ctx := context.Background()
	_, _, _, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}

	// Case 1: Empty disclaimer -> injected
	proj1, _, err := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug:            "empty-disclaimer",
		Title:           "Test 1",
		OriginalProduct: "Product 1",
		Disclaimer:      "",
	})
	if err != nil {
		t.Fatalf("CreateDraft(empty) error = %v", err)
	}
	if proj1.Disclaimer != MandatoryDisclaimer {
		t.Fatalf("disclaimer = %q, want %q", proj1.Disclaimer, MandatoryDisclaimer)
	}

	// Case 2: Custom disclaimer missing mandatory phrase -> injected
	proj2, _, err := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug:            "custom-disclaimer",
		Title:           "Test 2",
		OriginalProduct: "Product 2",
		Disclaimer:      "Custom experimental prototype.",
	})
	if err != nil {
		t.Fatalf("CreateDraft(custom) error = %v", err)
	}
	if !strings.Contains(proj2.Disclaimer, MandatoryDisclaimer) {
		t.Fatalf("disclaimer = %q, must contain %q", proj2.Disclaimer, MandatoryDisclaimer)
	}

	// Case 3: Disclaimer already contains mandatory disclaimer -> preserved
	existing := "Some note. " + MandatoryDisclaimer
	proj3, _, err := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug:            "full-disclaimer",
		Title:           "Test 3",
		OriginalProduct: "Product 3",
		Disclaimer:      existing,
	})
	if err != nil {
		t.Fatalf("CreateDraft(existing) error = %v", err)
	}
	if proj3.Disclaimer != existing {
		t.Fatalf("disclaimer = %q, want %q", proj3.Disclaimer, existing)
	}
}

func TestServiceAgentRejectionFromApproval(t *testing.T) {
	ctx := context.Background()
	_, store, _, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}
	art := createTestArtifact(t, store, "<html><body>Valid</body></html>")

	_, sub, err := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug:            "agent-approval-test",
		Title:           "Approve Test",
		OriginalProduct: "Product",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	sub, err = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example.com/test",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
	})
	if err != nil {
		t.Fatalf("MarkPreviewReady: %v", err)
	}

	sub, err = svc.RequestReview(ctx, agent, sub.ID)
	if err != nil {
		t.Fatalf("RequestReview: %v", err)
	}

	// Agent attempts approval -> MUST fail with forbidden
	_, err = svc.ApproveAndPublish(ctx, agent, ApproveInput{
		SubmissionID:   sub.ID,
		ArtifactSHA256: art.SHA256,
		IdempotencyKey: "agent-attempt-1",
	})
	if err == nil {
		t.Fatal("ApproveAndPublish by agent succeeded, want forbidden error")
	}
	if code := ErrorCode(err); code != ErrCodeForbidden {
		t.Fatalf("ErrorCode = %q, want %q", code, ErrCodeForbidden)
	}
}

func TestServiceReviewRequestOnlyAfterBuildTestScanPass(t *testing.T) {
	ctx := context.Background()
	_, store, _, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}
	art := createTestArtifact(t, store, "<html><body>Test</body></html>")

	// Case 1: Build failed
	_, sub1, _ := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug: "fail-build", Title: "Fail Build", OriginalProduct: "Prod",
	})
	sub1, err := svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub1.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example/1",
		BuildResult:        "failed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
	})
	if err != nil {
		t.Fatalf("MarkPreviewReady: %v", err)
	}
	if _, err := svc.RequestReview(ctx, agent, sub1.ID); err == nil {
		t.Fatal("RequestReview succeeded with failed build, want error")
	}

	// Case 2: Scan failed
	_, sub2, _ := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug: "fail-scan", Title: "Fail Scan", OriginalProduct: "Prod",
	})
	sub2, err = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub2.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example/2",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "failed",
	})
	if err != nil {
		t.Fatalf("MarkPreviewReady: %v", err)
	}
	if _, err := svc.RequestReview(ctx, agent, sub2.ID); err == nil {
		t.Fatal("RequestReview succeeded with failed scan, want error")
	} else if code := ErrorCode(err); code != ErrCodeScanFailed && code != ErrCodeValidationFailed {
		t.Fatalf("ErrorCode = %q, want %q or %q", code, ErrCodeScanFailed, ErrCodeValidationFailed)
	}

	// Case 3: All passed
	_, sub3, _ := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug: "pass-all", Title: "Pass All", OriginalProduct: "Prod",
	})
	sub3, err = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub3.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example/3",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
	})
	if err != nil {
		t.Fatalf("MarkPreviewReady: %v", err)
	}
	sub3, err = svc.RequestReview(ctx, agent, sub3.ID)
	if err != nil {
		t.Fatalf("RequestReview() unexpected error = %v", err)
	}
	if sub3.State != InReview {
		t.Fatalf("sub3 state = %s, want %s", sub3.State, InReview)
	}
}

func TestServiceMutationAfterReviewCreatesNewRevisionAndInvalidatesApproval(t *testing.T) {
	ctx := context.Background()
	repo, store, _, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}
	owner := Actor{Kind: ActorOwner, Identity: "owner:gio0z"}
	art1 := createTestArtifact(t, store, "<html><body>Rev1</body></html>")

	_, sub, err := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug: "mutate-review", Title: "Mutate Review", OriginalProduct: "Prod",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	sub, err = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub.ID,
		ArtifactSHA256:     art1.SHA256,
		PreviewURL:         "https://preview.example/1",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
	})
	if err != nil {
		t.Fatalf("MarkPreviewReady: %v", err)
	}

	sub, err = svc.RequestReview(ctx, agent, sub.ID)
	if err != nil {
		t.Fatalf("RequestReview: %v", err)
	}
	if sub.Revision != 3 || sub.State != InReview {
		t.Fatalf("submission revision = %d (want 3), state = %s (want IN_REVIEW)", sub.Revision, sub.State)
	}

	// Mutation during IN_REVIEW: agent updates draft
	art2 := createTestArtifact(t, store, "<html><body>Rev2 Changed</body></html>")
	mutatedSub, err := svc.UpdateDraft(ctx, agent, UpdateDraftInput{
		SubmissionID:   sub.ID,
		Title:          "Mutate Review Updated",
		ArtifactSHA256: art2.SHA256,
		RequestID:      "req-update-1",
	})
	if err != nil {
		t.Fatalf("UpdateDraft() error = %v", err)
	}

	if mutatedSub.Revision != sub.Revision+1 {
		t.Fatalf("mutated revision = %d, want %d", mutatedSub.Revision, sub.Revision+1)
	}
	if mutatedSub.State != Draft {
		t.Fatalf("mutated state = %s, want %s", mutatedSub.State, Draft)
	}
	if mutatedSub.BuildResult != "" || mutatedSub.TestResult != "" || mutatedSub.SecurityScanResult != "" {
		t.Fatalf("expected build/test/scan to reset, got build=%q test=%q scan=%q",
			mutatedSub.BuildResult, mutatedSub.TestResult, mutatedSub.SecurityScanResult)
	}

	// Verify repo returns the new revision in DRAFT
	latest, err := repo.GetSubmission(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubmission: %v", err)
	}
	if latest.Revision != mutatedSub.Revision || latest.State != Draft {
		t.Fatalf("latest submission in repo = %+v", latest)
	}

	// Verify project title was updated in repo
	proj, err := repo.GetLabProject(ctx, sub.LabProjectID)
	if err != nil {
		t.Fatalf("GetLabProject: %v", err)
	}
	if proj.Title != "Mutate Review Updated" {
		t.Fatalf("project title in repo = %q, want %q", proj.Title, "Mutate Review Updated")
	}

	// Attempting to approve now must fail because current state is DRAFT (approval invalidated)
	_, err = svc.ApproveAndPublish(ctx, owner, ApproveInput{
		SubmissionID:   sub.ID,
		ArtifactSHA256: art1.SHA256,
		IdempotencyKey: "approve-invalidated",
	})
	if err == nil {
		t.Fatal("ApproveAndPublish on mutated draft succeeded, want invalid_state error")
	}
	if code := ErrorCode(err); code != ErrCodeInvalidState {
		t.Fatalf("ErrorCode = %q, want %q", code, ErrCodeInvalidState)
	}
}

func TestServiceOwnerApprovalBoundToReviewedHash(t *testing.T) {
	ctx := context.Background()
	_, store, _, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}
	owner := Actor{Kind: ActorOwner, Identity: "owner:gio0z"}
	art := createTestArtifact(t, store, "<html><body>Reviewed Hash</body></html>")

	_, sub, _ := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug: "hash-bind-test", Title: "Hash Bind", OriginalProduct: "Prod",
	})
	sub, _ = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example/hash",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
	})
	sub, _ = svc.RequestReview(ctx, agent, sub.ID)

	// Owner passes mismatched artifact hash
	mismatchedHash := strings.Repeat("f", 64)
	_, err := svc.ApproveAndPublish(ctx, owner, ApproveInput{
		SubmissionID:   sub.ID,
		ArtifactSHA256: mismatchedHash,
		IdempotencyKey: "approve-mismatch",
	})
	if err == nil {
		t.Fatal("ApproveAndPublish with mismatched hash succeeded, want error")
	}
	if code := ErrorCode(err); code != ErrCodeArtifactMismatch {
		t.Fatalf("ErrorCode = %q, want %q", code, ErrCodeArtifactMismatch)
	}
}

func TestServiceRepeatedIdempotencyKeyReturnsIdenticalResult(t *testing.T) {
	ctx := context.Background()
	_, store, publisher, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}
	owner := Actor{Kind: ActorOwner, Identity: "owner:gio0z"}
	art := createTestArtifact(t, store, "<html><body>Idempotent</body></html>")

	_, sub, _ := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug: "idempotent-test", Title: "Idempotent", OriginalProduct: "Prod",
	})
	sub, _ = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example/idem",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
	})
	sub, _ = svc.RequestReview(ctx, agent, sub.ID)

	approveInput := ApproveInput{
		SubmissionID:   sub.ID,
		ArtifactSHA256: art.SHA256,
		IdempotencyKey: "unique-key-12345",
	}

	firstResult, err := svc.ApproveAndPublish(ctx, owner, approveInput)
	if err != nil {
		t.Fatalf("first ApproveAndPublish: %v", err)
	}
	if publisher.calls != 1 {
		t.Fatalf("publisher calls = %d, want 1", publisher.calls)
	}

	// Second call with same idempotency key
	secondResult, err := svc.ApproveAndPublish(ctx, owner, approveInput)
	if err != nil {
		t.Fatalf("second ApproveAndPublish: %v", err)
	}
	if publisher.calls != 1 {
		t.Fatalf("publisher calls after replay = %d, want 1 (no duplicate publish)", publisher.calls)
	}

	if firstResult.LabURL != secondResult.LabURL || firstResult.PortfolioURL != secondResult.PortfolioURL {
		t.Fatalf("results differ: first=%+v, second=%+v", firstResult, secondResult)
	}
	if firstResult.ArtifactSHA256 != secondResult.ArtifactSHA256 {
		t.Fatalf("artifact hashes differ: %q vs %q", firstResult.ArtifactSHA256, secondResult.ArtifactSHA256)
	}
}

func TestServiceAtomicPublishRollbackWhenDestinationFails(t *testing.T) {
	ctx := context.Background()
	repo, store, publisher, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}
	owner := Actor{Kind: ActorOwner, Identity: "owner:gio0z"}
	art := createTestArtifact(t, store, "<html><body>Rollback Test</body></html>")

	_, sub, _ := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug: "rollback-test", Title: "Rollback", OriginalProduct: "Prod",
	})
	sub, _ = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example/rb",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
	})
	sub, _ = svc.RequestReview(ctx, agent, sub.ID)

	// Simulate publication failure
	publisher.publishFunc = func(ctx context.Context, req PublicationRequest) (PublicationOutput, error) {
		return PublicationOutput{}, errors.New("lab endpoint 500 internal server error")
	}

	_, err := svc.ApproveAndPublish(ctx, owner, ApproveInput{
		SubmissionID:   sub.ID,
		ArtifactSHA256: art.SHA256,
		IdempotencyKey: "rollback-idem-key",
	})
	if err == nil {
		t.Fatal("ApproveAndPublish succeeded despite publisher failure, want error")
	}
	if code := ErrorCode(err); code != ErrCodePublicationFailed {
		t.Fatalf("ErrorCode = %q, want %q", code, ErrCodePublicationFailed)
	}

	// Verify submission is still IN_REVIEW and NOT PUBLISHED
	latest, err := repo.GetSubmission(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubmission: %v", err)
	}
	if latest.State == Published {
		t.Fatalf("submission state is %s, must NOT be PUBLISHED after failure", latest.State)
	}
}

func TestServiceCompleteAuditEventFields(t *testing.T) {
	ctx := context.Background()
	db, repo := openSQLiteTestRepository(t)
	root := t.TempDir()
	store := NewLocalArtifactStore(root, AssetPolicy{MaxBytes: 10 * 1024 * 1024})
	publisher := &fakeAtomicPublisher{}
	svc := NewPublishingService(repo, store, publisher)

	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}
	owner := Actor{Kind: ActorOwner, Identity: "owner:gio0z"}
	art := createTestArtifact(t, store, "<html><body>Audit Test</body></html>")

	// 1. Create draft with request ID and profile
	proj, sub, err := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug:            "audit-fields-test",
		Title:           "Audit Test",
		OriginalProduct: "Prod",
		RequestID:       "req-audit-1",
		Profile:         "hermes-agent",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	// 2. Mark preview ready
	sub, err = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example/audit",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
		RequestID:          "req-audit-2",
		Profile:            "hermes-agent",
	})
	if err != nil {
		t.Fatalf("MarkPreviewReady: %v", err)
	}

	// 3. Request review
	sub, err = svc.RequestReview(ctx, agent, sub.ID)
	if err != nil {
		t.Fatalf("RequestReview: %v", err)
	}

	// 4. Approve and publish
	pubRes, err := svc.ApproveAndPublish(ctx, owner, ApproveInput{
		SubmissionID:   sub.ID,
		ArtifactSHA256: art.SHA256,
		IdempotencyKey: "audit-idem-key",
		RequestID:      "req-audit-4",
		Profile:        "admin-ui",
	})
	if err != nil {
		t.Fatalf("ApproveAndPublish: %v", err)
	}
	if pubRes.LabURL == "" || pubRes.PortfolioURL == "" {
		t.Fatalf("missing URLs in publication result: %+v", pubRes)
	}

	// Query audit events from DB
	rows, err := db.QueryContext(ctx, `SELECT event_json, request_id, timestamp FROM audit_events ORDER BY sequence ASC`)
	if err != nil {
		t.Fatalf("query audit_events: %v", err)
	}
	defer rows.Close()

	var eventJSONs []string
	for rows.Next() {
		var raw, reqID, ts string
		if err := rows.Scan(&raw, &reqID, &ts); err != nil {
			t.Fatal(err)
		}
		eventJSONs = append(eventJSONs, raw)
	}

	if len(eventJSONs) < 4 {
		t.Fatalf("recorded %d audit events, want at least 4", len(eventJSONs))
	}

	// Verify approval audit event contains required fields
	lastEvent := eventJSONs[len(eventJSONs)-1]
	requiredStrings := []string{
		`"actor":{"kind":"OWNER","identity":"owner:gio0z"}`,
		`"action":"approve_and_publish"`,
		fmt.Sprintf(`"project_id":%q`, proj.ID),
		fmt.Sprintf(`"submission_id":%q`, sub.ID),
		`"new_state":"PUBLISHED"`,
		fmt.Sprintf(`"artifact_sha256":%q`, art.SHA256),
		`"result":"success"`,
		`"deployment_ids"`,
		`"destination_urls"`,
	}
	for _, req := range requiredStrings {
		if !strings.Contains(lastEvent, req) {
			t.Errorf("last audit event missing %q: %s", req, lastEvent)
		}
	}
}

func TestServiceReviewDecisions(t *testing.T) {
	ctx := context.Background()
	repo, store, _, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}
	owner := Actor{Kind: ActorOwner, Identity: "owner:gio0z"}
	art := createTestArtifact(t, store, "<html><body>Decisions</body></html>")

	// Test RequestChanges
	_, sub1, _ := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug: "changes-req", Title: "Changes Req", OriginalProduct: "Prod",
	})
	sub1, _ = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub1.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example/1",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
	})
	sub1, _ = svc.RequestReview(ctx, agent, sub1.ID)

	// Non-owner cannot request changes
	if _, err := svc.RequestChanges(ctx, agent, ReviewDecision{SubmissionID: sub1.ID}); err == nil {
		t.Fatal("agent RequestChanges succeeded, want forbidden")
	}

	// Owner requests changes
	sub1, err := svc.RequestChanges(ctx, owner, ReviewDecision{
		SubmissionID: sub1.ID,
		Reason:       "Fix contrast ratio on navbar",
	})
	if err != nil {
		t.Fatalf("owner RequestChanges error = %v", err)
	}
	if sub1.State != ChangesRequested {
		t.Fatalf("sub1 state = %s, want %s", sub1.State, ChangesRequested)
	}
	if reason, ok := sub1.PortfolioMetadata["review_reason"].(string); !ok || reason != "Fix contrast ratio on navbar" {
		t.Fatalf("sub1 review_reason = %v, want %q", sub1.PortfolioMetadata["review_reason"], "Fix contrast ratio on navbar")
	}
	persistedSub1, err := repo.GetSubmission(ctx, sub1.ID)
	if err != nil {
		t.Fatalf("GetSubmission(sub1) error = %v", err)
	}
	if reason, ok := persistedSub1.PortfolioMetadata["review_reason"].(string); !ok || reason != "Fix contrast ratio on navbar" {
		t.Fatalf("persistedSub1 review_reason = %v, want %q", persistedSub1.PortfolioMetadata["review_reason"], "Fix contrast ratio on navbar")
	}

	// Test Reject
	_, sub2, _ := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug: "reject-test", Title: "Reject Test", OriginalProduct: "Prod",
	})
	sub2, _ = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub2.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example/2",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
	})
	sub2, _ = svc.RequestReview(ctx, agent, sub2.ID)

	// Owner rejects
	sub2, err = svc.Reject(ctx, owner, ReviewDecision{
		SubmissionID: sub2.ID,
		Reason:       "Concept does not align with lab guidelines",
	})
	if err != nil {
		t.Fatalf("owner Reject error = %v", err)
	}
	if sub2.State != Rejected {
		t.Fatalf("sub2 state = %s, want %s", sub2.State, Rejected)
	}
	if reason, ok := sub2.PortfolioMetadata["review_reason"].(string); !ok || reason != "Concept does not align with lab guidelines" {
		t.Fatalf("sub2 review_reason = %v, want %q", sub2.PortfolioMetadata["review_reason"], "Concept does not align with lab guidelines")
	}
	persistedSub2, err := repo.GetSubmission(ctx, sub2.ID)
	if err != nil {
		t.Fatalf("GetSubmission(sub2) error = %v", err)
	}
	if reason, ok := persistedSub2.PortfolioMetadata["review_reason"].(string); !ok || reason != "Concept does not align with lab guidelines" {
		t.Fatalf("persistedSub2 review_reason = %v, want %q", persistedSub2.PortfolioMetadata["review_reason"], "Concept does not align with lab guidelines")
	}
}

func TestServiceArchive(t *testing.T) {
	ctx := context.Background()
	_, store, _, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}
	owner := Actor{Kind: ActorOwner, Identity: "owner:gio0z"}
	art := createTestArtifact(t, store, "<html><body>Archive</body></html>")

	_, sub, _ := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug: "archive-test", Title: "Archive Test", OriginalProduct: "Prod",
	})
	sub, _ = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example/arc",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
	})

	// Archive before publish -> invalid_state
	if err := svc.Archive(ctx, owner, sub.ID); err == nil {
		t.Fatal("Archive before publish succeeded, want error")
	}

	sub, _ = svc.RequestReview(ctx, agent, sub.ID)
	_, err := svc.ApproveAndPublish(ctx, owner, ApproveInput{
		SubmissionID:   sub.ID,
		ArtifactSHA256: art.SHA256,
		IdempotencyKey: "archive-publish-key",
	})
	if err != nil {
		t.Fatalf("ApproveAndPublish: %v", err)
	}

	// Non-owner cannot archive
	if err := svc.Archive(ctx, agent, sub.ID); err == nil {
		t.Fatal("agent Archive succeeded, want forbidden")
	}

	// Owner archives
	if err := svc.Archive(ctx, owner, sub.ID); err != nil {
		t.Fatalf("owner Archive error = %v", err)
	}
}

func TestServiceSlugConflict(t *testing.T) {
	ctx := context.Background()
	_, _, _, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}

	_, _, err := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug:            "duplicate-slug",
		Title:           "First Project",
		OriginalProduct: "Product 1",
	})
	if err != nil {
		t.Fatalf("CreateDraft 1 error = %v", err)
	}

	_, _, err = svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug:            "duplicate-slug",
		Title:           "Second Project",
		OriginalProduct: "Product 2",
	})
	if err == nil {
		t.Fatal("CreateDraft with duplicate slug succeeded, want conflict")
	}
	if code := ErrorCode(err); code != ErrCodeSlugConflict {
		t.Fatalf("ErrorCode = %q, want %q", code, ErrCodeSlugConflict)
	}
}

func TestServiceAttachAsset(t *testing.T) {
	ctx := context.Background()
	_, _, _, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}

	pngHeader := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	pngData := append(pngHeader, bytes.Repeat([]byte{0x00}, 20)...)

	ref, err := svc.AttachAsset(ctx, agent, AttachAssetInput{
		Metadata: AssetMetadata{
			Name:     "preview.png",
			MIMEType: "image/png",
		},
		Content: bytes.NewReader(pngData),
	})
	if err != nil {
		t.Fatalf("AttachAsset() error = %v", err)
	}
	if len(ref.SHA256) != 64 {
		t.Fatalf("ref.SHA256 length = %d, want 64", len(ref.SHA256))
	}
}

type auditFailingRepo struct {
	Repository
	err error
}

func (r *auditFailingRepo) AppendAudit(ctx context.Context, event AuditEvent) error {
	if r.err != nil {
		return r.err
	}
	return r.Repository.AppendAudit(ctx, event)
}

func TestServiceAttachAssetAuditError(t *testing.T) {
	ctx := context.Background()
	baseRepo, store, publisher, _ := setupServiceTest(t)
	failingRepo := &auditFailingRepo{Repository: baseRepo, err: errors.New("audit disk full")}
	svc := NewPublishingService(failingRepo, store, publisher)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}

	pngHeader := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	pngData := append(pngHeader, bytes.Repeat([]byte{0x00}, 20)...)

	_, err := svc.AttachAsset(ctx, agent, AttachAssetInput{
		Metadata: AssetMetadata{
			Name:     "preview.png",
			MIMEType: "image/png",
		},
		Content: bytes.NewReader(pngData),
	})
	if err == nil {
		t.Fatal("AttachAsset succeeded when audit append failed, want error")
	}
	if !strings.Contains(err.Error(), "audit disk full") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceSlugFormatValidation(t *testing.T) {
	ctx := context.Background()
	_, _, _, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}

	invalidSlugs := []string{
		"Uppercase-Slug",
		"../path-traversal",
		"slash/slug",
		"space in slug",
		"-leading-hyphen",
		"trailing-hyphen-",
		"double--hyphen",
		"under_score",
		"special@char",
		"",
	}

	for _, badSlug := range invalidSlugs {
		_, _, err := svc.CreateDraft(ctx, agent, CreateDraftInput{
			Slug:            badSlug,
			Title:           "Invalid Slug Test",
			OriginalProduct: "Product",
		})
		if err == nil {
			t.Errorf("CreateDraft(%q) succeeded, want validation error", badSlug)
		} else if code := ErrorCode(err); code != ErrCodeValidationFailed {
			t.Errorf("CreateDraft(%q) code = %q, want %q", badSlug, code, ErrCodeValidationFailed)
		}
	}

	validSlugs := []string{
		"mail",
		"mail-redesign",
		"v2-brand-redesign-2026",
	}
	for _, goodSlug := range validSlugs {
		proj, _, err := svc.CreateDraft(ctx, agent, CreateDraftInput{
			Slug:            goodSlug,
			Title:           "Valid Slug Test",
			OriginalProduct: "Product",
		})
		if err != nil {
			t.Errorf("CreateDraft(%q) failed: %v", goodSlug, err)
		}
		if proj.Slug != goodSlug {
			t.Errorf("CreateDraft(%q) slug = %q", goodSlug, proj.Slug)
		}
	}
}

func TestServiceRequestReviewAuthorization(t *testing.T) {
	ctx := context.Background()
	_, store, _, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}
	owner := Actor{Kind: ActorOwner, Identity: "owner:gio0z"}
	other := Actor{Kind: ActorKind("viewer"), Identity: "user:someone"}
	art := createTestArtifact(t, store, "<html><body>Review Auth</body></html>")

	_, sub, err := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug:            "review-auth-test",
		Title:           "Review Auth Test",
		OriginalProduct: "Prod",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	sub, err = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example/auth",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
	})
	if err != nil {
		t.Fatalf("MarkPreviewReady: %v", err)
	}

	// Non-agent (owner) must fail with ErrCodeForbidden
	if _, err := svc.RequestReview(ctx, owner, sub.ID); err == nil {
		t.Fatal("owner RequestReview succeeded, want forbidden")
	} else if code := ErrorCode(err); code != ErrCodeForbidden {
		t.Fatalf("owner RequestReview code = %q, want %q", code, ErrCodeForbidden)
	}

	// Non-agent (viewer) must fail with ErrCodeForbidden
	if _, err := svc.RequestReview(ctx, other, sub.ID); err == nil {
		t.Fatal("viewer RequestReview succeeded, want forbidden")
	} else if code := ErrorCode(err); code != ErrCodeForbidden {
		t.Fatalf("viewer RequestReview code = %q, want %q", code, ErrCodeForbidden)
	}

	// Agent succeeds
	sub, err = svc.RequestReview(ctx, agent, sub.ID)
	if err != nil {
		t.Fatalf("agent RequestReview failed: %v", err)
	}
	if sub.State != InReview {
		t.Fatalf("sub state = %s, want %s", sub.State, InReview)
	}
}

func TestServiceApproveAndPublishDisclaimerCheck(t *testing.T) {
	ctx := context.Background()
	repo, store, _, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}
	owner := Actor{Kind: ActorOwner, Identity: "owner:gio0z"}
	art := createTestArtifact(t, store, "<html><body>Disclaimer Check</body></html>")

	proj, sub, err := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug:            "disclaimer-check-test",
		Title:           "Disclaimer Check",
		OriginalProduct: "Prod",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	sub, err = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example/disc",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
	})
	if err != nil {
		t.Fatalf("MarkPreviewReady: %v", err)
	}

	sub, err = svc.RequestReview(ctx, agent, sub.ID)
	if err != nil {
		t.Fatalf("RequestReview: %v", err)
	}

	// Corrupt or remove mandatory disclaimer in the persisted project
	proj.Disclaimer = "Custom text without required legal statement."
	if err := repo.UpdateLabProject(ctx, proj); err != nil {
		t.Fatalf("UpdateLabProject: %v", err)
	}

	// ApproveAndPublish must fail because disclaimer lacks MandatoryDisclaimer
	_, err = svc.ApproveAndPublish(ctx, owner, ApproveInput{
		SubmissionID:   sub.ID,
		ArtifactSHA256: art.SHA256,
		IdempotencyKey: "disc-test-key",
	})
	if err == nil {
		t.Fatal("ApproveAndPublish succeeded without mandatory disclaimer, want validation error")
	}
	if code := ErrorCode(err); code != ErrCodeValidationFailed {
		t.Fatalf("ErrorCode = %q, want %q", code, ErrCodeValidationFailed)
	}
	if !strings.Contains(err.Error(), "mandatory disclaimer missing") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestServiceUpdateDraftBuildTestScanNotOverwritten(t *testing.T) {
	ctx := context.Background()
	_, store, _, svc := setupServiceTest(t)
	agent := Actor{Kind: ActorAgent, Identity: "agent:claude"}
	art := createTestArtifact(t, store, "<html><body>Build Gate Reset</body></html>")

	_, sub, err := svc.CreateDraft(ctx, agent, CreateDraftInput{
		Slug:            "build-gate-reset",
		Title:           "Build Gate Reset",
		OriginalProduct: "Prod",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	sub, err = svc.MarkPreviewReady(ctx, agent, PreviewResult{
		SubmissionID:       sub.ID,
		ArtifactSHA256:     art.SHA256,
		PreviewURL:         "https://preview.example/reset",
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
	})
	if err != nil {
		t.Fatalf("MarkPreviewReady: %v", err)
	}
	if sub.BuildResult != "passed" || sub.TestResult != "passed" || sub.SecurityScanResult != "passed" {
		t.Fatalf("preview ready results not set: build=%s test=%s scan=%s", sub.BuildResult, sub.TestResult, sub.SecurityScanResult)
	}

	// Update draft must reset build, test, and scan results to empty strings
	updatedSub, err := svc.UpdateDraft(ctx, agent, UpdateDraftInput{
		SubmissionID: sub.ID,
		Title:        "New Title",
	})
	if err != nil {
		t.Fatalf("UpdateDraft: %v", err)
	}

	if updatedSub.BuildResult != "" || updatedSub.TestResult != "" || updatedSub.SecurityScanResult != "" {
		t.Fatalf("UpdateDraft failed to reset build/test/scan: build=%q test=%q scan=%q",
			updatedSub.BuildResult, updatedSub.TestResult, updatedSub.SecurityScanResult)
	}

	// Cannot request review without passing build/test/scan again via MarkPreviewReady
	if _, err := svc.RequestReview(ctx, agent, updatedSub.ID); err == nil {
		t.Fatal("RequestReview succeeded on draft with reset build/test/scan, want error")
	}
}
