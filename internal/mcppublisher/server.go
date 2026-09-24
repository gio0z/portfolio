package mcppublisher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"portfolio/internal/publishing"
)

// MCPRoute is the Streamable HTTP endpoint served by Server.
const MCPRoute = "/mcp/portfolio"

// DefaultMaxBodyBytes bounds a single MCP tool call payload.
const DefaultMaxBodyBytes = 256 * 1024

// ServerOptions tunes the MCP server. Zero values select safe defaults.
type ServerOptions struct {
	MaxBodyBytes   int
	RequireTLS     bool
	ArtifactPolicy publishing.AssetPolicy
}

// Server is the authenticated Streamable HTTP MCP adapter. It delegates every
// tool to the domain service under a forced agent actor, so the publishing
// state machine — not the tool surface — is the final authority that stops
// agents at IN_REVIEW. Server is safe for concurrent use provided Service is.
type Server struct {
	auth       *Authenticator
	svc        Service
	maxBody    int
	requireTLS bool
	maxAsset   int64
	tools      []Tool
	byName     map[string]Tool
}

// NewServer builds the MCP server over an authenticator and domain service.
func NewServer(auth *Authenticator, svc Service, opts ServerOptions) (*Server, error) {
	if auth == nil {
		return nil, errors.New("mcppublisher: authenticator is required")
	}
	if svc == nil {
		return nil, errors.New("mcppublisher: publishing service is required")
	}
	maxBody := opts.MaxBodyBytes
	if maxBody <= 0 {
		maxBody = DefaultMaxBodyBytes
	}
	s := &Server{
		auth:       auth,
		svc:        svc,
		maxBody:    maxBody,
		requireTLS: opts.RequireTLS,
		maxAsset:   opts.ArtifactPolicy.MaxBytes,
		byName:     make(map[string]Tool),
	}
	s.tools = buildCatalog()
	for _, t := range s.tools {
		s.byName[t.Name] = t
	}
	return s, nil
}

// Tools returns the advertised tool catalog. Approval, publication,
// permission, token, and permanent-deletion tools are absent by construction.
func (s *Server) Tools() []Tool {
	out := make([]Tool, len(s.tools))
	copy(out, s.tools)
	return out
}

// Capabilities returns server capabilities. Sampling is always disabled.
func (s *Server) Capabilities() Capabilities {
	return Capabilities{
		ProtocolVersion: "2025-06-18",
		Route:           MCPRoute,
		ToolCount:       len(s.tools),
		Sampling:        false,
	}
}

// agentActor forces the agent identity server-side. Owner authority is
// unreachable through this package: no path mints or forwards an owner actor,
// so ApproveAndPublish, RequestChanges, Reject, and Archive fail closed in
// the publishing service for every MCP call.
func agentActor(id Identity) publishing.Actor {
	return publishing.Actor{Kind: publishing.ActorAgent, Identity: id.Actor.Identity}
}

// CallTool executes one tool call for an authenticated identity. Unknown
// tools are rejected, per-tool scopes are enforced, and every domain call
// runs under the agent actor so approval and publication stay unreachable.
func (s *Server) CallTool(ctx context.Context, id Identity, name string, args map[string]any) (map[string]any, error) {
	tool, ok := s.byName[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownTool, name)
	}
	if err := RequireScope(id, tool.RequiredScope); err != nil {
		return nil, &ToolError{Code: publishing.ErrCodeForbidden, Message: fmt.Sprintf("missing scope %q", tool.RequiredScope), Err: ErrForbidden}
	}
	if args == nil {
		args = map[string]any{}
	}
	if sizeGuard(args, s.maxBody) {
		return nil, &ToolError{Code: "payload_too_large", Message: "tool arguments exceed the maximum payload size"}
	}
	switch name {
	case ToolListGitHubProjects:
		var a listGitHubArgs
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return s.listGitHubProjects(ctx, id, a)
	case ToolGetGitHubProject:
		var a getGitHubArgs
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return s.getGitHubProject(ctx, id, a)
	case ToolListLabProjects:
		var a listLabArgs
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return s.listLabProjects(ctx, a)
	case ToolGetLabProject:
		var a getLabArgs
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return s.getLabProject(ctx, a)
	case ToolCreateLabDraft:
		var a createDraftArgs
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return s.createLabDraft(ctx, id, a)
	case ToolUpdateLabDraft:
		var a updateDraftArgs
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return s.updateLabDraft(ctx, id, a)
	case ToolUploadLabAsset:
		var a uploadAssetArgs
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return s.uploadLabAsset(ctx, id, a)
	case ToolDeployLabPreview:
		var a deployPreviewArgs
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return s.deployLabPreview(ctx, id, a)
	case ToolGetDeploymentStatus:
		var a submissionArgs
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return s.getDeploymentStatus(ctx, a)
	case ToolRequestReview:
		var a submissionArgs
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return s.requestReview(ctx, id, a)
	case ToolGetReviewFeedback:
		var a submissionArgs
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return s.getReviewFeedback(ctx, a)
	case ToolArchiveLabDraft:
		var a submissionArgs
		if err := decodeArgs(args, &a); err != nil {
			return nil, err
		}
		return s.archiveLabDraft(ctx, id, a)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownTool, name)
	}
}

