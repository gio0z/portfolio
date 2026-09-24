package adminapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"portfolio/internal/adminapi"
	"portfolio/internal/adminauth"
	"portfolio/internal/publishing"
)

type mockAtomicPublisher struct {
	publishFunc func(context.Context, publishing.PublicationRequest) (publishing.PublicationOutput, error)
	calls       int
}

func (m *mockAtomicPublisher) Publish(ctx context.Context, req publishing.PublicationRequest) (publishing.PublicationOutput, error) {
	m.calls++
	if m.publishFunc != nil {
		return m.publishFunc(ctx, req)
	}
	return publishing.PublicationOutput{
		LabURL:          "https://lab.gio0z.dev/" + req.Project.Slug,
		PortfolioURL:    "https://gio0z.dev/design-lab/" + req.Project.Slug,
		DeploymentIDs:   []string{"dep-lab-123", "dep-port-456"},
		DestinationURLs: []string{"https://lab.gio0z.dev/" + req.Project.Slug, "https://gio0z.dev/design-lab/" + req.Project.Slug},
	}, nil
}

type testHarness struct {
	repo         *publishing.SQLiteRepository
	service      publishing.PublishingService
	artifacts    *publishing.LocalArtifactStore
	publisher    *mockAtomicPublisher
	authService  *adminauth.Service
	handler      *adminapi.Handler
	mux          *http.ServeMux
	ownerCookie  *http.Cookie
	ownerCSRF    string
	attackerSess *http.Cookie
	attackerCSRF string
}

