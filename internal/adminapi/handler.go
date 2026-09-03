package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"portfolio/internal/adminauth"
	"portfolio/internal/publishing"
)

// Handler implements the administrative review, publication, and audit HTTP API.
type Handler struct {
	service publishing.PublishingService
	repo    publishing.Repository
}

// NewHandler constructs an administrative API handler.
func NewHandler(service publishing.PublishingService, repo publishing.Repository) *Handler {
	return &Handler{
		service: service,
		repo:    repo,
	}
}

// Routes returns an http.Handler that routes all admin API endpoints with the given auth service.
func (h *Handler) Routes(authSvc *adminauth.Service) http.Handler {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux, authSvc)
	return mux
}

// RegisterRoutes mounts admin review API endpoints on the provided ServeMux.
// Read endpoints are protected by OwnerMiddleware.
// Mutation endpoints are protected by OwnerMiddleware and CSRFMiddleware.
// Approve-and-publish is additionally protected by RequireStepUp.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, authSvc *adminauth.Service) {
	wrapOwner := func(next http.HandlerFunc) http.Handler {
		if authSvc != nil {
			return authSvc.OwnerMiddleware(next)
		}
		return next
	}

	wrapMutation := func(next http.HandlerFunc) http.Handler {
		if authSvc != nil {
			return authSvc.OwnerMiddleware(authSvc.CSRFMiddleware(next))
		}
		return next
	}

	wrapStepUp := func(next http.HandlerFunc) http.Handler {
		if authSvc != nil {
			return authSvc.OwnerMiddleware(authSvc.CSRFMiddleware(authSvc.RequireStepUp(next)))
		}
		return next
	}

	// Read endpoints
	mux.Handle("GET /api/admin/overview", wrapOwner(h.HandleOverview))
	mux.Handle("GET /api/admin/reviews", wrapOwner(h.HandleListReviews))
	mux.Handle("GET /api/admin/reviews/{id}", wrapOwner(h.HandleGetReviewDetail))
	mux.Handle("GET /api/admin/audit", wrapOwner(h.HandleListAudit))

	// Mutation endpoints
	mux.Handle("POST /api/admin/reviews/{id}/request-changes", wrapMutation(h.HandleRequestChanges))
	mux.Handle("POST /api/admin/reviews/{id}/reject", wrapMutation(h.HandleReject))
	mux.Handle("POST /api/admin/reviews/{id}/approve-and-publish", wrapStepUp(h.HandleApproveAndPublish))
	mux.Handle("POST /api/admin/projects/{id}/archive", wrapMutation(h.HandleArchiveProject))
}

// HandleOverview returns summary statistics for the admin dashboard.
func (h *Handler) HandleOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Pending reviews (submissions in IN_REVIEW)
	subs, err := h.repo.ListSubmissions(ctx, publishing.SubmissionFilter{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query submissions", "internal_error")
		return
	}

	pendingReviews := 0
	buildFailures := 0
	for _, sub := range subs {
		if sub.State == publishing.InReview {
			pendingReviews++
		}
		if !strings.EqualFold(sub.BuildResult, "passed") ||
			!strings.EqualFold(sub.TestResult, "passed") ||
			!strings.EqualFold(sub.SecurityScanResult, "passed") {
			buildFailures++
		}
	}

	// 2. Published projects count
	projects, err := h.repo.ListLabProjects(ctx, publishing.ProjectFilter{Status: "PUBLISHED"})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query published projects", "internal_error")
		return
	}

	// 3. Recent audit events
	auditEvents, err := h.repo.ListAuditEvents(ctx, publishing.AuditFilter{Limit: 10})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query audit events", "internal_error")
		return
	}

	recentAudits := make([]AuditSummary, 0, len(auditEvents))
	for _, evt := range auditEvents {
		recentAudits = append(recentAudits, cleanAuditSummary(evt))
	}

	writeJSON(w, http.StatusOK, OverviewResponse{
		PendingReviews:    pendingReviews,
		PublishedProjects: len(projects),
		BuildFailures:     buildFailures,
		RecentAudits:      recentAudits,
	})
}

