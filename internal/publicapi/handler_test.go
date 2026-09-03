package publicapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"portfolio/internal/publicapi"
	"portfolio/internal/publishing"
)

func TestPublicAPI_OnlyPublishedRecordsAppear(t *testing.T) {
	ctx := context.Background()
	db, err := publishing.OpenRegistry(":memory:")
	if err != nil {
		t.Fatalf("OpenRegistry: %v", err)
	}
	defer db.Close()
	repo := publishing.NewSQLiteRepository(db)

	now := time.Now().UTC()

	// Seed records in various states
	projects := []publishing.LabProject{
		{
			ID:              "lab-1",
			Slug:            "published-proj-1",
			Title:           "Published Project One",
			OriginalProduct: "Product One",
			Disclaimer:      publishing.MandatoryDisclaimer,
			Focus:           []string{"design", "ux"},
			Platforms:       []string{"web"},
			Status:          "PUBLISHED",
			Featured:        false,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "lab-2",
			Slug:            "published-proj-2",
			Title:           "Published Project Two",
			OriginalProduct: "Product Two",
			Disclaimer:      publishing.MandatoryDisclaimer,
			Focus:           []string{"mobile"},
			Platforms:       []string{"ios", "android"},
			Status:          "published", // case insensitivity check
			Featured:        true,
			CreatedAt:       now.Add(time.Minute),
			UpdatedAt:       now.Add(time.Minute),
		},
		{
			ID:              "lab-3",
			Slug:            "draft-proj",
			Title:           "Draft Project",
			OriginalProduct: "Product Three",
			Disclaimer:      publishing.MandatoryDisclaimer,
			Status:          "DRAFT",
			Featured:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "lab-4",
			Slug:            "in-review-proj",
			Title:           "In Review Project",
			OriginalProduct: "Product Four",
			Disclaimer:      publishing.MandatoryDisclaimer,
			Status:          "IN_REVIEW",
			Featured:        false,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "lab-5",
			Slug:            "archived-proj",
			Title:           "Archived Project",
			OriginalProduct: "Product Five",
			Disclaimer:      publishing.MandatoryDisclaimer,
			Status:          "ARCHIVED",
			Featured:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "lab-6",
			Slug:            "build-failed-proj",
			Title:           "Build Failed Project",
			OriginalProduct: "Product Six",
			Disclaimer:      publishing.MandatoryDisclaimer,
			Status:          "BUILD_FAILED",
			Featured:        false,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}

	for _, p := range projects {
		if err := repo.CreateLabProject(ctx, p); err != nil {
			t.Fatalf("CreateLabProject %s: %v", p.ID, err)
		}
	}

	handler := publicapi.NewHandlerWithRepo(repo)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Test GET /api/lab/projects
	req := httptest.NewRequest(http.MethodGet, "/api/lab/projects", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/lab/projects status = %d, want %d", rec.Code, http.StatusOK)
	}

	var labProjects []publicapi.PublicLabProject
	if err := json.Unmarshal(rec.Body.Bytes(), &labProjects); err != nil {
		t.Fatalf("decode /api/lab/projects: %v", err)
	}

	if len(labProjects) != 2 {
		t.Fatalf("got %d projects, want 2 published projects", len(labProjects))
	}

	for _, p := range labProjects {
		if !strings.EqualFold(p.Status, "PUBLISHED") {
			t.Errorf("expected project %s to have published status, got %s", p.Slug, p.Status)
		}
		if p.Slug == "draft-proj" || p.Slug == "in-review-proj" || p.Slug == "archived-proj" || p.Slug == "build-failed-proj" {
			t.Errorf("non-published project %s appeared in public response", p.Slug)
		}
	}
}

func TestPublicAPI_MandatoryDisclaimerEnforced(t *testing.T) {
	ctx := context.Background()
	db, err := publishing.OpenRegistry(":memory:")
	if err != nil {
		t.Fatalf("OpenRegistry: %v", err)
	}
	defer db.Close()
	repo := publishing.NewSQLiteRepository(db)

	now := time.Now().UTC()

	// Seed published project with EMPTY disclaimer in DB
	p := publishing.LabProject{
		ID:              "lab-no-disclaimer",
		Slug:            "no-disclaimer-proj",
		Title:           "Project Missing Disclaimer",
		OriginalProduct: "Some App",
		Disclaimer:      "", // empty in database
		Status:          "PUBLISHED",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.CreateLabProject(ctx, p); err != nil {
		t.Fatalf("CreateLabProject: %v", err)
	}

	handler := publicapi.NewHandlerWithRepo(repo)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Check GET /api/lab/projects
	req := httptest.NewRequest(http.MethodGet, "/api/lab/projects", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var labProjects []publicapi.PublicLabProject
	if err := json.Unmarshal(rec.Body.Bytes(), &labProjects); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(labProjects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(labProjects))
	}
	if !strings.Contains(labProjects[0].Disclaimer, publicapi.MandatoryDisclaimer) {
		t.Errorf("disclaimer %q does not contain mandatory disclaimer %q",
			labProjects[0].Disclaimer, publicapi.MandatoryDisclaimer)
	}

	// Check GET /api/lab/projects/{slug}
	reqSlug := httptest.NewRequest(http.MethodGet, "/api/lab/projects/no-disclaimer-proj", nil)
	recSlug := httptest.NewRecorder()
	mux.ServeHTTP(recSlug, reqSlug)

	var singleProject publicapi.PublicLabProject
	if err := json.Unmarshal(recSlug.Body.Bytes(), &singleProject); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.Contains(singleProject.Disclaimer, publicapi.MandatoryDisclaimer) {
		t.Errorf("disclaimer %q does not contain mandatory disclaimer %q",
			singleProject.Disclaimer, publicapi.MandatoryDisclaimer)
	}
}

func TestPublicAPI_PrivateSourceMetadataOmitted(t *testing.T) {
	ctx := context.Background()
	db, err := publishing.OpenRegistry(":memory:")
	if err != nil {
		t.Fatalf("OpenRegistry: %v", err)
	}
	defer db.Close()
	repo := publishing.NewSQLiteRepository(db)

	now := time.Now().UTC()

	p := publishing.LabProject{
		ID:              "lab-private",
		Slug:            "private-source-test",
		Title:           "Private Source Test",
		OriginalProduct: "Redesign Target",
		Disclaimer:      publishing.MandatoryDisclaimer,
		Status:          "PUBLISHED",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.CreateLabProject(ctx, p); err != nil {
		t.Fatalf("CreateLabProject: %v", err)
	}

	// Create submission with private source metadata, build results, agent identities
	sub := publishing.Submission{
		ID:                 "sub-private-1",
		LabProjectID:       p.ID,
		Revision:           1,
		State:              publishing.Published,
		ArtifactSHA256:     "d41d8cd98f00b204e9800998ecf8427e",
		PreviewURL:         "https://internal-preview.private.lan/v1/sub-private-1",
		BuildResult:        "secret-build-log-internal",
		TestResult:         "secret-test-log-internal",
		SecurityScanResult: "secret-scan-report-internal",
		SubmittedBy:        "agent:secret-internal-actor-999",
		PortfolioMetadata: publishing.PortfolioMetadata{
			"summary":         "Safe public summary",
			"submitted_by":    "agent:secret-internal-actor-999",
			"artifact_sha256": "d41d8cd98f00b204e9800998ecf8427e",
			"preview_url":     "https://internal-preview.private.lan/v1/sub-private-1",
			"source_repo":     "git@github.com:gio0z/super-secret-repo.git",
			"review_reason":   "private owner reason text",
			"internal_token":  "internal-secret-token-xyz",
			"server_path":     "/var/portfolio/internal/data",
		},
		SubmittedAt: now,
		UpdatedAt:   now,
	}
	if err := repo.CreateSubmission(ctx, sub); err != nil {
		t.Fatalf("CreateSubmission: %v", err)
	}

	handler := publicapi.NewHandlerWithRepo(repo)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	endpoints := []string{
		"/api/lab/projects",
		"/api/lab/projects/private-source-test",
		"/api/portfolio/design-lab",
	}

	forbiddenStrings := []string{
		"agent:secret-internal-actor-999",
		"d41d8cd98f00b204e9800998ecf8427e",
		"internal-preview.private.lan",
		"secret-build-log-internal",
		"secret-test-log-internal",
		"secret-scan-report-internal",
		"super-secret-repo.git",
		"private owner reason text",
		"internal-secret-token-xyz",
		"/var/portfolio/internal/data",
	}

	for _, endpoint := range endpoints {
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		body := rec.Body.String()
		for _, forbidden := range forbiddenStrings {
			if strings.Contains(body, forbidden) {
				t.Errorf("endpoint %s leaked private source metadata %q in body: %s",
					endpoint, forbidden, body)
			}
		}
	}
}

func TestPublicAPI_MissingSlugsReturn404(t *testing.T) {
	ctx := context.Background()
	db, err := publishing.OpenRegistry(":memory:")
	if err != nil {
		t.Fatalf("OpenRegistry: %v", err)
	}
	defer db.Close()
	repo := publishing.NewSQLiteRepository(db)

	now := time.Now().UTC()

	// Seed a draft project
	draftProject := publishing.LabProject{
		ID:              "lab-draft",
		Slug:            "hidden-draft",
		Title:           "Hidden Draft",
		OriginalProduct: "Draft App",
		Disclaimer:      publishing.MandatoryDisclaimer,
		Status:          "DRAFT",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.CreateLabProject(ctx, draftProject); err != nil {
		t.Fatalf("CreateLabProject: %v", err)
	}

	handler := publicapi.NewHandlerWithRepo(repo)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	testCases := []struct {
		name string
		url  string
	}{
		{"non-existent slug", "/api/lab/projects/completely-unknown-slug"},
		{"draft project slug", "/api/lab/projects/hidden-draft"},
		{"empty slug trailing slash", "/api/lab/projects/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("%s: expected 404, got %d. Body: %s", tc.name, rec.Code, rec.Body.String())
			}

			var errResp map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("response should be valid JSON: %v", err)
			}
			if errResp["error"] == nil || errResp["error"] == "" {
				t.Errorf("expected non-empty error field in 404 response: %v", errResp)
			}
		})
	}
}

func TestPublicAPI_CacheHeadersAndNoAdminData(t *testing.T) {
	ctx := context.Background()
	db, err := publishing.OpenRegistry(":memory:")
	if err != nil {
		t.Fatalf("OpenRegistry: %v", err)
	}
	defer db.Close()
	repo := publishing.NewSQLiteRepository(db)

	now := time.Now().UTC()
	p := publishing.LabProject{
		ID:              "lab-cache",
		Slug:            "cache-test-proj",
		Title:           "Cache Test Project",
		OriginalProduct: "App",
		Disclaimer:      publishing.MandatoryDisclaimer,
		Status:          "PUBLISHED",
		Featured:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.CreateLabProject(ctx, p); err != nil {
		t.Fatalf("CreateLabProject: %v", err)
	}

	handler := publicapi.NewHandlerWithRepo(repo)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	endpoints := []string{
		"/api/lab/projects",
		"/api/lab/projects/cache-test-proj",
		"/api/portfolio/design-lab",
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodGet, ep, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", ep, rec.Code)
		}

		cc := rec.Header().Get("Cache-Control")
		if cc == "" || !strings.Contains(cc, "public") {
			t.Errorf("%s: expected public Cache-Control header, got %q", ep, cc)
		}

		ct := rec.Header().Get("Content-Type")
		if !strings.HasPrefix(ct, "application/json") {
			t.Errorf("%s: expected application/json content type, got %q", ep, ct)
		}

		// Verify no admin headers or cookies leaked
		if setCookie := rec.Header().Get("Set-Cookie"); setCookie != "" {
			t.Errorf("%s: Set-Cookie must not be set on public endpoints, got %q", ep, setCookie)
		}
		if auth := rec.Header().Get("Authorization"); auth != "" {
			t.Errorf("%s: Authorization header leaked: %q", ep, auth)
		}
		if csrf := rec.Header().Get("X-CSRF-Token"); csrf != "" {
			t.Errorf("%s: X-CSRF-Token header leaked: %q", ep, csrf)
		}
	}
}

func TestPublicAPI_PortfolioDesignLab_OnlyFeaturedPublished(t *testing.T) {
	ctx := context.Background()
	db, err := publishing.OpenRegistry(":memory:")
	if err != nil {
		t.Fatalf("OpenRegistry: %v", err)
	}
	defer db.Close()
	repo := publishing.NewSQLiteRepository(db)

	now := time.Now().UTC()

	projects := []publishing.LabProject{
		{
			ID:              "lab-f-pub",
			Slug:            "featured-published",
			Title:           "Featured Published",
			OriginalProduct: "App F",
			Disclaimer:      publishing.MandatoryDisclaimer,
			Status:          "PUBLISHED",
			Featured:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "lab-nf-pub",
			Slug:            "not-featured-published",
			Title:           "Not Featured Published",
			OriginalProduct: "App NF",
			Disclaimer:      publishing.MandatoryDisclaimer,
			Status:          "PUBLISHED",
			Featured:        false,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "lab-f-draft",
			Slug:            "featured-draft",
			Title:           "Featured Draft",
			OriginalProduct: "App FD",
			Disclaimer:      publishing.MandatoryDisclaimer,
			Status:          "DRAFT",
			Featured:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "lab-f-archived",
			Slug:            "featured-archived",
			Title:           "Featured Archived",
			OriginalProduct: "App FA",
			Disclaimer:      publishing.MandatoryDisclaimer,
			Status:          "ARCHIVED",
			Featured:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}

	for _, p := range projects {
		if err := repo.CreateLabProject(ctx, p); err != nil {
			t.Fatalf("CreateLabProject %s: %v", p.ID, err)
		}
	}

	handler := publicapi.NewHandlerWithRepo(repo)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/portfolio/design-lab", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/portfolio/design-lab status = %d, want %d", rec.Code, http.StatusOK)
	}

	var designLabProjects []publicapi.PublicLabProject
	if err := json.Unmarshal(rec.Body.Bytes(), &designLabProjects); err != nil {
		t.Fatalf("decode /api/portfolio/design-lab: %v", err)
	}

	if len(designLabProjects) != 1 {
		t.Fatalf("got %d design lab projects, want exactly 1", len(designLabProjects))
	}

	if designLabProjects[0].Slug != "featured-published" {
		t.Errorf("got slug %q, want %q", designLabProjects[0].Slug, "featured-published")
	}
	if !designLabProjects[0].Featured {
		t.Errorf("expected project to be featured")
	}
	if !strings.Contains(designLabProjects[0].Disclaimer, publicapi.MandatoryDisclaimer) {
		t.Errorf("expected mandatory disclaimer on design lab project")
	}
}

func TestPublicAPI_MethodNotAllowed(t *testing.T) {
	handler := publicapi.NewHandler(nil)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	endpoints := []string{
		"/api/lab/projects",
		"/api/lab/projects/test-slug",
		"/api/portfolio/design-lab",
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodPost, ep, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s: expected 405 Method Not Allowed, got %d", ep, rec.Code)
		}
	}
}
