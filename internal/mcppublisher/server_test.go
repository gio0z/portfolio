package mcppublisher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"portfolio/internal/publishing"
)

// fakeAtomicPublisher supplies the AtomicPublisher dependency for service
// construction in tests; publication itself is never exercised via MCP.
type fakeAtomicPublisher struct{}

func (fakeAtomicPublisher) Publish(_ context.Context, req publishing.PublicationRequest) (publishing.PublicationOutput, error) {
	return publishing.PublicationOutput{
		LabURL:       "https://lab.example.com/" + req.Project.Slug,
		PortfolioURL: "https://portfolio.example.com/lab/" + req.Project.Slug,
	}, nil
}

// newPublishingBackend builds a real publishing service over SQLite so the
// agent-cannot-approve test exercises the actual server-side enforcement.
func newPublishingBackend(t *testing.T) (*publishing.SQLiteRepository, *publishing.LocalArtifactStore, publishing.PublishingService) {
	t.Helper()
	db, err := publishing.OpenRegistry(filepath.Join(t.TempDir(), "registry.db"))
	if err != nil {
		t.Fatalf("OpenRegistry() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := publishing.NewSQLiteRepository(db)
	store := publishing.NewLocalArtifactStore(t.TempDir(), publishing.AssetPolicy{MaxBytes: 1024 * 1024})
	svc := publishing.NewPublishingService(repo, store, fakeAtomicPublisher{})
	return repo, store, svc
}

func testAdapterService(t *testing.T) (*Adapter, *stubPublishingService) {
	t.Helper()
	stub := &stubPublishingService{}
	adapter := &Adapter{
		Publishing: stubAdapterPublishing{stub},
		Repo:       stubAdapterRepo{stub},
		GitHub: GitHubListFunc(func(_ context.Context, _ string) ([]GitHubRepo, error) {
			return stub.repos, stub.reposErr
		}),
	}
	return adapter, stub
}

// stubAdapterPublishing routes Adapter calls to the shared test stub so the
// adapter contract test exercises real delegation logic.
type stubAdapterPublishing struct{ stub *stubPublishingService }

func (a stubAdapterPublishing) CreateDraft(ctx context.Context, actor publishing.Actor, input publishing.CreateDraftInput) (publishing.LabProject, publishing.Submission, error) {
	return a.stub.CreateDraft(ctx, actor, input)
}

func (a stubAdapterPublishing) UpdateDraft(ctx context.Context, actor publishing.Actor, input publishing.UpdateDraftInput) (publishing.Submission, error) {
	return a.stub.UpdateDraft(ctx, actor, input)
}

func (a stubAdapterPublishing) AttachAsset(ctx context.Context, actor publishing.Actor, input publishing.AttachAssetInput) (publishing.ArtifactRef, error) {
	return a.stub.AttachAsset(ctx, actor, input)
}

func (a stubAdapterPublishing) MarkPreviewReady(ctx context.Context, actor publishing.Actor, result publishing.PreviewResult) (publishing.Submission, error) {
	return a.stub.MarkPreviewReady(ctx, actor, result)
}

func (a stubAdapterPublishing) RequestReview(ctx context.Context, actor publishing.Actor, submissionID string) (publishing.Submission, error) {
	return a.stub.RequestReview(ctx, actor, submissionID)
}

func (a stubAdapterPublishing) RequestChanges(ctx context.Context, actor publishing.Actor, decision publishing.ReviewDecision) (publishing.Submission, error) {
	return publishing.Submission{}, publishing.NewForbiddenError("stub: request changes not configured")
}

func (a stubAdapterPublishing) Reject(ctx context.Context, actor publishing.Actor, decision publishing.ReviewDecision) (publishing.Submission, error) {
	return publishing.Submission{}, publishing.NewForbiddenError("stub: reject not configured")
}

func (a stubAdapterPublishing) ApproveAndPublish(ctx context.Context, actor publishing.Actor, input publishing.ApproveInput) (publishing.PublicationResult, error) {
	return publishing.PublicationResult{}, publishing.NewForbiddenError("stub: approve not configured")
}

func (a stubAdapterPublishing) Archive(ctx context.Context, actor publishing.Actor, id string) error {
	return a.stub.Archive(ctx, actor, id)
}

type stubAdapterRepo struct{ stub *stubPublishingService }

func (a stubAdapterRepo) CreateLabProject(context.Context, publishing.LabProject) error {
	return errors.New("stub")
}

func (a stubAdapterRepo) GetLabProject(ctx context.Context, id string) (publishing.LabProject, error) {
	return a.stub.GetLabProject(ctx, id)
}

func (a stubAdapterRepo) ListLabProjects(ctx context.Context, filter publishing.ProjectFilter) ([]publishing.LabProject, error) {
	return a.stub.ListLabProjects(ctx, filter)
}

func (a stubAdapterRepo) UpdateLabProject(context.Context, publishing.LabProject) error {
	return errors.New("stub")
}

func (a stubAdapterRepo) CreateSubmission(context.Context, publishing.Submission) error {
	return errors.New("stub")
}

func (a stubAdapterRepo) GetSubmission(context.Context, string) (publishing.Submission, error) {
	return publishing.Submission{}, publishing.NewNotFoundError("not found", nil)
}

func (a stubAdapterRepo) UpdateSubmission(context.Context, publishing.Submission, int64) error {
	return errors.New("stub")
}

func (a stubAdapterRepo) RecordApproval(context.Context, publishing.Approval) error {
	return errors.New("stub")
}

func (a stubAdapterRepo) GetIdempotencyResult(context.Context, string) (*publishing.OperationResult, error) {
	return nil, nil
}

func (a stubAdapterRepo) RecordIdempotencyResult(context.Context, string, publishing.OperationResult) error {
	return errors.New("stub")
}

func (a stubAdapterRepo) AppendAudit(context.Context, publishing.AuditEvent) error { return nil }

func (a stubAdapterRepo) ListSubmissions(ctx context.Context, filter publishing.SubmissionFilter) ([]publishing.Submission, error) {
	return a.stub.ListSubmissions(ctx, filter)
}

func (a stubAdapterRepo) ListAuditEvents(context.Context, publishing.AuditFilter) ([]publishing.AuditEvent, error) {
	return nil, nil
}

func (a stubAdapterRepo) WithTx(ctx context.Context, fn func(publishing.Repository) error) error {
	return fn(a)
}

func TestAdapterExposesNoApprovalOrPublish(t *testing.T) {
	adapter, _ := testAdapterService(t)
	srv := testServer(t, adapter)
	names := toolNames(srv.Tools())
	for _, banned := range []string{"approve", "publish", "secret", "shell", "filesystem", "sql", "sampling"} {
		for name := range names {
			if strings.Contains(strings.ToLower(name), banned) {
				t.Errorf("adapter-backed catalog exposes %q (matches %q)", name, banned)
			}
		}
	}
}

func TestAdapterAgentCannotApproveOrPublishServerSide(t *testing.T) {
	repo, _, svc := newPublishingBackend(t)
	adapter := &Adapter{
		Publishing: svc,
		Repo:       repo,
		GitHub:     GitHubListFunc(func(context.Context, string) ([]GitHubRepo, error) { return nil, nil }),
	}
	auth := testAuthenticator(t)
	srv, err := NewServer(auth, adapter, ServerOptions{})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	agent := Identity{
		Actor:     publishing.Actor{Kind: publishing.ActorAgent, Identity: "agent:hermes-1"},
		ProfileID: "internal",
		Scopes:    AgentScopes(),
	}

	// End-to-end through the real service: draft -> preview -> IN_REVIEW, then
	// prove the agent actor cannot advance beyond IN_REVIEW.
	project, sub, err := svc.CreateDraft(context.Background(),
		publishing.Actor{Kind: publishing.ActorAgent, Identity: "agent:hermes-1"},
		publishing.CreateDraftInput{Slug: "mail-redesign", Title: "Mail Redesign", OriginalProduct: "Mail"})
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	_ = project

	if _, err := svc.MarkPreviewReady(context.Background(),
		publishing.Actor{Kind: publishing.ActorAgent, Identity: "agent:hermes-1"},
		publishing.PreviewResult{SubmissionID: sub.ID, PreviewURL: "https://preview.example/sub",
			BuildResult: "passed", TestResult: "passed", SecurityScanResult: "passed"}); err != nil {
		t.Fatalf("MarkPreviewReady() error = %v", err)
	}
	reviewed, err := srv.CallTool(context.Background(), agent, ToolRequestReview, map[string]any{
		"submission_id": sub.ID,
	})
	if err != nil {
		t.Fatalf("CallTool(request_review) error = %v", err)
	}
	if reviewed["state"] != string(publishing.InReview) {
		t.Fatalf("review state = %v, want IN_REVIEW", reviewed["state"])
	}

	// No approve/publish tool exists for the agent identity...
	for _, attempt := range []string{"approve", "publish", "lab_approve", "lab_publish"} {
		if _, err := srv.CallTool(context.Background(), agent, attempt, map[string]any{}); !errors.Is(err, ErrUnknownTool) {
			t.Errorf("CallTool(%q) error = %v, want unknown tool", attempt, err)
		}
	}
	// ...and the domain service itself rejects agent approval server-side.
	owner := publishing.Actor{Kind: publishing.ActorOwner, Identity: "owner:gio0z"}
	agentActor := publishing.Actor{Kind: publishing.ActorAgent, Identity: "agent:hermes-1"}
	if _, err := svc.ApproveAndPublish(context.Background(), agentActor, publishing.ApproveInput{
		SubmissionID: sub.ID, ArtifactSHA256: "abc", IdempotencyKey: "idem-1",
	}); publishing.ErrorCode(err) != publishing.ErrCodeForbidden {
		t.Errorf("ApproveAndPublish(agent) code = %q, want forbidden", publishing.ErrorCode(err))
	}
	if _, err := svc.RequestChanges(context.Background(), agentActor, publishing.ReviewDecision{SubmissionID: sub.ID}); publishing.ErrorCode(err) != publishing.ErrCodeForbidden {
		t.Errorf("RequestChanges(agent) code = %q, want forbidden", publishing.ErrorCode(err))
	}
	if _, err := svc.Reject(context.Background(), agentActor, publishing.ReviewDecision{SubmissionID: sub.ID}); publishing.ErrorCode(err) != publishing.ErrCodeForbidden {
		t.Errorf("Reject(agent) code = %q, want forbidden", publishing.ErrorCode(err))
	}
	_ = owner
}

func TestStreamableHTTPRequiresAuth(t *testing.T) {
	srv := testServer(t, &stubPublishingService{})
	call := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(call); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPRoute, &buf)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated status = %d, want 401", rec.Code)
	}
}