// HandleListReviews returns the submissions in the review queue.
func (h *Handler) HandleListReviews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stateParam := r.URL.Query().Get("state")

	var filter publishing.SubmissionFilter
	if stateParam != "" && !strings.EqualFold(stateParam, "all") {
		filter.State = publishing.SubmissionState(stateParam)
	}

	subs, err := h.repo.ListSubmissions(ctx, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list submissions", "internal_error")
		return
	}

	items := make([]ReviewQueueItem, 0, len(subs))
	for _, sub := range subs {
		proj, _ := h.repo.GetLabProject(ctx, sub.LabProjectID)
		canApprove := sub.State == publishing.InReview &&
			strings.EqualFold(sub.BuildResult, "passed") &&
			strings.EqualFold(sub.TestResult, "passed") &&
			strings.EqualFold(sub.SecurityScanResult, "passed")

		items = append(items, ReviewQueueItem{
			ID:                 sub.ID,
			ProjectID:          sub.LabProjectID,
			ProjectSlug:        proj.Slug,
			ProjectTitle:       proj.Title,
			OriginalProduct:    proj.OriginalProduct,
			Revision:           sub.Revision,
			State:              string(sub.State),
			ArtifactSHA256:     sub.ArtifactSHA256,
			PreviewURL:         sub.PreviewURL,
			BuildResult:        sub.BuildResult,
			TestResult:         sub.TestResult,
			SecurityScanResult: sub.SecurityScanResult,
			CanApprove:         canApprove,
			SubmittedBy:        sub.SubmittedBy,
			SubmittedAt:        sub.SubmittedAt,
			UpdatedAt:          sub.UpdatedAt,
			PortfolioMetadata:  sub.PortfolioMetadata,
		})
	}

	writeJSON(w, http.StatusOK, items)
}

// HandleGetReviewDetail returns full details for a single submission in review.
func (h *Handler) HandleGetReviewDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := extractID(r, "/api/admin/reviews/", "")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review ID is required", "validation_failed")
		return
	}

	sub, err := h.repo.GetSubmission(ctx, id)
	if err != nil {
		if errors.Is(err, publishing.ErrNotFound) {
			writeError(w, http.StatusNotFound, "review not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to retrieve review", "internal_error")
		return
	}

	proj, err := h.repo.GetLabProject(ctx, sub.LabProjectID)
	if err != nil && !errors.Is(err, publishing.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "failed to retrieve project", "internal_error")
		return
	}

	canApprove := sub.State == publishing.InReview &&
		strings.EqualFold(sub.BuildResult, "passed") &&
		strings.EqualFold(sub.TestResult, "passed") &&
		strings.EqualFold(sub.SecurityScanResult, "passed")

	writeJSON(w, http.StatusOK, ReviewDetailResponse{
		Submission: sub,
		Project:    proj,
		Artifact: ArtifactDetail{
			SHA256: sub.ArtifactSHA256,
		},
		CanApprove: canApprove,
	})
}

// HandleRequestChanges allows the owner to send a submission back with feedback.
func (h *Handler) HandleRequestChanges(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := requireRequestID(w, r)
	if reqID == "" {
		return
	}

	id := extractID(r, "/api/admin/reviews/", "/request-changes")
	if id == "" {
		writeError(w, http.StatusBadRequest, "submission ID is required", "validation_failed")
		return
	}

	var req RequestChangesRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error(), "validation_failed")
		return
	}

	if strings.TrimSpace(req.Reason) == "" {
		writeError(w, http.StatusBadRequest, "reason is required", "validation_failed")
		return
	}

	actor := actorFromContext(ctx)
	updatedSub, err := h.service.RequestChanges(ctx, actor, publishing.ReviewDecision{
		SubmissionID: id,
		Reason:       req.Reason,
		RequestID:    reqID,
	})
	if err != nil {
		status, msg, code := sanitizeServiceError(err)
		writeError(w, status, msg, code)
		return
	}

	writeJSON(w, http.StatusOK, DecisionResponse{
		Success:    true,
		Submission: updatedSub,
	})
}

