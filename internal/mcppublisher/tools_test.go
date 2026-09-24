package mcppublisher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"portfolio/internal/publishing"
)

type stubPublishingService struct {
	createFunc  func(ctx context.Context, actor publishing.Actor, input publishing.CreateDraftInput) (publishing.LabProject, publishing.Submission, error)
	updateFunc  func(ctx context.Context, actor publishing.Actor, input publishing.UpdateDraftInput) (publishing.Submission, error)
	attachFunc  func(ctx context.Context, actor publishing.Actor, input publishing.AttachAssetInput) (publishing.ArtifactRef, error)
	previewFunc func(ctx context.Context, actor publishing.Actor, result publishing.PreviewResult) (publishing.Submission, error)
	reviewFunc  func(ctx context.Context, actor publishing.Actor, submissionID string) (publishing.Submission, error)
	listProj    []publishing.LabProject
	listSub     []publishing.Submission
	listErr     error
	repos       []GitHubRepo
	reposErr    error
	archiveErr  error
	archivedID  string
}

func (s *stubPublishingService) ListLabProjects(_ context.Context, filter publishing.ProjectFilter) ([]publishing.LabProject, error) {
	return s.listProj, s.listErr
}

func (s *stubPublishingService) GetLabProject(_ context.Context, id string) (publishing.LabProject, error) {
	for _, p := range s.listProj {
		if p.ID == id {
			return p, nil
		}
	}
	return publishing.LabProject{}, publishing.NewNotFoundError("project not found", nil)
}

func (s *stubPublishingService) CreateDraft(ctx context.Context, actor publishing.Actor, input publishing.CreateDraftInput) (publishing.LabProject, publishing.Submission, error) {
	if s.createFunc != nil {
		return s.createFunc(ctx, actor, input)
	}
	return publishing.LabProject{}, publishing.Submission{}, errors.New("stub: create not configured")
}

func (s *stubPublishingService) UpdateDraft(ctx context.Context, actor publishing.Actor, input publishing.UpdateDraftInput) (publishing.Submission, error) {
	if s.updateFunc != nil {
		return s.updateFunc(ctx, actor, input)
	}
	return publishing.Submission{}, errors.New("stub: update not configured")
}

func (s *stubPublishingService) AttachAsset(ctx context.Context, actor publishing.Actor, input publishing.AttachAssetInput) (publishing.ArtifactRef, error) {
	if s.attachFunc != nil {
		return s.attachFunc(ctx, actor, input)
	}
	return publishing.ArtifactRef{}, errors.New("stub: attach not configured")
}

func (s *stubPublishingService) ListSubmissions(_ context.Context, filter publishing.SubmissionFilter) ([]publishing.Submission, error) {
	var out []publishing.Submission
	for _, sub := range s.listSub {
		if filter.LabProjectID != "" && sub.LabProjectID != filter.LabProjectID {
			continue
		}
		out = append(out, sub)
	}
	return out, s.listErr
}

func (s *stubPublishingService) MarkPreviewReady(ctx context.Context, actor publishing.Actor, result publishing.PreviewResult) (publishing.Submission, error) {
	if s.previewFunc != nil {
		return s.previewFunc(ctx, actor, result)
	}
	return publishing.Submission{}, errors.New("stub: preview not configured")
}

func (s *stubPublishingService) RequestReview(ctx context.Context, actor publishing.Actor, submissionID string) (publishing.Submission, error) {
	if s.reviewFunc != nil {
		return s.reviewFunc(ctx, actor, submissionID)
	}
	return publishing.Submission{}, errors.New("stub: review not configured")
}

func (s *stubPublishingService) ListGitHubProjects(_ context.Context, _ string) ([]GitHubRepo, error) {
	return s.repos, s.reposErr
}

func (s *stubPublishingService) Archive(_ context.Context, _ publishing.Actor, id string) error {
	s.archivedID = id
	return s.archiveErr
}

