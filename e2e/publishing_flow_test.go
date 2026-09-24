package e2e

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"portfolio/internal/adminauth"
	"portfolio/internal/app"
	"portfolio/internal/mcppublisher"
	"portfolio/internal/publicapi"
	"portfolio/internal/publishing"
)

// e2eSourceURL is the follower-nominated source site recorded on the draft and
// asserted to survive all the way to the public readback.
const e2eSourceURL = "https://example.com/source-site"

var e2eScopes = []string{
	mcppublisher.ScopePortfolioRead,
	mcppublisher.ScopeLabDraftCreate,
	mcppublisher.ScopeLabDraftUpdate,
	mcppublisher.ScopeLabAssetUpload,
	mcppublisher.ScopeLabPreviewDeploy,
	mcppublisher.ScopeLabReviewRequest,
}

func e2eIdentity() mcppublisher.Identity {
	return mcppublisher.Identity{
		Actor:     publishing.Actor{Kind: publishing.ActorAgent, Identity: "agent:e2e"},
		ProfileID: "e2e",
		Scopes:    e2eScopes,
		TokenID:   "e2e-token",
	}
}

func mustCall(t *testing.T, ctx context.Context, mcp *mcppublisher.Server, id mcppublisher.Identity, tool string, args map[string]any) map[string]any {
	t.Helper()
	out, err := mcp.CallTool(ctx, id, tool, args)
	if err != nil {
		t.Fatalf("CallTool(%s): %v", tool, err)
	}
	return out
}

func strField(t *testing.T, out map[string]any, key string) string {
	t.Helper()
	v, ok := out[key].(string)
	if !ok || v == "" {
		t.Fatalf("tool result missing %q: %#v", key, out)
	}
	return v
}