func setupTestHarness(t *testing.T) *testHarness {
	t.Helper()

	db, err := publishing.OpenRegistry(":memory:")
	if err != nil {
		t.Fatalf("OpenRegistry: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := publishing.NewSQLiteRepository(db)
	publisher := &mockAtomicPublisher{}
	artifacts := publishing.NewLocalArtifactStore(t.TempDir(), publishing.AssetPolicy{MaxBytes: 10 * 1024 * 1024})
	pubService := publishing.NewPublishingService(repo, artifacts, publisher)

	authCfg := adminauth.Config{
		ClientID:         "test-client-id",
		ClientSecret:     "test-client-secret",
		SessionSecret:    "super-secret-session-signing-key-32b!",
		AllowedLogin:     "gio0z",
		Environment:      "development",
		ApprovalVerifier: adminauth.NewDevApprovalVerifier("development"),
		SecureCookies:    false,
		SessionStore:     adminauth.NewMemorySessionStore(),
	}
	authSvc, err := adminauth.NewService(authCfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	// Create owner session
	ownerSess, ownerToken, err := authSvc.CreateSession("gio0z")
	if err != nil {
		t.Fatalf("CreateSession owner: %v", err)
	}
	ownerCSRF, err := authSvc.IssueCSRFToken(ownerSess.ID)
	if err != nil {
		t.Fatalf("IssueCSRFToken owner: %v", err)
	}
	ownerCookie := &http.Cookie{
		Name:  adminauth.SessionCookieName,
		Value: ownerToken,
	}

	// Create non-owner attacker session
	attackerSess, attackerToken, err := authSvc.CreateSession("attacker-hacker")
	if err != nil {
		t.Fatalf("CreateSession attacker: %v", err)
	}
	attackerCSRF, err := authSvc.IssueCSRFToken(attackerSess.ID)
	if err != nil {
		t.Fatalf("IssueCSRFToken attacker: %v", err)
	}
	attackerCookie := &http.Cookie{
		Name:  adminauth.SessionCookieName,
		Value: attackerToken,
	}

	handler := adminapi.NewHandler(pubService, repo)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, authSvc)

	return &testHarness{
		repo:         repo,
		service:      pubService,
		artifacts:    artifacts,
		publisher:    publisher,
		authService:  authSvc,
		handler:      handler,
		mux:          mux,
		ownerCookie:  ownerCookie,
		ownerCSRF:    ownerCSRF,
		attackerSess: attackerCookie,
		attackerCSRF: attackerCSRF,
	}
}

// seedProjectAndSubmission creates a lab project and submission in IN_REVIEW state for testing.
func seedProjectAndSubmission(t *testing.T, h *testHarness, slug, title string, state publishing.SubmissionState, scanResult string) (publishing.LabProject, publishing.Submission) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	proj := publishing.LabProject{
		ID:              "proj-" + slug,
		Slug:            slug,
		Title:           title,
		OriginalProduct: "Legacy App",
		Disclaimer:      publishing.MandatoryDisclaimer,
		Focus:           []string{"design", "systems"},
		Platforms:       []string{"web"},
		Status:          "DRAFT",
		Featured:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := h.repo.CreateLabProject(ctx, proj); err != nil {
		t.Fatalf("seed CreateLabProject: %v", err)
	}

	// Store an artifact in the artifact store
	ref, err := h.artifacts.Put(ctx, strings.NewReader("<html><body>Test "+slug+"</body></html>"), publishing.AssetMetadata{
		Name:     "index.html",
		MIMEType: "text/html",
	})
	if err != nil {
		t.Fatalf("seed Put artifact: %v", err)
	}

	sub := publishing.Submission{
		ID:                 "sub-" + slug,
		LabProjectID:       proj.ID,
		Revision:           1,
		State:              state,
		ArtifactSHA256:     ref.SHA256,
		PreviewURL:         "https://preview.example.com/" + slug,
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: scanResult,
		PortfolioMetadata: publishing.PortfolioMetadata{
			"cover_image": "https://example.com/cover.png",
			"summary":     "Redesign summary",
		},
		SubmittedBy: "agent-hermes",
		SubmittedAt: now,
		UpdatedAt:   now,
	}
	if err := h.repo.CreateSubmission(ctx, sub); err != nil {
		t.Fatalf("seed CreateSubmission: %v", err)
	}

	return proj, sub
}

func TestAdminAPI_AuthenticationRequired(t *testing.T) {
	h := setupTestHarness(t)
	seedProjectAndSubmission(t, h, "auth-test", "Auth Test", publishing.InReview, "passed")

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/admin/overview"},
		{http.MethodGet, "/api/admin/reviews"},
		{http.MethodGet, "/api/admin/reviews/sub-auth-test"},
		{http.MethodPost, "/api/admin/reviews/sub-auth-test/request-changes"},
		{http.MethodPost, "/api/admin/reviews/sub-auth-test/reject"},
		{http.MethodPost, "/api/admin/reviews/sub-auth-test/approve-and-publish"},
		{http.MethodPost, "/api/admin/projects/proj-auth-test/archive"},
		{http.MethodGet, "/api/admin/audit"},
	}

	for _, tc := range routes {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(`{}`)))
			req.Header.Set("Content-Type", "application/json")
			// No session cookie provided!
			rr := httptest.NewRecorder()
			h.mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected status 401 Unauthorized, got %d for %s %s", rr.Code, tc.method, tc.path)
			}
		})
	}
}

func TestAdminAPI_AuthorizationForbiddenForNonOwner(t *testing.T) {
	h := setupTestHarness(t)
	seedProjectAndSubmission(t, h, "forbidden-test", "Forbidden Test", publishing.InReview, "passed")

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/admin/overview"},
		{http.MethodGet, "/api/admin/reviews"},
		{http.MethodGet, "/api/admin/reviews/sub-forbidden-test"},
		{http.MethodPost, "/api/admin/reviews/sub-forbidden-test/request-changes"},
		{http.MethodPost, "/api/admin/reviews/sub-forbidden-test/reject"},
		{http.MethodPost, "/api/admin/reviews/sub-forbidden-test/approve-and-publish"},
		{http.MethodPost, "/api/admin/projects/proj-forbidden-test/archive"},
		{http.MethodGet, "/api/admin/audit"},
	}

	for _, tc := range routes {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(`{}`)))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(h.attackerSess)
			req.Header.Set(adminauth.CSRFHeaderName, h.attackerCSRF)
			req.Header.Set("X-Request-ID", "req-attacker")
			rr := httptest.NewRecorder()
			h.mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Fatalf("expected status 403 Forbidden, got %d for %s %s", rr.Code, tc.method, tc.path)
			}
		})
	}
}

