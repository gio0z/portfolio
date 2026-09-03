package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"portfolio/internal/publicapi"
	"portfolio/internal/publishing"
	"portfolio/pkg/api"
)

func TestHealthEndpoint(t *testing.T) {
	srv := api.NewServer()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}
}

func TestGetProfile(t *testing.T) {
	srv := api.NewServer()
	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var profile api.Profile
	if err := json.Unmarshal(rr.Body.Bytes(), &profile); err != nil {
		t.Fatalf("failed to decode profile JSON: %v", err)
	}

	if profile.Name == "" {
		t.Error("expected non-empty profile name")
	}
	if len(profile.SocialLinks) == 0 {
		t.Error("expected social links to be populated")
	}
}

func TestGetProjects(t *testing.T) {
	srv := api.NewServer()
	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var projects []api.Project
	if err := json.Unmarshal(rr.Body.Bytes(), &projects); err != nil {
		t.Fatalf("failed to decode projects JSON: %v", err)
	}

	if len(projects) == 0 {
		t.Error("expected at least 1 project in portfolio")
	}
}

func TestGetSkills(t *testing.T) {
	srv := api.NewServer()
	req := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var categories []api.SkillCategory
	if err := json.Unmarshal(rr.Body.Bytes(), &categories); err != nil {
		t.Fatalf("failed to decode skills JSON: %v", err)
	}

	if len(categories) == 0 {
		t.Error("expected skill categories to be present")
	}
}

func TestContactEndpoint(t *testing.T) {
	srv := api.NewServer()

	// Test valid contact submission
	validPayload := api.ContactRequest{
		Name:    "John Doe",
		Email:   "john@example.com",
		Subject: "Collaboration on AI Agent",
		Message: "Hi Regio, I'd like to collaborate on an autonomous systems project.",
	}
	body, _ := json.Marshal(validPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Test invalid email validation
	invalidPayload := api.ContactRequest{
		Name:    "Invalid User",
		Email:   "not-an-email",
		Subject: "Test",
		Message: "Test message",
	}
	body, _ = json.Marshal(invalidPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request for invalid email, got %d", rr.Code)
	}
}

func TestServerPublicLabEndpoints(t *testing.T) {
	// Test default server with no repository returns empty lists and 404 for slugs
	srv := api.NewServer()

	// 1. GET /api/lab/projects
	req := httptest.NewRequest(http.MethodGet, "/api/lab/projects", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/lab/projects status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Cache-Control"), "public") {
		t.Errorf("expected public Cache-Control header, got %q", rr.Header().Get("Cache-Control"))
	}

	// 2. GET /api/lab/projects/unknown-slug
	req = httptest.NewRequest(http.MethodGet, "/api/lab/projects/unknown-slug", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("GET /api/lab/projects/unknown-slug status = %d, want 404", rr.Code)
	}

	// 3. GET /api/portfolio/design-lab
	req = httptest.NewRequest(http.MethodGet, "/api/portfolio/design-lab", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/portfolio/design-lab status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Cache-Control"), "public") {
		t.Errorf("expected public Cache-Control header, got %q", rr.Header().Get("Cache-Control"))
	}
}

func TestServerWithRepositoryIntegration(t *testing.T) {
	db, err := publishing.OpenRegistry(":memory:")
	if err != nil {
		t.Fatalf("OpenRegistry: %v", err)
	}
	defer db.Close()

	repo := publishing.NewSQLiteRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	pub := publishing.LabProject{
		ID:              "lab-p1",
		Slug:            "awesome-redesign",
		Title:           "Awesome Redesign",
		OriginalProduct: "Legacy App",
		Disclaimer:      publishing.MandatoryDisclaimer,
		Status:          "PUBLISHED",
		Featured:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.CreateLabProject(ctx, pub); err != nil {
		t.Fatalf("CreateLabProject: %v", err)
	}

	draft := publishing.LabProject{
		ID:              "lab-p2",
		Slug:            "hidden-draft",
		Title:           "Draft Project",
		OriginalProduct: "Another App",
		Disclaimer:      publishing.MandatoryDisclaimer,
		Status:          "DRAFT",
		Featured:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.CreateLabProject(ctx, draft); err != nil {
		t.Fatalf("CreateLabProject: %v", err)
	}

	srv := api.NewServer(api.WithRepository(repo))

	// Verify published project listed
	req := httptest.NewRequest(http.MethodGet, "/api/lab/projects", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var list []publicapi.PublicLabProject
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != 1 || list[0].Slug != "awesome-redesign" {
		t.Fatalf("expected 1 published project, got %d", len(list))
	}

	// Verify slug lookup
	req = httptest.NewRequest(http.MethodGet, "/api/lab/projects/awesome-redesign", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	// Verify draft slug lookup returns 404
	req = httptest.NewRequest(http.MethodGet, "/api/lab/projects/hidden-draft", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}

	// Verify design-lab tab endpoint
	req = httptest.NewRequest(http.MethodGet, "/api/portfolio/design-lab", nil)
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var dlList []publicapi.PublicLabProject
	if err := json.Unmarshal(rr.Body.Bytes(), &dlList); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(dlList) != 1 || dlList[0].Slug != "awesome-redesign" {
		t.Fatalf("expected 1 featured project in design-lab, got %d", len(dlList))
	}
}