func testServer(t *testing.T, svc Service) *Server {
	t.Helper()
	auth := testAuthenticator(t)
	srv, err := NewServer(auth, svc, ServerOptions{
		MaxBodyBytes:   64 * 1024,
		RequireTLS:     false,
		ArtifactPolicy: publishing.AssetPolicy{MaxBytes: 1024 * 1024},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return srv
}

func agentIdentity(t *testing.T, auth *Authenticator, scopes []string) (Identity, string) {
	t.Helper()
	token, _, err := auth.Issue("agent:hermes-1", "internal", scopes, 3600_000_000_000)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	req := bearerRequest(token)
	id, err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	return id, token
}

func toolNames(tools []Tool) map[string]bool {
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool.Name] = true
	}
	return names
}

func TestToolCatalogContainsExpectedTools(t *testing.T) {
	srv := testServer(t, &stubPublishingService{})
	names := toolNames(srv.Tools())
	for _, want := range []string{
		ToolListGitHubProjects, ToolGetGitHubProject,
		ToolListLabProjects, ToolGetLabProject,
		ToolCreateLabDraft, ToolUpdateLabDraft,
		ToolUploadLabAsset, ToolDeployLabPreview,
		ToolGetDeploymentStatus, ToolRequestReview,
		ToolGetReviewFeedback, ToolArchiveLabDraft,
	} {
		if !names[want] {
			t.Errorf("tool catalog missing %q", want)
		}
	}
}

func TestToolCatalogExcludesForbiddenTools(t *testing.T) {
	srv := testServer(t, &stubPublishingService{})
	names := toolNames(srv.Tools())
	for name := range names {
		lower := strings.ToLower(name)
		for _, banned := range []string{"approve", "publish", "delete_permanently", "secret", "shell", "filesystem", "sql", "sampling"} {
			if strings.Contains(lower, banned) {
				t.Errorf("tool catalog exposes forbidden tool %q (matches %q)", name, banned)
			}
		}
	}
}

func TestSamplingNotAdvertised(t *testing.T) {
	srv := testServer(t, &stubPublishingService{})
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(srv.Capabilities()); err != nil {
		t.Fatalf("encode capabilities: %v", err)
	}
	if strings.Contains(strings.ToLower(buf.String()), "sampling") {
		t.Errorf("capabilities advertise sampling: %s", buf.String())
	}
	if srv.Capabilities().Sampling {
		t.Error("Capabilities().Sampling = true, want false")
	}
}

func TestCreateDraftStopsAtDraft(t *testing.T) {
	var gotActor publishing.Actor
	svc := &stubPublishingService{
		createFunc: func(_ context.Context, actor publishing.Actor, input publishing.CreateDraftInput) (publishing.LabProject, publishing.Submission, error) {
			gotActor = actor
			return publishing.LabProject{ID: "proj-1", Slug: input.Slug},
				publishing.Submission{ID: "sub-1", LabProjectID: "proj-1", Revision: 1, State: publishing.Draft}, nil
		},
	}
	srv := testServer(t, svc)
	id, _ := agentIdentity(t, srv.auth, []string{ScopeLabDraftCreate})
	out, callErr := srv.CallTool(context.Background(), id, ToolCreateLabDraft, map[string]any{
		"slug":             "mail-redesign",
		"title":            "Mail Redesign",
		"original_product": "Mail",
	})
	if callErr != nil {
		t.Fatalf("CallTool(create_lab_draft) error = %v", callErr)
	}
	if out["state"] != string(publishing.Draft) {
		t.Errorf("created state = %v, want %q", out["state"], publishing.Draft)
	}
	if gotActor.Kind != publishing.ActorAgent {
		t.Errorf("service actor kind = %q, want AGENT (owner-only approve/publish unreachable)", gotActor.Kind)
	}
}