func TestAdminAPI_CSRFValidationOnMutations(t *testing.T) {
	h := setupTestHarness(t)
	seedProjectAndSubmission(t, h, "csrf-test", "CSRF Test", publishing.InReview, "passed")

	mutationRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/admin/reviews/sub-csrf-test/request-changes"},
		{http.MethodPost, "/api/admin/reviews/sub-csrf-test/reject"},
		{http.MethodPost, "/api/admin/reviews/sub-csrf-test/approve-and-publish"},
		{http.MethodPost, "/api/admin/projects/proj-csrf-test/archive"},
	}

	for _, tc := range mutationRoutes {
		t.Run("MissingCSRF_"+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(`{}`)))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(h.ownerCookie)
			req.Header.Set("X-Request-ID", "req-test-csrf")
			// Omit CSRF header!
			rr := httptest.NewRecorder()
			h.mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Fatalf("expected 403 Forbidden for missing CSRF, got %d", rr.Code)
			}
		})

		t.Run("InvalidCSRF_"+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(`{}`)))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(h.ownerCookie)
			req.Header.Set(adminauth.CSRFHeaderName, "bogus-forged-csrf-token")
			req.Header.Set("X-Request-ID", "req-test-csrf")
			rr := httptest.NewRecorder()
			h.mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Fatalf("expected 403 Forbidden for invalid CSRF, got %d", rr.Code)
			}
		})
	}
}

func TestAdminAPI_RequiredRequestID(t *testing.T) {
	h := setupTestHarness(t)
	seedProjectAndSubmission(t, h, "reqid-test", "Req ID Test", publishing.InReview, "passed")

	mutationRoutes := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/admin/reviews/sub-reqid-test/request-changes", `{"reason":"Need edits"}`},
		{http.MethodPost, "/api/admin/reviews/sub-reqid-test/reject", `{"reason":"Not good"}`},
		{http.MethodPost, "/api/admin/reviews/sub-reqid-test/approve-and-publish", `{"artifact_sha256":"1111111111222222222233333333334444444444555555555566666666667777","idempotency_key":"k1"}`},
		{http.MethodPost, "/api/admin/projects/proj-reqid-test/archive", `{}`},
	}

	for _, tc := range mutationRoutes {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(h.ownerCookie)
			req.Header.Set(adminauth.CSRFHeaderName, h.ownerCSRF)
			req.Header.Set(adminauth.DevStepUpHeader, "true")
			// Omit X-Request-ID header!
			rr := httptest.NewRecorder()
			h.mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request for missing X-Request-ID, got %d. Body: %s", rr.Code, rr.Body.String())
			}

			var resp map[string]interface{}
			_ = json.Unmarshal(rr.Body.Bytes(), &resp)
			errMsg, _ := resp["error"].(string)
			if !strings.Contains(strings.ToLower(errMsg), "request-id") && !strings.Contains(strings.ToLower(errMsg), "request id") {
				t.Errorf("expected error message to mention request ID, got %q", errMsg)
			}
		})
	}
}

