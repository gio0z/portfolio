package publicapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"portfolio/internal/publishing"
)

// MandatoryDisclaimer is the required legal disclaimer for all published lab redesign concepts.
const MandatoryDisclaimer = publishing.MandatoryDisclaimer

var (
	// ErrNotFound indicates a requested project was not found or is not published.
	ErrNotFound = errors.New("publicapi: project not found")
)

// PublicLabProject represents the sanitized public view of a published Design Lab project.
// All private source metadata, author identities, preview URLs, build artifacts,
// and internal audit notes are strictly omitted.
type PublicLabProject struct {
	ID              string    `json:"id"`
	Slug            string    `json:"slug"`
	Title           string    `json:"title"`
	OriginalProduct string    `json:"original_product"`
	Disclaimer      string    `json:"disclaimer"`
	Focus           []string  `json:"focus,omitempty"`
	Platforms       []string  `json:"platforms,omitempty"`
	Status          string    `json:"status"`
	Featured        bool      `json:"featured"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CoverImage      string    `json:"cover_image,omitempty"`
	Summary         string    `json:"summary,omitempty"`
	CaseStudyURL    string    `json:"case_study_url,omitempty"`
	LiveDemoURL     string    `json:"live_demo_url,omitempty"`
}

// ProjectReader queries published lab projects.
type ProjectReader interface {
	ListPublishedProjects(ctx context.Context) ([]PublicLabProject, error)
	GetPublishedProjectBySlug(ctx context.Context, slug string) (PublicLabProject, error)
	ListFeaturedProjects(ctx context.Context) ([]PublicLabProject, error)
}

// RepositoryReader implements ProjectReader using a publishing.Repository.
type RepositoryReader struct {
	repo publishing.Repository
}

// NewRepositoryReader creates a reader backed by a publishing repository.
func NewRepositoryReader(repo publishing.Repository) *RepositoryReader {
	return &RepositoryReader{repo: repo}
}

// ListPublishedProjects returns all projects with status PUBLISHED.
func (r *RepositoryReader) ListPublishedProjects(ctx context.Context) ([]PublicLabProject, error) {
	if r.repo == nil {
		return []PublicLabProject{}, nil
	}

	all, err := r.repo.ListLabProjects(ctx, publishing.ProjectFilter{})
	if err != nil {
		return nil, err
	}

	result := make([]PublicLabProject, 0)
	for _, p := range all {
		if strings.EqualFold(p.Status, "PUBLISHED") {
			result = append(result, cleanPublicProject(p))
		}
	}
	return result, nil
}

// GetPublishedProjectBySlug returns a single published project by slug, or ErrNotFound.
func (r *RepositoryReader) GetPublishedProjectBySlug(ctx context.Context, slug string) (PublicLabProject, error) {
	cleanSlug := strings.TrimSpace(slug)
	if cleanSlug == "" {
		return PublicLabProject{}, ErrNotFound
	}

	published, err := r.ListPublishedProjects(ctx)
	if err != nil {
		return PublicLabProject{}, err
	}

	for _, p := range published {
		if p.Slug == cleanSlug {
			return p, nil
		}
	}
	return PublicLabProject{}, ErrNotFound
}

// ListFeaturedProjects returns all published projects that are flagged as featured.
func (r *RepositoryReader) ListFeaturedProjects(ctx context.Context) ([]PublicLabProject, error) {
	published, err := r.ListPublishedProjects(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]PublicLabProject, 0)
	for _, p := range published {
		if p.Featured {
			result = append(result, p)
		}
	}
	return result, nil
}

// cleanPublicProject transforms a raw LabProject into a sanitized PublicLabProject,
// enforcing the mandatory disclaimer and removing any private source metadata.
func cleanPublicProject(p publishing.LabProject) PublicLabProject {
	disclaimer := strings.TrimSpace(p.Disclaimer)
	if !strings.Contains(disclaimer, MandatoryDisclaimer) {
		if disclaimer == "" {
			disclaimer = MandatoryDisclaimer
		} else {
			disclaimer = MandatoryDisclaimer + " " + disclaimer
		}
	}

	return PublicLabProject{
		ID:              p.ID,
		Slug:            p.Slug,
		Title:           p.Title,
		OriginalProduct: p.OriginalProduct,
		Disclaimer:      disclaimer,
		Focus:           p.Focus,
		Platforms:       p.Platforms,
		Status:          "PUBLISHED",
		Featured:        p.Featured,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

// Handler handles public HTTP requests for Lab projects and Design Lab endpoints.
type Handler struct {
	reader ProjectReader
}

// NewHandler creates a new public API handler with the specified reader.
func NewHandler(reader ProjectReader) *Handler {
	return &Handler{reader: reader}
}

// NewHandlerWithRepo creates a new public API handler backed by a publishing repository.
func NewHandlerWithRepo(repo publishing.Repository) *Handler {
	return &Handler{reader: NewRepositoryReader(repo)}
}

// RegisterRoutes registers the public routes on the provided ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/lab/projects", h.HandleListProjects)
	mux.HandleFunc("/api/lab/projects/", h.HandleProjectSubpath)
	mux.HandleFunc("/api/portfolio/design-lab", h.HandlePortfolioDesignLab)
}

func (h *Handler) HandleListProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	if h.reader == nil {
		writeJSON(w, http.StatusOK, []PublicLabProject{})
		return
	}

	projects, err := h.reader.ListPublishedProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list lab projects")
		return
	}

	writeJSON(w, http.StatusOK, projects)
}

func (h *Handler) HandleProjectSubpath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	slug := strings.TrimPrefix(r.URL.Path, "/api/lab/projects/")
	slug = strings.TrimSpace(slug)
	if slug == "" || strings.Contains(slug, "/") {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	if h.reader == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	project, err := h.reader.GetPublishedProjectBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, publishing.ErrNotFound) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get lab project")
		return
	}

	writeJSON(w, http.StatusOK, project)
}

func (h *Handler) HandlePortfolioDesignLab(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	if h.reader == nil {
		writeJSON(w, http.StatusOK, []PublicLabProject{})
		return
	}

	featured, err := h.reader.ListFeaturedProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list design lab projects")
		return
	}

	writeJSON(w, http.StatusOK, featured)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func methodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Allow", http.MethodGet)
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "Method not allowed",
	})
}