func TestStreamableHTTPToolsList(t *testing.T) {
	srv := testServer(t, &stubPublishingService{})
	token, _, err := srv.auth.Issue("agent:hermes-1", "internal", []string{ScopePortfolioRead}, 3600_000_000_000)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	call := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(call); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPRoute, &buf)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("tools/list status = %d, want 200", rec.Code)
	}
	var envelope struct {
		Result struct {
			Tools []Tool `json:"tools"`
		} `json:"result"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode tools/list: %v", err)
	}
	if len(envelope.Result.Tools) == 0 {
		t.Fatal("tools/list returned no tools")
	}
	for _, tool := range envelope.Result.Tools {
		if strings.Contains(strings.ToLower(tool.Name), "approve") || strings.Contains(strings.ToLower(tool.Name), "publish") {
			t.Errorf("tools/list exposes forbidden tool %q", tool.Name)
		}
	}
}

func TestStreamableHTTPRejectsUnknownMethod(t *testing.T) {
	srv := testServer(t, &stubPublishingService{})
	token, _, err := srv.auth.Issue("agent:hermes-1", "internal", []string{ScopePortfolioRead}, 3600_000_000_000)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	call := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "sampling/createMessage"}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(call); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPRoute, &buf)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var envelope struct {
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Error.Code != -32601 {
		t.Errorf("sampling method error code = %d, want -32601", envelope.Error.Code)
	}
}

func TestStreamableHTTPEnforcesTLSWhenRequired(t *testing.T) {
	auth := testAuthenticator(t)
	srv, err := NewServer(auth, &stubPublishingService{}, ServerOptions{RequireTLS: true})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	call := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(call); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://example.com"+MCPRoute, &buf)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("plaintext status = %d, want 403", rec.Code)
	}
}