func TestAdminAPI_OverviewDTO(t *testing.T) {
	h := setupTestHarness(t)
	seedProjectAndSubmission(t, h, "ov-rev1", "Overview Review 1", publishing.InReview, "passed")
	seedProjectAndSubmission(t, h, "ov-rev2", "Overview Review 2", publishing.InReview, "failed")

	// Add an audit event
	_ = h.repo.AppendAudit(context.Background(), publishing.AuditEvent{
		Timestamp:    time.Now().UTC(),
		RequestID:    "req-ov-1",
		Actor:        publishing.Actor{Kind: publishing.ActorOwner, Identity: "gio0z"},
		Action:       "overview_check",
		ProjectID:    "proj-ov-rev1",
		SubmissionID: "sub-ov-rev1",
		Result:       "success",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/overview", nil)
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var overview adminapi.OverviewResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &overview); err != nil {
		t.Fatalf("decode overview: %v", err)
	}

	if overview.PendingReviews != 2 {
		t.Errorf("expected 2 pending reviews, got %d", overview.PendingReviews)
	}
	if overview.BuildFailures < 1 {
		t.Errorf("expected at least 1 build/scan failure, got %d", overview.BuildFailures)
	}
	if len(overview.RecentAudits) == 0 {
		t.Errorf("expected recent audits to be populated")
	}
}

func TestAdminAPI_ListReviews(t *testing.T) {
	h := setupTestHarness(t)
	seedProjectAndSubmission(t, h, "lr-1", "Queue Item 1", publishing.InReview, "passed")
	seedProjectAndSubmission(t, h, "lr-2", "Queue Item 2", publishing.InReview, "failed")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/reviews", nil)
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var items []adminapi.ReviewQueueItem
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode review list: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items in review queue, got %d", len(items))
	}

	// Verify required fields on queue items
	for _, item := range items {
		if item.ID == "" || item.ProjectTitle == "" || item.OriginalProduct == "" || item.ArtifactSHA256 == "" {
			t.Errorf("missing expected fields in review item: %+v", item)
		}
		if item.SecurityScanResult == "passed" && !item.CanApprove {
			t.Errorf("item with passed scan and build should be approvable: %+v", item)
		}
		if item.SecurityScanResult == "failed" && item.CanApprove {
			t.Errorf("item with failed scan MUST NOT be approvable: %+v", item)
		}
	}
}

func TestAdminAPI_GetReviewDetail(t *testing.T) {
	h := setupTestHarness(t)
	proj, sub := seedProjectAndSubmission(t, h, "detail-1", "Detail Project", publishing.InReview, "passed")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/reviews/"+sub.ID, nil)
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var detail adminapi.ReviewDetailResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode review detail: %v", err)
	}

	if detail.Submission.ID != sub.ID {
		t.Errorf("expected submission ID %s, got %s", sub.ID, detail.Submission.ID)
	}
	if detail.Project.ID != proj.ID {
		t.Errorf("expected project ID %s, got %s", proj.ID, detail.Project.ID)
	}
	if detail.Artifact.SHA256 != sub.ArtifactSHA256 {
		t.Errorf("expected artifact sha256 %s, got %s", sub.ArtifactSHA256, detail.Artifact.SHA256)
	}
	if !detail.CanApprove {
		t.Errorf("expected CanApprove to be true for passed review")
	}

	// Non-existent review returns 404
	req404 := httptest.NewRequest(http.MethodGet, "/api/admin/reviews/sub-does-not-exist", nil)
	req404.AddCookie(h.ownerCookie)
	rr404 := httptest.NewRecorder()
	h.mux.ServeHTTP(rr404, req404)
	if rr404.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent review, got %d", rr404.Code)
	}
}

// Source URLs are review provenance: the owner must be able to see the
// third-party source site the redesign was derived from before approving.
func TestAdminAPI_ReviewDetailExposesSourceURLs(t *testing.T) {
	h := setupTestHarness(t)

	proj, sub := seedProjectAndSubmission(t, h, "source-urls", "Source URL Project", publishing.InReview, "passed")
	proj.SourceURLs = []string{"https://example.com/original", "https://example.com/pricing"}
	if err := h.repo.UpdateLabProject(context.Background(), proj); err != nil {
		t.Fatalf("UpdateLabProject: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/reviews/"+sub.ID, nil)
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var detail adminapi.ReviewDetailResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode review detail: %v", err)
	}
	want := []string{"https://example.com/original", "https://example.com/pricing"}
	if !reflect.DeepEqual(detail.Project.SourceURLs, want) {
		t.Fatalf("source_urls = %#v, want %#v", detail.Project.SourceURLs, want)
	}
	// The frontend reads this exact wire key, so pin the name rather than only
	// the Go-side field.
	if !strings.Contains(rr.Body.String(), `"source_urls"`) {
		t.Fatalf("review detail JSON is missing the source_urls key: %s", rr.Body.String())
	}
}