// HandleReject allows the owner to reject a submission.
func (h *Handler) HandleReject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := requireRequestID(w, r)
	if reqID == "" {
		return
	}

	id := extractID(r, "/api/admin/reviews/", "/reject")
	if id == "" {
		writeError(w, http.StatusBadRequest, "submission ID is required", "validation_failed")
		return
	}

	var req RejectRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error(), "validation_failed")
		return
	}

	if strings.TrimSpace(req.Reason) == "" {
		writeError(w, http.StatusBadRequest, "reason is required", "validation_failed")
		return
	}

	actor := actorFromContext(ctx)
	updatedSub, err := h.service.Reject(ctx, actor, publishing.ReviewDecision{
		SubmissionID: id,
		Reason:       req.Reason,
		RequestID:    reqID,
	})
	if err != nil {
		status, msg, code := sanitizeServiceError(err)
		writeError(w, status, msg, code)
		return
	}

	writeJSON(w, http.StatusOK, DecisionResponse{
		Success:    true,
		Submission: updatedSub,
	})
}

// HandleApproveAndPublish approves and publishes a submission atomically.
func (h *Handler) HandleApproveAndPublish(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := requireRequestID(w, r)
	if reqID == "" {
		return
	}

	id := extractID(r, "/api/admin/reviews/", "/approve-and-publish")
	if id == "" {
		writeError(w, http.StatusBadRequest, "submission ID is required", "validation_failed")
		return
	}

	var req ApproveAndPublishRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error(), "validation_failed")
		return
	}

	// Allow idempotency key from header if omitted from body
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		req.IdempotencyKey = r.Header.Get("Idempotency-Key")
	}

	if strings.TrimSpace(req.ArtifactSHA256) == "" {
		writeError(w, http.StatusBadRequest, "artifact_sha256 is required", "validation_failed")
		return
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		writeError(w, http.StatusBadRequest, "idempotency_key is required", "validation_failed")
		return
	}

	actor := actorFromContext(ctx)
	pubResult, err := h.service.ApproveAndPublish(ctx, actor, publishing.ApproveInput{
		SubmissionID:   id,
		ArtifactSHA256: req.ArtifactSHA256,
		IdempotencyKey: req.IdempotencyKey,
		RequestID:      reqID,
	})
	if err != nil {
		status, msg, code := sanitizeServiceError(err)
		writeError(w, status, msg, code)
		return
	}

	writeJSON(w, http.StatusOK, PublicationResponse{
		Success:         true,
		SubmissionID:    pubResult.SubmissionID,
		Revision:        pubResult.Revision,
		ArtifactSHA256:  pubResult.ArtifactSHA256,
		LabURL:          pubResult.LabURL,
		PortfolioURL:    pubResult.PortfolioURL,
		DeploymentIDs:   pubResult.DeploymentIDs,
		DestinationURLs: pubResult.DestinationURLs,
		PublishedAt:     pubResult.PublishedAt,
	})
}

// HandleArchiveProject archives a project/submission.
func (h *Handler) HandleArchiveProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := requireRequestID(w, r)
	if reqID == "" {
		return
	}

	id := extractID(r, "/api/admin/projects/", "/archive")
	if id == "" {
		writeError(w, http.StatusBadRequest, "target ID is required", "validation_failed")
		return
	}

	// Consume optional JSON body with size limit
	var body struct{}
	if r.Body != nil {
		_ = decodeJSON(w, r, &body)
	}

	actor := actorFromContext(ctx)
	err := h.service.Archive(ctx, actor, id)
	if err != nil {
		// If direct lookup by submission ID failed, try resolving by project ID
		if errors.Is(err, publishing.ErrNotFound) || strings.Contains(err.Error(), "not found") {
			subs, listErr := h.repo.ListSubmissions(ctx, publishing.SubmissionFilter{LabProjectID: id})
			if listErr == nil && len(subs) > 0 {
				err = h.service.Archive(ctx, actor, subs[0].ID)
			}
		}
	}

	if err != nil {
		status, msg, code := sanitizeServiceError(err)
		writeError(w, status, msg, code)
		return
	}

	// Also update project status if present
	if proj, getErr := h.repo.GetLabProject(ctx, id); getErr == nil {
		proj.Status = "ARCHIVED"
		proj.UpdatedAt = time.Now().UTC()
		_ = h.repo.UpdateLabProject(ctx, proj)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "project archived",
	})
}