// GitHubLister supplies repository discovery for the adapter. Production
// wiring provides a read-only GitHub App or fine-grained token behind this.
type GitHubLister interface {
	ListGitHubProjects(ctx context.Context, profileID string) ([]GitHubRepo, error)
}

// GitHubListFunc adapts a function to GitHubLister.
type GitHubListFunc func(ctx context.Context, profileID string) ([]GitHubRepo, error)

// ListGitHubProjects implements GitHubLister.
func (f GitHubListFunc) ListGitHubProjects(ctx context.Context, profileID string) ([]GitHubRepo, error) {
	return f(ctx, profileID)
}

// Adapter wires the real publishing service, registry repository, and GitHub
// discovery into the narrow Service surface Task 15 and production main will
// consume. Approval and publication stay unreachable: Adapter exposes no such
// methods, and Server forces the agent actor on every delegated call.
type Adapter struct {
	Publishing publishing.PublishingService
	Repo       publishing.Repository
	GitHub     GitHubLister
}

// ListLabProjects implements Service via the registry repository.
func (a *Adapter) ListLabProjects(ctx context.Context, filter publishing.ProjectFilter) ([]publishing.LabProject, error) {
	if a.Repo == nil {
		return nil, errors.New("mcppublisher: registry repository is not configured")
	}
	return a.Repo.ListLabProjects(ctx, filter)
}

// GetLabProject implements Service via the registry repository.
func (a *Adapter) GetLabProject(ctx context.Context, id string) (publishing.LabProject, error) {
	if a.Repo == nil {
		return publishing.LabProject{}, errors.New("mcppublisher: registry repository is not configured")
	}
	return a.Repo.GetLabProject(ctx, id)
}

// CreateDraft implements Service via the publishing service.
func (a *Adapter) CreateDraft(ctx context.Context, actor publishing.Actor, input publishing.CreateDraftInput) (publishing.LabProject, publishing.Submission, error) {
	if a.Publishing == nil {
		return publishing.LabProject{}, publishing.Submission{}, errors.New("mcppublisher: publishing service is not configured")
	}
	return a.Publishing.CreateDraft(ctx, actor, input)
}

// UpdateDraft implements Service via the publishing service.
func (a *Adapter) UpdateDraft(ctx context.Context, actor publishing.Actor, input publishing.UpdateDraftInput) (publishing.Submission, error) {
	if a.Publishing == nil {
		return publishing.Submission{}, errors.New("mcppublisher: publishing service is not configured")
	}
	return a.Publishing.UpdateDraft(ctx, actor, input)
}

// AttachAsset implements Service via the publishing service.
func (a *Adapter) AttachAsset(ctx context.Context, actor publishing.Actor, input publishing.AttachAssetInput) (publishing.ArtifactRef, error) {
	if a.Publishing == nil {
		return publishing.ArtifactRef{}, errors.New("mcppublisher: publishing service is not configured")
	}
	return a.Publishing.AttachAsset(ctx, actor, input)
}

// ListSubmissions implements Service via the registry repository.
func (a *Adapter) ListSubmissions(ctx context.Context, filter publishing.SubmissionFilter) ([]publishing.Submission, error) {
	if a.Repo == nil {
		return nil, errors.New("mcppublisher: registry repository is not configured")
	}
	return a.Repo.ListSubmissions(ctx, filter)
}