func TestAdminAPI_RequestChanges(t *testing.T) {
	h := setupTestHarness(t)
	_, sub := seedProjectAndSubmission(t, h, "req-changes", "Request Changes Project", publishing.InReview, "passed")

	payload := `{"reason":"Please refine the typography and spacing."}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/"+sub.ID+"/request-changes", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-change-123")
	req.Header.Set(adminauth.CSRFHeaderName, h.ownerCSRF)
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp adminapi.DecisionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode decision response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success true")
	}
	if resp.Submission.State != publishing.ChangesRequested {
		t.Errorf("expected submission state CHANGES_REQUESTED, got %s", resp.Submission.State)
	}

	// Verify database submission state changed
	updated, err := h.repo.GetSubmission(context.Background(), sub.ID)
	if err != nil {
		t.Fatalf("GetSubmission: %v", err)
	}
	if updated.State != publishing.ChangesRequested {
		t.Errorf("expected DB submission state %s, got %s", publishing.ChangesRequested, updated.State)
	}
}

func TestAdminAPI_Reject(t *testing.T) {
	h := setupTestHarness(t)
	_, sub := seedProjectAndSubmission(t, h, "reject-proj", "Reject Project", publishing.InReview, "passed")

	payload := `{"reason":"Concept does not fit the design system."}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/"+sub.ID+"/reject", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-reject-123")
	req.Header.Set(adminauth.CSRFHeaderName, h.ownerCSRF)
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp adminapi.DecisionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode decision response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success true")
	}
	if resp.Submission.State != publishing.Rejected {
		t.Errorf("expected submission state REJECTED, got %s", resp.Submission.State)
	}

	// Verify database submission state changed
	updated, err := h.repo.GetSubmission(context.Background(), sub.ID)
	if err != nil {
		t.Fatalf("GetSubmission: %v", err)
	}
	if updated.State != publishing.Rejected {
		t.Errorf("expected DB submission state %s, got %s", publishing.Rejected, updated.State)
	}
}