// HandleListAudit returns the audit event log.
func (h *Handler) HandleListAudit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := 50
	if limStr := r.URL.Query().Get("limit"); limStr != "" {
		if parsed, err := strconv.Atoi(limStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	filter := publishing.AuditFilter{
		Limit:        limit,
		SubmissionID: r.URL.Query().Get("submission_id"),
		ProjectID:    r.URL.Query().Get("project_id"),
	}

	events, err := h.repo.ListAuditEvents(ctx, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query audit log", "internal_error")
		return
	}

	summaries := make([]AuditSummary, 0, len(events))
	for _, evt := range events {
		summaries = append(summaries, cleanAuditSummary(evt))
	}

	writeJSON(w, http.StatusOK, summaries)
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func requireRequestID(w http.ResponseWriter, r *http.Request) string {
	reqID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
	if reqID == "" {
		writeError(w, http.StatusBadRequest, "X-Request-ID header is required", "validation_failed")
		return ""
	}
	return reqID
}

func actorFromContext(ctx context.Context) publishing.Actor {
	if sess, ok := adminauth.SessionFromContext(ctx); ok && sess != nil {
		return publishing.Actor{
			Kind:     publishing.ActorOwner,
			Identity: sess.Login,
		}
	}
	return publishing.Actor{
		Kind:     publishing.ActorOwner,
		Identity: "gio0z",
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	const maxBytes = 1024 * 1024 // 1MB limit
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	// Verify no extraneous content after JSON object
	if dec.More() {
		return errors.New("extraneous content in request body")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: cleanErrorString(message),
		Code:  code,
	})
}

func cleanErrorString(msg string) string {
	lower := strings.ToLower(msg)
	for _, sensitive := range []string{"/home/", "/var/", "sqlite", "sql:", "bearer", "token", "password", "secret", "stack trace"} {
		if strings.Contains(lower, sensitive) {
			return "an internal processing error occurred"
		}
	}
	return msg
}

func sanitizeServiceError(err error) (int, string, string) {
	if err == nil {
		return http.StatusOK, "", ""
	}

	var se *publishing.ServiceError
	if errors.As(err, &se) {
		status := http.StatusBadRequest
		switch se.Code {
		case publishing.ErrCodeNotFound:
			status = http.StatusNotFound
		case publishing.ErrCodeForbidden:
			status = http.StatusForbidden
		case publishing.ErrCodeInvalidState, publishing.ErrCodeSlugConflict, publishing.ErrCodeArtifactMismatch, publishing.ErrCodeScanFailed:
			status = http.StatusConflict
		case publishing.ErrCodePublicationFailed:
			status = http.StatusInternalServerError
		default:
			status = http.StatusBadRequest
		}
		return status, cleanErrorString(se.Message), se.Code
	}

	if errors.Is(err, publishing.ErrNotFound) {
		return http.StatusNotFound, "resource not found", publishing.ErrCodeNotFound
	}
	if errors.Is(err, publishing.ErrConflict) || errors.Is(err, publishing.ErrDuplicateIdempotencyKey) {
		return http.StatusConflict, "resource conflict", publishing.ErrCodeSlugConflict
	}

	return http.StatusInternalServerError, "an internal error occurred", "internal_error"
}

func cleanAuditSummary(evt publishing.AuditEvent) AuditSummary {
	return AuditSummary{
		Timestamp:    evt.Timestamp,
		RequestID:    evt.RequestID,
		ActorKind:    string(evt.Actor.Kind),
		ActorID:      evt.Actor.Identity,
		Action:       evt.Action,
		ProjectID:    evt.ProjectID,
		SubmissionID: evt.SubmissionID,
		Result:       evt.Result,
	}
}

func extractID(r *http.Request, prefix, suffix string) string {
	if id := r.PathValue("id"); id != "" {
		return id
	}
	path := r.URL.Path
	if strings.HasPrefix(path, prefix) {
		path = strings.TrimPrefix(path, prefix)
	}
	if suffix != "" && strings.HasSuffix(path, suffix) {
		path = strings.TrimSuffix(path, suffix)
	}
	return strings.Trim(path, "/")
}