// MarkPreviewReady implements Service via the publishing service.
func (a *Adapter) MarkPreviewReady(ctx context.Context, actor publishing.Actor, result publishing.PreviewResult) (publishing.Submission, error) {
	if a.Publishing == nil {
		return publishing.Submission{}, errors.New("mcppublisher: publishing service is not configured")
	}
	return a.Publishing.MarkPreviewReady(ctx, actor, result)
}

// RequestReview implements Service via the publishing service.
func (a *Adapter) RequestReview(ctx context.Context, actor publishing.Actor, submissionID string) (publishing.Submission, error) {
	if a.Publishing == nil {
		return publishing.Submission{}, errors.New("mcppublisher: publishing service is not configured")
	}
	return a.Publishing.RequestReview(ctx, actor, submissionID)
}

// ListGitHubProjects implements Service via the injected read-only lister.
func (a *Adapter) ListGitHubProjects(ctx context.Context, profileID string) ([]GitHubRepo, error) {
	if a.GitHub == nil {
		return nil, errors.New("mcppublisher: GitHub discovery is not configured")
	}
	return a.GitHub.ListGitHubProjects(ctx, profileID)
}

// Archive implements Service via the publishing service. Agent calls fail
// closed inside the service, which restricts archival to the owner.
func (a *Adapter) Archive(ctx context.Context, actor publishing.Actor, id string) error {
	if a.Publishing == nil {
		return errors.New("mcppublisher: publishing service is not configured")
	}
	return a.Publishing.Archive(ctx, actor, id)
}

// ServeHTTP exposes the Streamable HTTP MCP endpoint. When RequireTLS is set,
// non-TLS requests are rejected at the edge; every request must carry a valid
// agent bearer token.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPRoute {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.requireTLS && r.TLS == nil && !isForwardedHTTPS(r) {
		http.Error(w, "tls required", http.StatusForbidden)
		return
	}
	id, err := s.auth.Authenticate(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(s.maxBody))
	var envelope struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      any             `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		writeRPCError(w, nil, -32700, "parse error")
		return
	}
	switch envelope.Method {
	case "tools/list":
		writeRPCResult(w, envelope.ID, map[string]any{"tools": s.Tools()})
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		dec := json.NewDecoder(bytes.NewReader(envelope.Params))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&params); err != nil {
			writeRPCError(w, envelope.ID, -32602, "invalid params")
			return
		}
		out, callErr := s.CallTool(r.Context(), id, params.Name, params.Arguments)
		if callErr != nil {
			var toolErr *ToolError
			if errors.As(callErr, &toolErr) {
				writeRPCErrorData(w, envelope.ID, -32000, toolErr.Message, map[string]any{"code": toolErr.Code})
				return
			}
			if errors.Is(callErr, ErrUnknownTool) {
				writeRPCError(w, envelope.ID, -32601, "method not found")
				return
			}
			if errors.Is(callErr, ErrForbidden) {
				writeRPCError(w, envelope.ID, -32000, "forbidden")
				return
			}
			writeRPCError(w, envelope.ID, -32603, "internal error")
			return
		}
		writeRPCResult(w, envelope.ID, map[string]any{"content": out})
	default:
		writeRPCError(w, envelope.ID, -32601, "method not found")
	}
}

func isForwardedHTTPS(r *http.Request) bool {
	proto := r.Header.Get("X-Forwarded-Proto")
	return strings.EqualFold(strings.TrimSpace(strings.Split(proto, ",")[0]), "https")
}

func writeRPCResult(w http.ResponseWriter, rpcID any, result any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0", "id": rpcID, "result": result,
	})
}

func writeRPCError(w http.ResponseWriter, rpcID any, code int, message string) {
	writeRPCErrorData(w, rpcID, code, message, nil)
}

func writeRPCErrorData(w http.ResponseWriter, rpcID any, code int, message string, data any) {
	w.Header().Set("Content-Type", "application/json")
	errObj := map[string]any{"code": code, "message": message}
	if data != nil {
		errObj["data"] = data
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0", "id": rpcID, "error": errObj,
	})
}
