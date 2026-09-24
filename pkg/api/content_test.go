package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"portfolio/pkg/api"
)

// TestPortfolioContentIsExported pins the contract the static build depends on.
// The frontend content exporter reads PortfolioContent, so a silently dropped
// project or skill category would ship a thinner public site while every other
// test stayed green.
func TestPortfolioContentIsExported(t *testing.T) {
	content := api.PortfolioContent()

	if content.Profile.Name == "" {
		t.Error("expected a profile name")
	}
	if len(content.Profile.Stats) == 0 {
		t.Error("expected profile stats")
	}
	if len(content.Profile.SocialLinks) == 0 {
		t.Error("expected profile social links")
	}
	if content.Profile.Bio == "" || content.Profile.Tagline == "" {
		t.Error("expected profile bio and tagline")
	}

	if len(content.Projects) != 6 {
		t.Errorf("projects = %d, want 6", len(content.Projects))
	}
	seen := make(map[string]bool, len(content.Projects))
	for _, p := range content.Projects {
		if p.ID == "" {
			t.Error("project with empty ID")
			continue
		}
		if seen[p.ID] {
			t.Errorf("duplicate project ID %q", p.ID)
		}
		seen[p.ID] = true
		if p.Title == "" || p.Tagline == "" || p.Description == "" {
			t.Errorf("project %q is missing display text", p.ID)
		}
		if p.Category == "" || p.Metrics == "" {
			t.Errorf("project %q is missing category or metrics", p.ID)
		}
		if len(p.Tags) == 0 {
			t.Errorf("project %q has no tags", p.ID)
		}
	}

	if len(content.SkillCategories) != 4 {
		t.Errorf("skill categories = %d, want 4", len(content.SkillCategories))
	}
	seenCategories := make(map[string]bool, len(content.SkillCategories))
	for _, c := range content.SkillCategories {
		if c.Category == "" || c.Summary == "" {
			t.Errorf("skill category %q is missing a category name or summary", c.Category)
		}
		if seenCategories[c.Category] {
			t.Errorf("duplicate skill category %q", c.Category)
		}
		seenCategories[c.Category] = true
		if len(c.Skills) == 0 {
			t.Errorf("skill category %q has no skills", c.Category)
		}
	}
}

// TestPublicEndpointsServeExportedContent guards against a second copy of the
// content reappearing in a handler. The exported content and the live API must
// stay the same data, or the static build and the runtime drift apart.
func TestPublicEndpointsServeExportedContent(t *testing.T) {
	content := api.PortfolioContent()
	srv := api.NewServer()

	var profile api.Profile
	getJSON(t, srv, "/api/profile", &profile)
	if profile.Name != content.Profile.Name {
		t.Errorf("profile name = %q, want %q", profile.Name, content.Profile.Name)
	}
	if len(profile.Stats) != len(content.Profile.Stats) {
		t.Errorf("profile stats = %d, want %d", len(profile.Stats), len(content.Profile.Stats))
	}

	var projects []api.Project
	getJSON(t, srv, "/api/projects", &projects)
	if len(projects) != len(content.Projects) {
		t.Fatalf("projects = %d, want %d", len(projects), len(content.Projects))
	}
	for i, p := range projects {
		if p.ID != content.Projects[i].ID {
			t.Errorf("project[%d] ID = %q, want %q", i, p.ID, content.Projects[i].ID)
		}
	}

	var skills []api.SkillCategory
	getJSON(t, srv, "/api/skills", &skills)
	if len(skills) != len(content.SkillCategories) {
		t.Fatalf("skill categories = %d, want %d", len(skills), len(content.SkillCategories))
	}
	for i, c := range skills {
		if c.Category != content.SkillCategories[i].Category {
			t.Errorf("skill category[%d] = %q, want %q", i, c.Category, content.SkillCategories[i].Category)
		}
	}

	// Category filtering must keep working off the exported slice.
	var filtered []api.Project
	getJSON(t, srv, "/api/projects?category=Government", &filtered)
	if len(filtered) == 0 {
		t.Error("expected at least one Government project")
	}
	for _, p := range filtered {
		if p.Category != "Government" {
			t.Errorf("filtered project %q has category %q", p.ID, p.Category)
		}
	}
}

func getJSON(t *testing.T, srv *api.Server, route string, out interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, route, nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", route, rr.Code)
	}
	if err := json.Unmarshal(rr.Body.Bytes(), out); err != nil {
		t.Fatalf("GET %s: failed to decode JSON: %v", route, err)
	}
}
