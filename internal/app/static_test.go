package app_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"portfolio/internal/adminauth"
	"portfolio/internal/app"
)

// newStaticSiteApp composes an application over a temporary dist directory that
// mimics a prerendered multi-page build: real files at the root, one directory
// per route with its own index.html, and a client-only admin shell that must
// answer every /admin/* deep link.
func newStaticSiteApp(t *testing.T, dist string) *app.App {
	t.Helper()

	writeFile(t, filepath.Join(dist, "index.html"), "<html><body>HOME</body></html>")
	writeFile(t, filepath.Join(dist, "about", "index.html"), "<html><body>ABOUT PAGE</body></html>")
	writeFile(t, filepath.Join(dist, "work", "index.html"), "<html><body>WORK INDEX</body></html>")
	writeFile(t, filepath.Join(dist, "work", "jam-nguar", "index.html"), "<html><body>CASE STUDY</body></html>")
	writeFile(t, filepath.Join(dist, "admin", "index.html"), "<html><body>ADMIN SHELL</body></html>")
	writeFile(t, filepath.Join(dist, "sitemap.xml"), "<urlset></urlset>")

	data := t.TempDir()
	a, err := app.New(app.Config{
		Env:               "test",
		FrontendDist:      dist,
		PreviewOrigin:     "https://preview.example.com",
		LabOrigin:         "https://lab.example.com",
		PortfolioOrigin:   "https://portfolio.example.com",
		RegistryPath:      filepath.Join(data, "registry.sqlite"),
		ArtifactStoreRoot: filepath.Join(data, "artifacts"),
		MCPTokenSecret:    strings.Repeat("m", 32),
		ApprovalVerifier:  adminauth.NewDevApprovalVerifier("test"),
	})
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })
	return a
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func get(t *testing.T, a *app.App, route string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, route, nil)
	rr := httptest.NewRecorder()
	a.ServeHTTP(rr, req)
	body, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatalf("read body for %s: %v", route, err)
	}
	return rr.Code, string(body)
}

// TestServeDirectoryIndexFiles pins the contract that makes a prerendered
// multi-page build work: /about must serve about/index.html, not the root
// fallback. Serving the homepage for every route would give every page the same
// document and defeat the point of prerendering.
func TestServeDirectoryIndexFiles(t *testing.T) {
	dist := t.TempDir()
	a := newStaticSiteApp(t, dist)

	cases := []struct {
		route string
		want  string
	}{
		{"/", "HOME"},
		{"/about", "ABOUT PAGE"},
		{"/about/", "ABOUT PAGE"},
		{"/work", "WORK INDEX"},
		{"/work/jam-nguar", "CASE STUDY"},
		{"/sitemap.xml", "<urlset>"},
	}
	for _, tc := range cases {
		status, body := get(t, a, tc.route)
		if status != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", tc.route, status)
			continue
		}
		if !contains(body, tc.want) {
			t.Errorf("GET %s body = %q, want it to contain %q", tc.route, body, tc.want)
		}
	}
}

// TestAdminDeepLinkFallsBackToAdminShell covers the client-only admin island.
// The admin app is a single-page application: React Router owns /admin/reviews
// and every sibling, so no file exists for those paths at build time. Without a
// fallback inside /admin, refreshing a deep link returns the public homepage.
func TestAdminDeepLinkFallsBackToAdminShell(t *testing.T) {
	dist := t.TempDir()
	a := newStaticSiteApp(t, dist)

	for _, route := range []string{
		"/admin",
		"/admin/",
		"/admin/reviews",
		"/admin/reviews/abc-123",
		"/admin/queue?status=open",
	} {
		status, body := get(t, a, route)
		if status != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", route, status)
			continue
		}
		if !contains(body, "ADMIN SHELL") {
			t.Errorf("GET %s body = %q, want the admin shell", route, body)
		}
	}
}

