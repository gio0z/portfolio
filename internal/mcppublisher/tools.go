package mcppublisher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"portfolio/internal/publishing"
)

// MCP tool names. The catalog is closed: approval, publication, permission,
// token, and permanent-deletion capabilities do not exist here by design, and
// CallTool rejects any name outside this list.
const (
	ToolListGitHubProjects  = "list_github_projects"
	ToolGetGitHubProject    = "get_github_project"
	ToolListLabProjects     = "list_lab_projects"
	ToolGetLabProject       = "get_lab_project"
	ToolCreateLabDraft      = "create_lab_draft"
	ToolUpdateLabDraft      = "update_lab_draft"
	ToolUploadLabAsset      = "upload_lab_asset"
	ToolDeployLabPreview    = "deploy_lab_preview"
	ToolGetDeploymentStatus = "get_deployment_status"
	ToolRequestReview       = "request_review"
	ToolGetReviewFeedback   = "get_review_feedback"
	ToolArchiveLabDraft     = "archive_lab_draft"
)

// GitHubRepo is the MCP-visible projection of a GitHub repository.
// Private-source publication requires explicit owner configuration and
// defaults to metadata-only; private repos are excluded from listings unless
// the caller explicitly opts in.
type GitHubRepo struct {
	Name          string `json:"name"`
	Private       bool   `json:"private"`
	HTMLURL       string `json:"html_url"`
	Description   string `json:"description,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

// Service is the narrow domain surface the MCP server delegates to. It
// mirrors the agent-reachable subset of publishing.PublishingService plus the
// read and GitHub-discovery operations the read tools need. Production wiring
// (PublishingAdapter in server.go) supplies the real publishing service,
// registry reader, and GitHub lister; approval and publication stay
// unreachable because this interface has no such methods and every call
// forces the agent actor kind.
type Service interface {
	ListLabProjects(context.Context, publishing.ProjectFilter) ([]publishing.LabProject, error)
	GetLabProject(context.Context, string) (publishing.LabProject, error)
	CreateDraft(context.Context, publishing.Actor, publishing.CreateDraftInput) (publishing.LabProject, publishing.Submission, error)
	UpdateDraft(context.Context, publishing.Actor, publishing.UpdateDraftInput) (publishing.Submission, error)
	AttachAsset(context.Context, publishing.Actor, publishing.AttachAssetInput) (publishing.ArtifactRef, error)
	ListSubmissions(context.Context, publishing.SubmissionFilter) ([]publishing.Submission, error)
	MarkPreviewReady(context.Context, publishing.Actor, publishing.PreviewResult) (publishing.Submission, error)
	RequestReview(context.Context, publishing.Actor, string) (publishing.Submission, error)
	ListGitHubProjects(context.Context, string) ([]GitHubRepo, error)
	Archive(context.Context, publishing.Actor, string) error
}

// ToolError is a stable, redacted tool-call failure. Code carries the
// publishing error code; Message carries only the authored service message,
// never wrapped internals, paths, or secrets.
type ToolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *ToolError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("mcppublisher [%s]: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("mcppublisher: %s", e.Message)
}

// Unwrap preserves errors.Is/As against the underlying sentinel.
func (e *ToolError) Unwrap() error { return e.Err }

// Tool describes one MCP tool and the scope guarding it.
type Tool struct {
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	InputSchema   map[string]any `json:"inputSchema"`
	RequiredScope string         `json:"-"`
}

// Capabilities describes the server. Sampling is disabled and — to keep the
// wire format free of any sampling advertisement — the field is never
// serialized.
type Capabilities struct {
	ProtocolVersion string `json:"protocol_version"`
	Route           string `json:"route"`
	ToolCount       int    `json:"tool_count"`
	Sampling        bool   `json:"-"`
}

// buildCatalog defines the closed tool set with per-tool scope guards.
// Approval, publication, permission, token, and permanent-deletion tools are
// absent by construction.
func buildCatalog() []Tool {
	return []Tool{
		{Name: ToolListGitHubProjects, Description: "List public GitHub projects available as redesign sources. Private repositories are excluded unless include_private is set.", InputSchema: schema(map[string]any{"include_private": prop("boolean", "Include private repositories as metadata-only entries.")}, nil), RequiredScope: ScopePortfolioRead},
		{Name: ToolGetGitHubProject, Description: "Fetch one GitHub project by name. Private repositories are hidden unless include_private is set.", InputSchema: schema(map[string]any{"name": prop("string", "Repository name."), "include_private": prop("boolean", "Allow metadata-only private entries.")}, []string{"name"}), RequiredScope: ScopePortfolioRead},
		{Name: ToolListLabProjects, Description: "List Lab projects, optionally filtered by status or featured flag.", InputSchema: schema(map[string]any{"status": prop("string", "Project status filter."), "featured": prop("boolean", "Featured filter.")}, nil), RequiredScope: ScopePortfolioRead},
		{Name: ToolGetLabProject, Description: "Fetch one Lab project with its submissions.", InputSchema: schema(map[string]any{"project_id": prop("string", "Lab project ID.")}, []string{"project_id"}), RequiredScope: ScopePortfolioRead},
		{Name: ToolCreateLabDraft, Description: "Create a Lab draft. The submission stops at DRAFT; review and beyond are separate steps.", InputSchema: schema(map[string]any{
			"slug":               prop("string", "URL-safe slug, ^[a-z0-9]+(?:-[a-z0-9]+)*$."),
			"title":              prop("string", "Project title."),
			"original_product":   prop("string", "Redesigned product name."),
			"disclaimer":         prop("string", "Disclaimer; the mandatory text is appended when absent."),
			"focus":              arrayProp("Redesign focus areas."),
			"platforms":          arrayProp("Target platforms."),
			"source_urls":        arrayProp("Optional reference links to the source site, stored as metadata only and never fetched. Each entry must be an absolute http or https URL."),
			"featured":           prop("boolean", "Feature on the homepage subset."),
			"artifact_sha256":    prop("string", "Immutable artifact hash, when already known."),
			"portfolio_metadata": prop("object", "Public portfolio metadata."),
		}, []string{"slug", "title", "original_product"}), RequiredScope: ScopeLabDraftCreate},
		{Name: ToolUpdateLabDraft, Description: "Revise a draft submission. Mutations return it to DRAFT.", InputSchema: schema(map[string]any{
			"submission_id":      prop("string", "Submission ID."),
			"title":              prop("string", "Replacement title."),
			"artifact_sha256":    prop("string", "Replacement artifact hash."),
			"preview_url":        prop("string", "Replacement preview URL."),
			"source_urls":        arrayProp("Optional reference links to the source site, stored as metadata only and never fetched. Supplying this replaces the existing reference links; each entry must be an absolute http or https URL."),
			"portfolio_metadata": prop("object", "Replacement portfolio metadata."),
		}, []string{"submission_id"}), RequiredScope: ScopeLabDraftUpdate},
		{Name: ToolUploadLabAsset, Description: "Upload one validated asset into immutable content-addressed storage.", InputSchema: schema(map[string]any{
			"name":      prop("string", "Asset file name."),
			"mime_type": prop("string", "Declared MIME type; must match detected content."),
			"content":   prop("string", "Asset content."),
		}, []string{"name", "mime_type", "content"}), RequiredScope: ScopeLabAssetUpload},
		{Name: ToolDeployLabPreview, Description: "Record a sandboxed preview deployment result for a submission.", InputSchema: schema(map[string]any{
			"submission_id":        prop("string", "Submission ID."),
			"preview_url":          prop("string", "Sandboxed preview URL."),
			"build_result":         prop("string", "Build outcome: passed or failed."),
			"test_result":          prop("string", "Test outcome."),
			"security_scan_result": prop("string", "Security scan outcome."),
			"artifact_sha256":      prop("string", "Built artifact hash."),
			"deployment_id":        prop("string", "Deployment identifier."),
		}, []string{"submission_id", "preview_url", "build_result", "test_result", "security_scan_result"}), RequiredScope: ScopeLabPreviewDeploy},
		{Name: ToolGetDeploymentStatus, Description: "Read the deployment status of a submission.", InputSchema: schema(map[string]any{"submission_id": prop("string", "Submission ID.")}, []string{"submission_id"}), RequiredScope: ScopePortfolioRead},
		{Name: ToolRequestReview, Description: "Submit a preview-ready submission for owner review. Terminal agent step: the submission stops at IN_REVIEW.", InputSchema: schema(map[string]any{"submission_id": prop("string", "Submission ID.")}, []string{"submission_id"}), RequiredScope: ScopeLabReviewRequest},
		{Name: ToolGetReviewFeedback, Description: "Read owner review feedback for a submission.", InputSchema: schema(map[string]any{"submission_id": prop("string", "Submission ID.")}, []string{"submission_id"}), RequiredScope: ScopePortfolioRead},
		{Name: ToolArchiveLabDraft, Description: "Archive a submission. Owner-only downstream: agent calls are rejected server-side.", InputSchema: schema(map[string]any{"submission_id": prop("string", "Submission ID.")}, []string{"submission_id"}), RequiredScope: ScopeLabDraftUpdate},
	}
}

func prop(typ, desc string) map[string]any {
	return map[string]any{"type": typ, "description": desc}
}

func arrayProp(desc string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
}

func schema(properties map[string]any, required []string) map[string]any {
	req := make([]any, 0, len(required))
	for _, r := range required {
		req = append(req, r)
	}
	return map[string]any{
		"type": "object", "properties": properties,
		"required": req, "additionalProperties": false,
	}
}

// sizeGuard reports whether the JSON encoding of args exceeds the limit.
func sizeGuard(args map[string]any, limit int) bool {
	if limit <= 0 {
		return false
	}
	buf, err := json.Marshal(args)
	if err != nil {
		return true
	}
	return len(buf) > limit
}

// decodeArgs re-encodes the argument map and decodes it strictly: unknown
// fields are rejected per the tool JSON schemas.
func decodeArgs(args map[string]any, dst any) error {
	raw, err := json.Marshal(args)
	if err != nil {
		return &ToolError{Code: publishing.ErrCodeValidationFailed, Message: "invalid tool arguments"}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return &ToolError{Code: publishing.ErrCodeValidationFailed, Message: fmt.Sprintf("invalid tool arguments: %v", err)}
	}
	return nil
}

// mapError translates domain failures to stable, redacted tool errors.
// Only the authored service message crosses the boundary; wrapped causes,
// paths, tokens, and secrets never do.
func mapError(err error) *ToolError {
	if err == nil {
		return nil
	}
	var toolErr *ToolError
	if errors.As(err, &toolErr) {
		return toolErr
	}
	var svcErr *publishing.ServiceError
	if errors.As(err, &svcErr) {
		return &ToolError{Code: svcErr.Code, Message: svcErr.Message, Err: err}
	}
	if errors.Is(err, publishing.ErrNotFound) {
		return &ToolError{Code: publishing.ErrCodeNotFound, Message: "resource not found", Err: err}
	}
	return &ToolError{Code: "internal_error", Message: "internal error", Err: err}
}

type listGitHubArgs struct {
	IncludePrivate bool `json:"include_private"`
}

type getGitHubArgs struct {
	Name           string `json:"name"`
	IncludePrivate bool   `json:"include_private"`
}

type listLabArgs struct {
	Status   string `json:"status"`
	Featured *bool  `json:"featured"`
}

type getLabArgs struct {
	ProjectID string `json:"project_id"`
}

type createDraftArgs struct {
	Slug              string         `json:"slug"`
	Title             string         `json:"title"`
	OriginalProduct   string         `json:"original_product"`
	Disclaimer        string         `json:"disclaimer"`
	Focus             []string       `json:"focus"`
	Platforms         []string       `json:"platforms"`
	SourceURLs        []string       `json:"source_urls"`
	Featured          bool           `json:"featured"`
	ArtifactSHA256    string         `json:"artifact_sha256"`
	PortfolioMetadata map[string]any `json:"portfolio_metadata"`
}

type updateDraftArgs struct {
	SubmissionID      string         `json:"submission_id"`
	Title             string         `json:"title"`
	ArtifactSHA256    string         `json:"artifact_sha256"`
	PreviewURL        string         `json:"preview_url"`
	SourceURLs        []string       `json:"source_urls"`
	PortfolioMetadata map[string]any `json:"portfolio_metadata"`
}

type uploadAssetArgs struct {
	Name     string `json:"name"`
	MIMEType string `json:"mime_type"`
	Content  string `json:"content"`
}

type deployPreviewArgs struct {
	SubmissionID       string `json:"submission_id"`
	PreviewURL         string `json:"preview_url"`
	BuildResult        string `json:"build_result"`
	TestResult         string `json:"test_result"`
	SecurityScanResult string `json:"security_scan_result"`
	ArtifactSHA256     string `json:"artifact_sha256"`
	DeploymentID       string `json:"deployment_id"`
}

type submissionArgs struct {
	SubmissionID string `json:"submission_id"`
}

func projectView(p publishing.LabProject) map[string]any {
	return map[string]any{
		"id": p.ID, "slug": p.Slug, "title": p.Title,
		"original_product": p.OriginalProduct, "disclaimer": p.Disclaimer,
		"focus": p.Focus, "platforms": p.Platforms,
		"source_urls": p.SourceURLs,
		"status":      p.Status, "featured": p.Featured,
		"created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
	}
}

func submissionView(sub publishing.Submission) map[string]any {
	return map[string]any{
		"submission_id": sub.ID, "lab_project_id": sub.LabProjectID,
		"revision": sub.Revision, "state": string(sub.State),
		"artifact_sha256": sub.ArtifactSHA256, "preview_url": sub.PreviewURL,
		"build_result": sub.BuildResult, "test_result": sub.TestResult,
		"security_scan_result": sub.SecurityScanResult,
		"submitted_by":         sub.SubmittedBy,
	}
}

func repoView(r GitHubRepo, metadataOnly bool) map[string]any {
	out := map[string]any{
		"name": r.Name, "private": r.Private, "html_url": r.HTMLURL,
	}
	if r.Description != "" {
		out["description"] = r.Description
	}
	if metadataOnly {
		out["metadata_only"] = true
	}
	return out
}

func (s *Server) listGitHubProjects(ctx context.Context, id Identity, a listGitHubArgs) (map[string]any, error) {
	repos, err := s.svc.ListGitHubProjects(ctx, id.ProfileID)
	if err != nil {
		return nil, mapError(err)
	}
	projects := make([]map[string]any, 0, len(repos))
	for _, r := range repos {
		if r.Private && !a.IncludePrivate {
			continue
		}
		projects = append(projects, repoView(r, r.Private))
	}
	return map[string]any{"projects": projects}, nil
}

func (s *Server) getGitHubProject(ctx context.Context, id Identity, a getGitHubArgs) (map[string]any, error) {
	if strings.TrimSpace(a.Name) == "" {
		return nil, &ToolError{Code: publishing.ErrCodeValidationFailed, Message: "name is required"}
	}
	repos, err := s.svc.ListGitHubProjects(ctx, id.ProfileID)
	if err != nil {
		return nil, mapError(err)
	}
	for _, r := range repos {
		if r.Name != a.Name {
			continue
		}
		if r.Private && !a.IncludePrivate {
			return nil, &ToolError{Code: publishing.ErrCodeNotFound, Message: "resource not found"}
		}
		return map[string]any{"project": repoView(r, r.Private)}, nil
	}
	return nil, &ToolError{Code: publishing.ErrCodeNotFound, Message: "resource not found"}
}

func (s *Server) listLabProjects(ctx context.Context, a listLabArgs) (map[string]any, error) {
	filter := publishing.ProjectFilter{Status: a.Status, Featured: a.Featured}
	projects, err := s.svc.ListLabProjects(ctx, filter)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]map[string]any, 0, len(projects))
	for _, p := range projects {
		out = append(out, projectView(p))
	}
	return map[string]any{"projects": out}, nil
}

func (s *Server) getLabProject(ctx context.Context, a getLabArgs) (map[string]any, error) {
	if strings.TrimSpace(a.ProjectID) == "" {
		return nil, &ToolError{Code: publishing.ErrCodeValidationFailed, Message: "project_id is required"}
	}
	project, err := s.svc.GetLabProject(ctx, a.ProjectID)
	if err != nil {
		return nil, mapError(err)
	}
	subs, err := s.svc.ListSubmissions(ctx, publishing.SubmissionFilter{LabProjectID: project.ID})
	if err != nil {
		return nil, mapError(err)
	}
	subViews := make([]map[string]any, 0, len(subs))
	for _, sub := range subs {
		subViews = append(subViews, submissionView(sub))
	}
	return map[string]any{"project": projectView(project), "submissions": subViews}, nil
}

func (s *Server) createLabDraft(ctx context.Context, id Identity, a createDraftArgs) (map[string]any, error) {
	project, sub, err := s.svc.CreateDraft(ctx, agentActor(id), publishing.CreateDraftInput{
		Slug: a.Slug, Title: a.Title, OriginalProduct: a.OriginalProduct,
		Disclaimer: a.Disclaimer, Focus: a.Focus, Platforms: a.Platforms,
		SourceURLs: a.SourceURLs,
		Featured:   a.Featured, ArtifactSHA256: a.ArtifactSHA256,
		PortfolioMetadata: publishing.PortfolioMetadata(a.PortfolioMetadata),
		Profile:           id.ProfileID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return map[string]any{
		"project_id": project.ID, "submission_id": sub.ID,
		"revision": sub.Revision, "state": string(sub.State),
	}, nil
}

func (s *Server) updateLabDraft(ctx context.Context, id Identity, a updateDraftArgs) (map[string]any, error) {
	sub, err := s.svc.UpdateDraft(ctx, agentActor(id), publishing.UpdateDraftInput{
		SubmissionID: a.SubmissionID, Title: a.Title,
		ArtifactSHA256: a.ArtifactSHA256, PreviewURL: a.PreviewURL,
		SourceURLs:        a.SourceURLs,
		PortfolioMetadata: publishing.PortfolioMetadata(a.PortfolioMetadata),
		Profile:           id.ProfileID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return map[string]any{
		"submission_id": sub.ID, "revision": sub.Revision, "state": string(sub.State),
	}, nil
}

func (s *Server) uploadLabAsset(ctx context.Context, id Identity, a uploadAssetArgs) (map[string]any, error) {
	if s.maxAsset > 0 && int64(len(a.Content)) > s.maxAsset {
		return nil, &ToolError{Code: "payload_too_large", Message: "asset content exceeds the maximum asset size"}
	}
	ref, err := s.svc.AttachAsset(ctx, agentActor(id), publishing.AttachAssetInput{
		Metadata: publishing.AssetMetadata{Name: a.Name, MIMEType: a.MIMEType},
		Content:  strings.NewReader(a.Content),
		Profile:  id.ProfileID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return map[string]any{
		"sha256": ref.SHA256, "mime_type": ref.MIMEType, "size": ref.Size,
	}, nil
}

func (s *Server) deployLabPreview(ctx context.Context, id Identity, a deployPreviewArgs) (map[string]any, error) {
	sub, err := s.svc.MarkPreviewReady(ctx, agentActor(id), publishing.PreviewResult{
		SubmissionID: a.SubmissionID, ArtifactSHA256: a.ArtifactSHA256,
		PreviewURL: a.PreviewURL, BuildResult: a.BuildResult,
		TestResult: a.TestResult, SecurityScanResult: a.SecurityScanResult,
		DeploymentID: a.DeploymentID, Profile: id.ProfileID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return map[string]any{
		"submission_id": sub.ID, "revision": sub.Revision,
		"state": string(sub.State), "preview_url": sub.PreviewURL,
	}, nil
}

func (s *Server) findSubmission(ctx context.Context, submissionID string) (publishing.Submission, error) {
	if strings.TrimSpace(submissionID) == "" {
		return publishing.Submission{}, &ToolError{Code: publishing.ErrCodeValidationFailed, Message: "submission_id is required"}
	}
	subs, err := s.svc.ListSubmissions(ctx, publishing.SubmissionFilter{})
	if err != nil {
		return publishing.Submission{}, mapError(err)
	}
	for _, sub := range subs {
		if sub.ID == submissionID {
			return sub, nil
		}
	}
	return publishing.Submission{}, &ToolError{Code: publishing.ErrCodeNotFound, Message: "resource not found"}
}

func (s *Server) getDeploymentStatus(ctx context.Context, a submissionArgs) (map[string]any, error) {
	sub, err := s.findSubmission(ctx, a.SubmissionID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"submission_id": sub.ID, "state": string(sub.State),
		"preview_url": sub.PreviewURL, "build_result": sub.BuildResult,
		"test_result": sub.TestResult, "security_scan_result": sub.SecurityScanResult,
		"artifact_sha256": sub.ArtifactSHA256, "revision": sub.Revision,
	}, nil
}

// requestReview is the terminal agent step. The domain service only allows
// PREVIEW_READY -> IN_REVIEW for the agent actor, so approval and publication
// are unreachable here even if a future caller widens the tool surface.
func (s *Server) requestReview(ctx context.Context, id Identity, a submissionArgs) (map[string]any, error) {
	sub, err := s.svc.RequestReview(ctx, agentActor(id), a.SubmissionID)
	if err != nil {
		return nil, mapError(err)
	}
	return map[string]any{
		"submission_id": sub.ID, "revision": sub.Revision, "state": string(sub.State),
	}, nil
}

func (s *Server) getReviewFeedback(ctx context.Context, a submissionArgs) (map[string]any, error) {
	sub, err := s.findSubmission(ctx, a.SubmissionID)
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"submission_id": sub.ID, "state": string(sub.State),
		"revision": sub.Revision, "artifact_sha256": sub.ArtifactSHA256,
	}
	if reason, ok := sub.PortfolioMetadata["review_reason"]; ok {
		out["review_reason"] = reason
	}
	return out, nil
}

// archiveLabDraft delegates with the agent actor. The publishing service
// restricts archival to the owner, so agent calls fail closed server-side;
// the tool exists so owners driving the same catalog see a stable surface.
func (s *Server) archiveLabDraft(ctx context.Context, id Identity, a submissionArgs) (map[string]any, error) {
	if err := s.svc.Archive(ctx, agentActor(id), a.SubmissionID); err != nil {
		return nil, mapError(err)
	}
	return map[string]any{"submission_id": a.SubmissionID, "archived": true}, nil
}