func TestApprovePublishAbsentForAgentIdentity(t *testing.T) {
	srv := testServer(t, &stubPublishingService{})
	for _, attempt := range []string{"approve_lab_submission", "publish_lab_submission", "lab_approve", "lab_publish", "approve", "publish"} {
		_, err := srv.CallTool(context.Background(), mustAgent(t, srv), attempt, map[string]any{})
		if err == nil {
			t.Errorf("CallTool(%q) succeeded for agent identity, want rejection", attempt)
		} else if !errors.Is(err, ErrUnknownTool) && !errors.Is(err, ErrForbidden) {
			t.Errorf("CallTool(%q) error = %v, want unknown-tool or forbidden", attempt, err)
		}
	}
}

func mustAgent(t *testing.T, srv *Server) Identity {
	t.Helper()
	id, _ := agentIdentity(t, srv.auth, AgentScopes())
	return id
}

func TestRequestReviewStopsAtInReview(t *testing.T) {
	var gotActor publishing.Actor
	svc := &stubPublishingService{
		reviewFunc: func(_ context.Context, actor publishing.Actor, submissionID string) (publishing.Submission, error) {
			gotActor = actor
			return publishing.Submission{ID: submissionID, Revision: 3, State: publishing.InReview}, nil
		},
	}
	srv := testServer(t, svc)
	out, err := srv.CallTool(context.Background(), mustAgent(t, srv), ToolRequestReview, map[string]any{
		"submission_id": "sub-1",
	})
	if err != nil {
		t.Fatalf("CallTool(request_review) error = %v", err)
	}
	if out["state"] != string(publishing.InReview) {
		t.Errorf("review state = %v, want IN_REVIEW (never APPROVED/PUBLISHED)", out["state"])
	}
	if gotActor.Kind != publishing.ActorAgent {
		t.Errorf("service actor kind = %q, want AGENT", gotActor.Kind)
	}
}

func TestUploadArtifactDelegatesToService(t *testing.T) {
	var gotName, gotMIME string
	var gotSize int64
	svc := &stubPublishingService{
		attachFunc: func(_ context.Context, _ publishing.Actor, input publishing.AttachAssetInput) (publishing.ArtifactRef, error) {
			gotName, gotMIME = input.Metadata.Name, input.Metadata.MIMEType
			content, _ := io.ReadAll(input.Content)
			gotSize = int64(len(content))
			return publishing.ArtifactRef{SHA256: strings.Repeat("a", 64), MIMEType: gotMIME, Size: gotSize}, nil
		},
	}
	srv := testServer(t, svc)
	out, err := srv.CallTool(context.Background(), mustAgent(t, srv), ToolUploadLabAsset, map[string]any{
		"name":      "index.html",
		"mime_type": "text/html",
		"content":   "<h1>hello</h1>",
	})
	if err != nil {
		t.Fatalf("CallTool(upload_lab_asset) error = %v", err)
	}
	if out["sha256"] != strings.Repeat("a", 64) {
		t.Errorf("sha256 = %v, want stub hash", out["sha256"])
	}
	if gotName != "index.html" || gotMIME != "text/html" || gotSize == 0 {
		t.Errorf("service input = (%q, %q, %d), want validated asset metadata", gotName, gotMIME, gotSize)
	}
}

func TestUploadArtifactRejectsUnknownFields(t *testing.T) {
	srv := testServer(t, &stubPublishingService{})
	_, err := srv.CallTool(context.Background(), mustAgent(t, srv), ToolUploadLabAsset, map[string]any{
		"name":         "index.html",
		"mime_type":    "text/html",
		"content":      "<h1>hi</h1>",
		"shell_escape": true,
	})
	if err == nil {
		t.Fatal("CallTool with unknown field succeeded, want validation error")
	}
}

func TestUploadArtifactRejectsOversizedPayload(t *testing.T) {
	svc := &stubPublishingService{
		attachFunc: func(_ context.Context, _ publishing.Actor, _ publishing.AttachAssetInput) (publishing.ArtifactRef, error) {
			return publishing.ArtifactRef{}, nil
		},
	}
	srv := testServer(t, svc)
	big := strings.Repeat("a", 2*1024*1024)
	_, err := srv.CallTool(context.Background(), mustAgent(t, srv), ToolUploadLabAsset, map[string]any{
		"name":      "index.html",
		"mime_type": "text/html",
		"content":   big,
	})
	if err == nil {
		t.Fatal("CallTool with oversized payload succeeded, want rejection")
	}
}