// TestAdminIndexCanonicalisesToDirectory pins that the explicit shell filename
// redirects to its directory form, so the admin area has one canonical URL
// instead of two that serve the same document.
func TestAdminIndexCanonicalisesToDirectory(t *testing.T) {
	dist := t.TempDir()
	a := newStaticSiteApp(t, dist)

	req := httptest.NewRequest(http.MethodGet, "/admin/index.html", nil)
	rr := httptest.NewRecorder()
	a.ServeHTTP(rr, req)

	if rr.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want 301", rr.Code)
	}
	// http.ServeFile issues a relative redirect; the browser resolves it to
	// /admin/ against the request path.
	if loc := rr.Header().Get("Location"); loc != "./" {
		t.Errorf("Location = %q, want ./", loc)
	}
}

// TestAdminFallbackDoesNotShadowRealFiles guards the fallback against
// swallowing static admin assets, which would break the shell it serves.
func TestAdminFallbackDoesNotShadowRealFiles(t *testing.T) {
	dist := t.TempDir()
	a := newStaticSiteApp(t, dist)
	writeFile(t, filepath.Join(dist, "admin", "assets", "app.js"), "console.log('admin')")

	status, body := get(t, a, "/admin/assets/app.js")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if !contains(body, "console.log") {
		t.Errorf("body = %q, want the real asset", body)
	}
}

// TestApiRoutesSurviveTheFallback checks that the static fallback logic never
// intercepts the API surface, including the admin auth routes it sits beside.
func TestApiRoutesSurviveTheFallback(t *testing.T) {
	dist := t.TempDir()
	a := newStaticSiteApp(t, dist)

	for _, route := range []string{
		"/api/health",
		"/api/profile",
		"/api/projects",
		"/api/skills",
		"/api/admin/auth/session",
	} {
		status, body := get(t, a, route)
		if status == http.StatusOK && contains(body, "HOME") {
			t.Errorf("GET %s was served the static fallback", route)
		}
	}
}

// TestUnknownPublicPathFallsBackToHome preserves the original SPA behaviour for
// public routes that have no prerendered document.
func TestUnknownPublicPathFallsBackToHome(t *testing.T) {
	dist := t.TempDir()
	a := newStaticSiteApp(t, dist)

	status, body := get(t, a, "/no-such-page")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if !contains(body, "HOME") {
		t.Errorf("body = %q, want the root index", body)
	}
}

// TestMissingAdminShellDegradesGracefully covers a dist built without the admin
// island: the request must fall through to the public surface rather than 404
// or panic.
func TestMissingAdminShellDegradesGracefully(t *testing.T) {
	dist := t.TempDir()
	a := newStaticSiteApp(t, dist)
	if err := os.RemoveAll(filepath.Join(dist, "admin")); err != nil {
		t.Fatalf("remove admin: %v", err)
	}

	status, body := get(t, a, "/admin/reviews")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if !contains(body, "HOME") {
		t.Errorf("body = %q, want the public fallback", body)
	}
}

// TestLabDeepLinkFallsBackToLabShell covers the client-routed Design Lab. Its
// case studies are reached at /lab/<slug> and resolved client-side, so the
// server must deliver the lab document for those paths.
func TestLabDeepLinkFallsBackToLabShell(t *testing.T) {
	dist := t.TempDir()
	a := newStaticSiteApp(t, dist)
	writeFile(t, filepath.Join(dist, "lab", "index.html"), "<html><body>LAB SHELL</body></html>")

	for _, route := range []string{"/lab", "/lab/", "/lab/checkout-redesign"} {
		status, body := get(t, a, route)
		if status != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", route, status)
			continue
		}
		if !contains(body, "LAB SHELL") {
			t.Errorf("GET %s body = %q, want the lab shell", route, body)
		}
	}
}

// TestShellPrefixIsExact guards the prefix match: a neighbouring route that
// merely starts with the same letters is a public page, not a shell deep link.
func TestShellPrefixIsExact(t *testing.T) {
	dist := t.TempDir()
	a := newStaticSiteApp(t, dist)

	for _, route := range []string{"/administrator", "/laboratory"} {
		status, body := get(t, a, route)
		if status != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", route, status)
			continue
		}
		if !contains(body, "HOME") {
			t.Errorf("GET %s body = %q, want the public fallback", route, body)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}
