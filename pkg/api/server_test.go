package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