func TestServiceErrorsMappedWithoutSecrets(t *testing.T) {
	svc := &stubPublishingService{
		createFunc: func(_ context.Context, _ publishing.Actor, _ publishing.CreateDraftInput) (publishing.LabProject, publishing.Submission, error) {
			return publishing.LabProject{}, publishing.Submission{},
				publishing.NewForbiddenError("only agent or owner may create a draft")
		},
	}
	srv := testServer(t, svc)
	_, err := srv.CallTool(context.Background(), mustAgent(t, srv), ToolCreateLabDraft, map[string]any{
		"slug":             "x",
		"title":            "X",
		"original_product": "Y",
	})
	var toolErr *ToolError
	if !errors.As(err, &toolErr) {
		t.Fatalf("CallTool error type = %T, want *ToolError", err)
	}
	if toolErr.Code != publishing.ErrCodeForbidden {
		t.Errorf("ToolError.Code = %q, want %q", toolErr.Code, publishing.ErrCodeForbidden)
	}
	if strings.Contains(strings.ToLower(toolErr.Message), "secret") || strings.Contains(toolErr.Message, "/var/") {
		t.Errorf("ToolError leaks internals: %q", toolErr.Message)
	}
}

func TestPrivateGitHubReposExcludedByDefault(t *testing.T) {
	svc := &stubPublishingService{
		repos: []GitHubRepo{
			{Name: "public-shop", Private: false, HTMLURL: "https://github.com/gio0z/public-shop"},
			{Name: "secret-banking", Private: true, HTMLURL: "https://github.com/gio0z/secret-banking"},
		},
	}
	srv := testServer(t, svc)
	out, err := srv.CallTool(context.Background(), mustAgent(t, srv), ToolListGitHubProjects, map[string]any{})
	if err != nil {
		t.Fatalf("CallTool(list_github_projects) error = %v", err)
	}
	projects, ok := out["projects"].([]map[string]any)
	if !ok {
		t.Fatalf("projects type = %T, want []map[string]any", out["projects"])
	}
	if len(projects) != 1 || projects[0]["name"] != "public-shop" {
		t.Errorf("default listing = %v, want only public repos", out["projects"])
	}
	if strings.Contains(jsonString(out), "secret-banking") {
		t.Errorf("default listing leaks private repo: %v", out["projects"])
	}
	outAll, err := srv.CallTool(context.Background(), mustAgent(t, srv), ToolListGitHubProjects, map[string]any{
		"include_private": true,
	})
	if err != nil {
		t.Fatalf("CallTool(include_private) error = %v", err)
	}
	if !strings.Contains(jsonString(outAll), "secret-banking") {
		t.Errorf("include_private listing = %v, want private repo with metadata-only flag", outAll["projects"])
	}
}

func jsonString(v any) string {
	buf, _ := json.Marshal(v)
	return string(buf)
}

// newDraftBackend wires the MCP Service surface to the real publishing service
// over SQLite, so source_urls round-trips through validation, the registry, and
// the project view rather than a stub echo.
func newDraftBackend(t *testing.T) *Adapter {
	t.Helper()
	repo, _, svc := newPublishingBackend(t)
	return &Adapter{Publishing: svc, Repo: repo}
}

// createDraftViaTool creates a draft through the tool and returns the project
// ID, the submission ID, and the raw tool result.
func createDraftViaTool(t *testing.T, srv *Server, args map[string]any) (string, string, map[string]any) {
	t.Helper()
	out, err := srv.CallTool(context.Background(), mustAgent(t, srv), ToolCreateLabDraft, args)
	if err != nil {
		t.Fatalf("CallTool(create_lab_draft) error = %v", err)
	}
	projectID, _ := out["project_id"].(string)
	submissionID, _ := out["submission_id"].(string)
	if projectID == "" || submissionID == "" {
		t.Fatalf("create result = %v, want project_id and submission_id", out)
	}
	return projectID, submissionID, out
}