func TestPublishingFlow(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	fixture, err := os.ReadFile(filepath.Join("fixtures", "lab-redesign", "index.html"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	rev2Content := string(fixture) + "\n<!-- rev2 -->\n"

	a, err := app.New(app.Config{
		Env:               "test",
		FrontendDist:      dir,
		PreviewOrigin:     "https://preview.example.com",
		LabOrigin:         "https://lab.example.com",
		PortfolioOrigin:   "https://portfolio.example.com",
		RegistryPath:      filepath.Join(dir, "registry.sqlite"),
		ArtifactStoreRoot: filepath.Join(dir, "artifacts"),
		MCPTokenSecret:    strings.Repeat("m", 32),
		ApprovalVerifier:  adminauth.NewDevApprovalVerifier("test"),
	})
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })
	if a.MCP == nil {
		t.Fatal("app.MCP is nil: MCPTokenSecret not wired")
	}
	id := e2eIdentity()
	owner := publishing.Actor{Kind: publishing.ActorOwner, Identity: "owner:gio0z"}

	// 1. MCP creates draft.
	draft := mustCall(t, ctx, a.MCP, id, mcppublisher.ToolCreateLabDraft, map[string]any{
		"slug": "e2e-redesign", "title": "E2E Redesign", "original_product": "Example Product",
		"source_urls": []string{e2eSourceURL},
	})
	subID := strField(t, draft, "submission_id")

	// 2. MCP uploads fixture and deploys preview.
	up1 := mustCall(t, ctx, a.MCP, id, mcppublisher.ToolUploadLabAsset, map[string]any{
		"name": "index.html", "mime_type": "text/html", "content": string(fixture),
	})
	sha1 := strField(t, up1, "sha256")
	prev1 := mustCall(t, ctx, a.MCP, id, mcppublisher.ToolDeployLabPreview, map[string]any{
		"submission_id": subID, "preview_url": "https://preview.example.com/e2e",
		"build_result": "passed", "test_result": "passed", "security_scan_result": "passed",
		"artifact_sha256": sha1,
	})
	if prev1["state"] != string(publishing.PreviewReady) {
		t.Fatalf("after deploy state = %v, want PREVIEW_READY", prev1["state"])
	}

	// 3. MCP requests review.
	rev := mustCall(t, ctx, a.MCP, id, mcppublisher.ToolRequestReview, map[string]any{"submission_id": subID})
	if rev["state"] != string(publishing.InReview) {
		t.Fatalf("after request_review state = %v, want IN_REVIEW", rev["state"])
	}

	// Agent approval must stay unreachable through the MCP/service boundary.
	if _, err := a.Publishing.ApproveAndPublish(ctx,
		publishing.Actor{Kind: publishing.ActorAgent, Identity: "agent:e2e"},
		publishing.ApproveInput{SubmissionID: subID, ArtifactSHA256: sha1, IdempotencyKey: "e2e-agent-attempt"},
	); err == nil {
		t.Fatal("agent ApproveAndPublish succeeded, want forbidden")
	}

	// 4. Admin requests changes.
	sub, err := a.Publishing.RequestChanges(ctx, owner, publishing.ReviewDecision{SubmissionID: subID, Reason: "e2e: adjust hierarchy"})
	if err != nil {
		t.Fatalf("RequestChanges: %v", err)
	}
	if sub.State != publishing.ChangesRequested {
		t.Fatalf("after request_changes state = %v, want CHANGES_REQUESTED", sub.State)
	}

	// 5. MCP submits revision 2.
	mustCall(t, ctx, a.MCP, id, mcppublisher.ToolUpdateLabDraft, map[string]any{
		"submission_id": subID, "title": "E2E Redesign Rev2",
	})
	up2 := mustCall(t, ctx, a.MCP, id, mcppublisher.ToolUploadLabAsset, map[string]any{
		"name": "index.html", "mime_type": "text/html", "content": rev2Content,
	})
	sha2 := strField(t, up2, "sha256")
	if sha2 == sha1 {
		t.Fatal("rev2 artifact hash equals rev1: fixture mutation did not change bytes")
	}
	mustCall(t, ctx, a.MCP, id, mcppublisher.ToolDeployLabPreview, map[string]any{
		"submission_id": subID, "preview_url": "https://preview.example.com/e2e-rev2",
		"build_result": "passed", "test_result": "passed", "security_scan_result": "passed",
		"artifact_sha256": sha2,
	})
	rev2 := mustCall(t, ctx, a.MCP, id, mcppublisher.ToolRequestReview, map[string]any{"submission_id": subID})
	if rev2["state"] != string(publishing.InReview) {
		t.Fatalf("rev2 request_review state = %v, want IN_REVIEW", rev2["state"])
	}

	// 6. Admin approves revision 2 with matching hash.
	first, err := a.Publishing.ApproveAndPublish(ctx, owner, publishing.ApproveInput{
		SubmissionID: subID, ArtifactSHA256: sha2, IdempotencyKey: "e2e-key-1",
	})
	if err != nil {
		t.Fatalf("ApproveAndPublish: %v", err)
	}
	if first.ArtifactSHA256 != sha2 {
		t.Fatalf("published hash = %q, want rev2 %q", first.ArtifactSHA256, sha2)
	}

	// 7. Lab route and Design Lab API expose revision 2.
	reader := publicapi.NewRepositoryReader(a.Repo)
	published, err := reader.ListPublishedProjects(ctx)
	if err != nil {
		t.Fatalf("ListPublishedProjects: %v", err)
	}
	found := false
	for _, p := range published {
		if p.Slug == "e2e-redesign" {
			found = true
			if !strings.Contains(p.Disclaimer, publishing.MandatoryDisclaimer) {
				t.Fatalf("published disclaimer missing mandatory text: %q", p.Disclaimer)
			}
			// Source-site references must survive draft creation, review, and
			// publication to reach the public case study.
			if len(p.SourceURLs) != 1 || p.SourceURLs[0] != e2eSourceURL {
				t.Fatalf("published source_urls = %#v, want [%q]", p.SourceURLs, e2eSourceURL)
			}
		}
	}
	if !found {
		t.Fatalf("slug e2e-redesign not in published catalog (%d projects)", len(published))
	}

	// The by-slug case study read must carry the same reference links.
	caseStudy, err := reader.GetPublishedProjectBySlug(ctx, "e2e-redesign")
	if err != nil {
		t.Fatalf("GetPublishedProjectBySlug: %v", err)
	}
	if len(caseStudy.SourceURLs) != 1 || caseStudy.SourceURLs[0] != e2eSourceURL {
		t.Fatalf("case study source_urls = %#v, want [%q]", caseStudy.SourceURLs, e2eSourceURL)
	}

	// 8. Revision 1 remains unavailable: the durable submission now carries rev2 bytes.
	current, err := a.Repo.GetSubmission(ctx, subID)
	if err != nil {
		t.Fatalf("GetSubmission: %v", err)
	}
	if current.State != publishing.Published {
		t.Fatalf("submission state = %v, want PUBLISHED", current.State)
	}
	if current.ArtifactSHA256 != sha2 {
		t.Fatalf("submission hash = %q, want rev2 %q", current.ArtifactSHA256, sha2)
	}

	// 9. Audit contains every transition. AttachAsset events carry no
	// submission linkage by contract (AttachAssetInput has no
	// SubmissionID), so query the isolated registry's full audit log:
	// this test owns the registry.
	events, err := a.Repo.ListAuditEvents(ctx, publishing.AuditFilter{Limit: 200})
	if err != nil {
		t.Fatalf("ListAuditEvents: %v", err)
	}
	seen := map[string]bool{}
	for _, e := range events {
		seen[e.Action] = true
	}
	for _, want := range []string{"create_draft", "attach_asset", "mark_preview_ready", "request_review", "request_changes", "update_draft", "approve_and_publish"} {
		if !seen[want] {
			t.Fatalf("audit missing %q (have %v)", want, seen)
		}
	}

	// 10. Repeated approval returns the original publication.
	second, err := a.Publishing.ApproveAndPublish(ctx, owner, publishing.ApproveInput{
		SubmissionID: subID, ArtifactSHA256: sha2, IdempotencyKey: "e2e-key-1",
	})
	if err != nil {
		t.Fatalf("repeated ApproveAndPublish: %v", err)
	}
	if second.LabURL != first.LabURL || second.PortfolioURL != first.PortfolioURL || second.ArtifactSHA256 != first.ArtifactSHA256 {
		t.Fatalf("repeated approval differs: first=%+v second=%+v", first, second)
	}
}