func TestAdminAPI_ApproveAndPublish_StepUpRequired(t *testing.T) {
	h := setupTestHarness(t)
	_, sub := seedProjectAndSubmission(t, h, "stepup-proj", "Step Up Project", publishing.InReview, "passed")

	payload := fmt.Sprintf(`{"artifact_sha256":"%s","idempotency_key":"stepup-key-1"}`, sub.ArtifactSHA256)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/"+sub.ID+"/approve-and-publish", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-stepup-1")
	req.Header.Set(adminauth.CSRFHeaderName, h.ownerCSRF)
	req.AddCookie(h.ownerCookie)
	// Omit step-up header (X-Admin-StepUp-Dev)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden without step-up authentication, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminAPI_ApproveAndPublish_ArtifactMismatch(t *testing.T) {
	h := setupTestHarness(t)
	_, sub := seedProjectAndSubmission(t, h, "mismatch-proj", "Mismatch Project", publishing.InReview, "passed")

	payload := `{"artifact_sha256":"wronghashwronghashwronghashwronghashwronghashwronghashwronghashwrong","idempotency_key":"mismatch-k1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/"+sub.ID+"/approve-and-publish", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-mismatch-1")
	req.Header.Set(adminauth.CSRFHeaderName, h.ownerCSRF)
	req.Header.Set(adminauth.DevStepUpHeader, "true")
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusConflict {
		t.Fatalf("expected 400 or 409 for artifact mismatch, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var errResp adminapi.ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if errResp.Code != publishing.ErrCodeArtifactMismatch {
		t.Errorf("expected error code %s, got %s", publishing.ErrCodeArtifactMismatch, errResp.Code)
	}
}

func TestAdminAPI_ApproveAndPublish_InvalidState(t *testing.T) {
	h := setupTestHarness(t)
	// Seed with DRAFT state instead of IN_REVIEW
	_, sub := seedProjectAndSubmission(t, h, "draft-state-proj", "Draft Project", publishing.Draft, "passed")

	payload := fmt.Sprintf(`{"artifact_sha256":"%s","idempotency_key":"draft-state-k1"}`, sub.ArtifactSHA256)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/"+sub.ID+"/approve-and-publish", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-invalid-state")
	req.Header.Set(adminauth.CSRFHeaderName, h.ownerCSRF)
	req.Header.Set(adminauth.DevStepUpHeader, "true")
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict && rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 409 Conflict or 400 for invalid state, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminAPI_ApproveAndPublish_AtomicSuccessAndIdempotency(t *testing.T) {
	h := setupTestHarness(t)
	_, sub := seedProjectAndSubmission(t, h, "atomic-publish", "Atomic Publish Project", publishing.InReview, "passed")

	payload := fmt.Sprintf(`{"artifact_sha256":"%s","idempotency_key":"idem-atomic-key-999"}`, sub.ArtifactSHA256)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/"+sub.ID+"/approve-and-publish", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-publish-1")
	req.Header.Set("Idempotency-Key", "idem-atomic-key-999")
	req.Header.Set(adminauth.CSRFHeaderName, h.ownerCSRF)
	req.Header.Set(adminauth.DevStepUpHeader, "true")
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on publication, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var pubResp adminapi.PublicationResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &pubResp); err != nil {
		t.Fatalf("decode publication response: %v", err)
	}

	if !pubResp.Success {
		t.Errorf("expected publication success true")
	}
	if pubResp.LabURL == "" || pubResp.PortfolioURL == "" {
		t.Errorf("expected dual-destination URLs to be populated: %+v", pubResp)
	}
	if h.publisher.calls != 1 {
		t.Errorf("expected atomic publisher called once, got %d", h.publisher.calls)
	}

	// Verify submission is now PUBLISHED in database
	updated, err := h.repo.GetSubmission(context.Background(), sub.ID)
	if err != nil {
		t.Fatalf("GetSubmission: %v", err)
	}
	if updated.State != publishing.Published {
		t.Errorf("expected DB submission state %s, got %s", publishing.Published, updated.State)
	}

	// Second request with SAME idempotency key returns exact same cached publication
	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/"+sub.ID+"/approve-and-publish", strings.NewReader(payload))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Request-ID", "req-publish-2")
	req2.Header.Set("Idempotency-Key", "idem-atomic-key-999")
	req2.Header.Set(adminauth.CSRFHeaderName, h.ownerCSRF)
	req2.Header.Set(adminauth.DevStepUpHeader, "true")
	req2.AddCookie(h.ownerCookie)
	rr2 := httptest.NewRecorder()
	h.mux.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on idempotent retry, got %d. Body: %s", rr2.Code, rr2.Body.String())
	}

	var pubResp2 adminapi.PublicationResponse
	if err := json.Unmarshal(rr2.Body.Bytes(), &pubResp2); err != nil {
		t.Fatalf("decode idempotent publication response: %v", err)
	}

	if pubResp2.LabURL != pubResp.LabURL || pubResp2.PortfolioURL != pubResp.PortfolioURL {
		t.Errorf("idempotent publication returned differing URLs")
	}
	// Atomic publisher MUST NOT have been invoked again!
	if h.publisher.calls != 1 {
		t.Errorf("atomic publisher should NOT have been invoked on idempotent replay, total calls: %d", h.publisher.calls)
	}
}

func TestAdminAPI_ArchiveProject(t *testing.T) {
	h := setupTestHarness(t)
	proj, sub := seedProjectAndSubmission(t, h, "archive-proj", "Archive Project", publishing.Published, "passed")

	req := httptest.NewRequest(http.MethodPost, "/api/admin/projects/"+sub.ID+"/archive", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-archive-123")
	req.Header.Set(adminauth.CSRFHeaderName, h.ownerCSRF)
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["success"] != true {
		t.Errorf("expected success true, got %v", resp["success"])
	}

	// Verify project status in DB
	updated, err := h.repo.GetSubmission(context.Background(), sub.ID)
	if err != nil {
		t.Fatalf("GetSubmission: %v", err)
	}
	if updated.State != publishing.Archived {
		t.Errorf("expected submission state ARCHIVED, got %s", updated.State)
	}
	_ = proj
}

func TestAdminAPI_AuditLog(t *testing.T) {
	h := setupTestHarness(t)

	// Append two audit events
	ctx := context.Background()
	_ = h.repo.AppendAudit(ctx, publishing.AuditEvent{
		Timestamp:    time.Now().UTC(),
		RequestID:    "audit-req-1",
		Actor:        publishing.Actor{Kind: publishing.ActorOwner, Identity: "gio0z"},
		Action:       "review_queue_read",
		SubmissionID: "sub-audit-1",
		Result:       "success",
	})
	_ = h.repo.AppendAudit(ctx, publishing.AuditEvent{
		Timestamp:    time.Now().UTC(),
		RequestID:    "audit-req-2",
		Actor:        publishing.Actor{Kind: publishing.ActorAgent, Identity: "agent-1"},
		Action:       "create_draft",
		SubmissionID: "sub-audit-2",
		Result:       "success",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/audit", nil)
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var audits []adminapi.AuditSummary
	if err := json.Unmarshal(rr.Body.Bytes(), &audits); err != nil {
		t.Fatalf("decode audit events: %v", err)
	}

	if len(audits) < 2 {
		t.Fatalf("expected at least 2 audit events, got %d", len(audits))
	}
	if audits[0].RequestID == "" || audits[0].Action == "" {
		t.Errorf("audit summary missing required fields: %+v", audits[0])
	}
}

func TestAdminAPI_UnknownFieldsRejected(t *testing.T) {
	h := setupTestHarness(t)
	_, sub := seedProjectAndSubmission(t, h, "unknown-fld", "Unknown Field Project", publishing.InReview, "passed")

	// Payload with unknown field "malicious_injection"
	payload := `{"reason":"Need fix", "malicious_injection":"bad_data"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/"+sub.ID+"/request-changes", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-unknown-1")
	req.Header.Set(adminauth.CSRFHeaderName, h.ownerCSRF)
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for unknown field, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminAPI_SanitizedErrors(t *testing.T) {
	h := setupTestHarness(t)

	// Trigger error by sending invalid JSON with internal looking data
	payload := `{"reason": 123}` // wrong type
	req := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/sub-none/request-changes", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-sanitize-1")
	req.Header.Set(adminauth.CSRFHeaderName, h.ownerCSRF)
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	body := rr.Body.String()
	// Assert no internal filesystem paths, bearer tokens, or SQL error details are leaked
	for _, forbidden := range []string{"/home/", "/var/", "sqlite3", "sql:", "Bearer", "goroutine"} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(forbidden)) {
			t.Errorf("error response leaked internal details %q: %s", forbidden, body)
		}
	}
}

func TestAdminAPI_PayloadSizeLimit(t *testing.T) {
	h := setupTestHarness(t)
	_, sub := seedProjectAndSubmission(t, h, "size-limit", "Size Limit Project", publishing.InReview, "passed")

	// Create payload larger than 1MB
	oversized := strings.Repeat("A", 1024*1024+100)
	payload := fmt.Sprintf(`{"reason":"%s"}`, oversized)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/reviews/"+sub.ID+"/request-changes", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-oversized-1")
	req.Header.Set(adminauth.CSRFHeaderName, h.ownerCSRF)
	req.AddCookie(h.ownerCookie)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 400 or 413 for oversized body, got %d", rr.Code)
	}
}