// draftSourceURLs reads a project back through the read tool and returns the
// source_urls the agent sees.
func draftSourceURLs(t *testing.T, srv *Server, projectID string) []string {
	t.Helper()
	out, err := srv.CallTool(context.Background(), mustAgent(t, srv), ToolGetLabProject, map[string]any{
		"project_id": projectID,
	})
	if err != nil {
		t.Fatalf("CallTool(get_lab_project) error = %v", err)
	}
	project, ok := out["project"].(map[string]any)
	if !ok {
		t.Fatalf("project type = %T, want map[string]any", out["project"])
	}
	raw, ok := project["source_urls"].([]string)
	if !ok {
		t.Fatalf("project source_urls type = %T, want []string", project["source_urls"])
	}
	return raw
}

func TestDraftToolSchemasDeclareSourceURLs(t *testing.T) {
	srv := testServer(t, &stubPublishingService{})
	for _, tc := range []struct {
		tool string
		want string
	}{
		{ToolCreateLabDraft, "reference links to the source site"},
		{ToolUpdateLabDraft, "replaces the existing reference links"},
	} {
		var found *Tool
		for _, tool := range srv.Tools() {
			if tool.Name == tc.tool {
				candidate := tool
				found = &candidate
				break
			}
		}
		if found == nil {
			t.Fatalf("tool %q not in catalog", tc.tool)
		}
		properties, ok := found.InputSchema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s properties type = %T, want map[string]any", tc.tool, found.InputSchema["properties"])
		}
		sourceURLs, ok := properties["source_urls"].(map[string]any)
		if !ok {
			t.Fatalf("%s schema is missing the source_urls property", tc.tool)
		}
		if sourceURLs["type"] != "array" {
			t.Errorf("%s source_urls type = %v, want array", tc.tool, sourceURLs["type"])
		}
		items, ok := sourceURLs["items"].(map[string]any)
		if !ok || items["type"] != "string" {
			t.Errorf("%s source_urls items = %v, want string items", tc.tool, sourceURLs["items"])
		}
		desc, _ := sourceURLs["description"].(string)
		if !strings.Contains(desc, tc.want) {
			t.Errorf("%s source_urls description = %q, want it to contain %q", tc.tool, desc, tc.want)
		}
		if !strings.Contains(desc, "http") {
			t.Errorf("%s source_urls description = %q, want it to state the http/https requirement", tc.tool, desc)
		}
		required, ok := found.InputSchema["required"].([]any)
		if !ok {
			t.Fatalf("%s required type = %T, want []any", tc.tool, found.InputSchema["required"])
		}
		for _, r := range required {
			if r == "source_urls" {
				t.Errorf("%s lists source_urls as required, want optional", tc.tool)
			}
		}
	}
}

func TestCreateLabDraftPersistsSourceURLs(t *testing.T) {
	srv := testServer(t, newDraftBackend(t))
	projectID, _, _ := createDraftViaTool(t, srv, map[string]any{
		"slug": "mail-redesign", "title": "Mail Redesign", "original_product": "Mail",
		"source_urls": []string{"https://source.example/mail", "https://source.example/mail-v2"},
	})
	got := draftSourceURLs(t, srv, projectID)
	want := []string{"https://source.example/mail", "https://source.example/mail-v2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("source_urls = %v, want %v", got, want)
	}
	// The read path must round-trip through the registry, not just the create
	// response, so a second independent read repeats the assertion.
	if again := draftSourceURLs(t, srv, projectID); !reflect.DeepEqual(again, want) {
		t.Errorf("source_urls on reread = %v, want %v", again, want)
	}
}

func TestCreateLabDraftWithoutSourceURLsSucceeds(t *testing.T) {
	srv := testServer(t, newDraftBackend(t))
	projectID, _, _ := createDraftViaTool(t, srv, map[string]any{
		"slug": "no-source", "title": "No Source", "original_product": "Widget",
	})
	if got := draftSourceURLs(t, srv, projectID); len(got) != 0 {
		t.Errorf("source_urls = %v, want empty when the optional field is omitted", got)
	}
}

func TestCreateLabDraftRejectsNonHTTPSourceURL(t *testing.T) {
	srv := testServer(t, newDraftBackend(t))
	_, err := srv.CallTool(context.Background(), mustAgent(t, srv), ToolCreateLabDraft, map[string]any{
		"slug": "dangerous-source", "title": "Dangerous", "original_product": "Widget",
		"source_urls": []string{"javascript:alert(1)"},
	})
	if err == nil {
		t.Fatal("CallTool with javascript: source URL succeeded, want validation failure")
	}
	var toolErr *ToolError
	if !errors.As(err, &toolErr) {
		t.Fatalf("error = %v (%T), want *ToolError", err, err)
	}
	if toolErr.Code != publishing.ErrCodeValidationFailed {
		t.Errorf("error code = %q, want %q", toolErr.Code, publishing.ErrCodeValidationFailed)
	}
	if publishing.ErrorCode(err) != publishing.ErrCodeValidationFailed {
		t.Errorf("publishing.ErrorCode(err) = %q, want %q", publishing.ErrorCode(err), publishing.ErrCodeValidationFailed)
	}
}

func TestUpdateLabDraftReplacesSourceURLs(t *testing.T) {
	srv := testServer(t, newDraftBackend(t))
	projectID, submissionID, _ := createDraftViaTool(t, srv, map[string]any{
		"slug": "mail-redesign", "title": "Mail Redesign", "original_product": "Mail",
		"source_urls": []string{"https://old.example/mail"},
	})
	out, err := srv.CallTool(context.Background(), mustAgent(t, srv), ToolUpdateLabDraft, map[string]any{
		"submission_id": submissionID,
		"source_urls":   []string{"https://new.example/mail"},
	})
	if err != nil {
		t.Fatalf("CallTool(update_lab_draft) error = %v", err)
	}
	if out["state"] != string(publishing.Draft) {
		t.Errorf("state = %v, want %q after a source mutation", out["state"], publishing.Draft)
	}
	got := draftSourceURLs(t, srv, projectID)
	want := []string{"https://new.example/mail"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("source_urls = %v, want replacement %v", got, want)
	}
}

func TestUpdateLabDraftOmittedSourceURLsLeavesLinksIntact(t *testing.T) {
	srv := testServer(t, newDraftBackend(t))
	original := []string{"https://source.example/mail"}
	projectID, submissionID, _ := createDraftViaTool(t, srv, map[string]any{
		"slug": "mail-redesign", "title": "Mail Redesign", "original_product": "Mail",
		"source_urls": original,
	})
	if _, err := srv.CallTool(context.Background(), mustAgent(t, srv), ToolUpdateLabDraft, map[string]any{
		"submission_id": submissionID,
		"title":         "Mail Redesign v2",
	}); err != nil {
		t.Fatalf("CallTool(update_lab_draft) error = %v", err)
	}
	if got := draftSourceURLs(t, srv, projectID); !reflect.DeepEqual(got, original) {
		t.Errorf("source_urls = %v, want unchanged %v when the field is omitted", got, original)
	}
}

func TestScopeEnforcedPerTool(t *testing.T) {
	srv := testServer(t, &stubPublishingService{})
	readOnly, _ := agentIdentity(t, srv.auth, []string{ScopePortfolioRead})
	if _, err := srv.CallTool(context.Background(), readOnly, ToolCreateLabDraft, map[string]any{
		"slug": "x", "title": "X", "original_product": "Y",
	}); !errors.Is(err, ErrForbidden) {
		t.Errorf("create without scope error = %v, want forbidden", err)
	}
	draftOnly, _ := agentIdentity(t, srv.auth, []string{ScopeLabDraftCreate})
	if _, err := srv.CallTool(context.Background(), draftOnly, ToolRequestReview, map[string]any{
		"submission_id": "sub-1",
	}); !errors.Is(err, ErrForbidden) {
		t.Errorf("review without scope error = %v, want forbidden", err)
	}
}
