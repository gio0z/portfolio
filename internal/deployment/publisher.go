package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"portfolio/internal/publishing"
)

// LabStaged is a staged-but-invisible Lab release. It becomes public only
// when MakeVisible swaps the slug pointer to the immutable version path.
type LabStaged struct {
	ArtifactHash string
	Slug         string
	VersionPath  string
}

// PortfolioStaged is a staged-but-invisible portfolio catalog entry.
type PortfolioStaged struct {
	ArtifactHash string
	Slug         string
	VersionPath  string
}

// PublishedDestination is one destination after its pointer swap.
type PublishedDestination struct {
	URL          string
	DeploymentID string
}

// LabPublisher stages approved Lab artifacts under immutable version paths
// and exposes them below the configured LAB_PUBLIC_ORIGIN via a final
// pointer swap.
type LabPublisher interface {
	Stage(context.Context, publishing.PublicationRequest) (LabStaged, error)
	MakeVisible(context.Context, LabStaged) (PublishedDestination, error)
	Rollback(context.Context, LabStaged) error
}

// PortfolioCatalogPublisher stages approved portfolio Design Lab catalog
// entries and exposes them below PORTFOLIO_PUBLIC_ORIGIN via a final
// pointer swap.
type PortfolioCatalogPublisher interface {
	Stage(context.Context, publishing.PublicationRequest) (PortfolioStaged, error)
	MakeVisible(context.Context, PortfolioStaged) (PublishedDestination, error)
	Rollback(context.Context, PortfolioStaged) error
}

// AtomicPublisher deploys an approved artifact to Lab and portfolio
// destinations atomically. It matches publishing.AtomicPublisher so Task 4's
// service consumes it without adaptation:
//
//	Publish(context.Context, publishing.PublicationRequest) (publishing.PublicationOutput, error)
//
// Ruling: deployment republishes the publishing contract verbatim instead of
// aliasing publishing.AtomicPublisher — keeps the interface discoverable next
// to its implementation and avoids an import-direction trap for Task 16
// consumers — cost if wrong: one duplicated method signature to keep in sync.
type AtomicPublisher interface {
	Publish(context.Context, publishing.PublicationRequest) (publishing.PublicationOutput, error)
}

// Compile-time check: *atomicPublisher satisfies the publishing service's
// publisher contract.
var _ publishing.AtomicPublisher = (*atomicPublisher)(nil)

// labPublisher is the filesystem LabPublisher.
type labPublisher struct {
	root   string
	origin string
}

// NewLabPublisher returns a LabPublisher rooted at root exposing slugs below
// origin (LAB_PUBLIC_ORIGIN).
func NewLabPublisher(root, origin string) LabPublisher {
	return &labPublisher{root: root, origin: strings.TrimSuffix(strings.TrimSpace(origin), "/")}
}

// labVersionsDir is the immutable version root for Lab releases.
func (p *labPublisher) labVersionsDir() string {
	return filepath.Join(p.root, "var", "portfolio", "lab", "versions")
}

// pointerPath is the per-slug pointer file naming the visible version.
func (p *labPublisher) pointerPath(slug string) string {
	return filepath.Join(p.root, "var", "portfolio", "lab", "pointers", slug+".json")
}

// Stage writes the immutable version directory for hash. Repeat stages for
// the same hash are no-ops (idempotent): if the pointer already names this
// version, or the version directory already exists, Stage returns success.
func (p *labPublisher) Stage(_ context.Context, req publishing.PublicationRequest) (LabStaged, error) {
	hash, err := normalizeHash(req.Submission.ArtifactSHA256)
	if err != nil {
		return LabStaged{}, err
	}
	slug, err := normalizeSlug(req.Project.Slug)
	if err != nil {
		return LabStaged{}, err
	}
	if strings.TrimSpace(p.root) == "" {
		return LabStaged{}, errors.New("deployment: lab artifact store root is not configured")
	}

	versionDir := filepath.Join(p.labVersionsDir(), hash)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		return LabStaged{}, fmt.Errorf("deployment: create lab version dir: %w", err)
	}

	// Immutable version manifest: created once, never overwritten. The
	// sentinel + O_EXCL keeps concurrent stages from racing each other.
	staged := LabStaged{ArtifactHash: hash, Slug: slug, VersionPath: versionDir}
	payload, err := json.Marshal(map[string]string{
		"slug":            slug,
		"artifact_sha256": hash,
	})
	if err != nil {
		return LabStaged{}, fmt.Errorf("deployment: marshal lab manifest: %w", err)
	}
	sentinel := filepath.Join(versionDir, "release.json")
	f, err := os.OpenFile(sentinel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return staged, nil
		}
		return LabStaged{}, fmt.Errorf("deployment: stage lab version: %w", err)
	}
	if _, err := f.Write(payload); err != nil {
		_ = f.Close()
		return LabStaged{}, fmt.Errorf("deployment: stage lab version: %w", err)
	}
	if err := f.Close(); err != nil {
		return LabStaged{}, fmt.Errorf("deployment: stage lab version: %w", err)
	}
	return staged, nil
}

// pointerFor returns the version currently visible for slug, or "".
func (p *labPublisher) pointerFor(slug string) string {
	raw, err := os.ReadFile(p.pointerPath(slug))
	if err != nil {
		return ""
	}
	var ptr struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &ptr); err != nil {
		return ""
	}
	return ptr.Version
}

// MakeVisible atomically points slug at staged.VersionPath. Repointing to the
// already-visible version is a no-op for idempotent retries.
func (p *labPublisher) MakeVisible(_ context.Context, staged LabStaged) (PublishedDestination, error) {
	if strings.TrimSpace(p.origin) == "" {
		return PublishedDestination{}, errors.New("deployment: LAB_PUBLIC_ORIGIN is not configured")
	}
	if _, err := normalizeHash(staged.ArtifactHash); err != nil {
		return PublishedDestination{}, err
	}
	slug, err := normalizeSlug(staged.Slug)
	if err != nil {
		return PublishedDestination{}, err
	}
	if _, err := os.Stat(staged.VersionPath); err != nil {
		return PublishedDestination{}, fmt.Errorf("deployment: lab version not staged: %w", err)
	}
	if p.pointerFor(slug) == staged.ArtifactHash {
		return p.destination(staged), nil
	}
	if err := writePointerFile(p.pointerPath(slug), staged.ArtifactHash); err != nil {
		return PublishedDestination{}, fmt.Errorf("deployment: swap lab pointer: %w", err)
	}
	return p.destination(staged), nil
}

func (p *labPublisher) destination(staged LabStaged) PublishedDestination {
	return PublishedDestination{
		URL:          p.origin + "/" + staged.Slug,
		DeploymentID: "lab-" + staged.ArtifactHash[:12],
	}
}

// Rollback removes the pointer only if it names this staged version, and
// removes the version directory only if nothing points at it. This keeps a
// rollback from unpublishing an older live release that shares nothing with
// the failed one, while cleaning the failed attempt's own artifacts.
func (p *labPublisher) Rollback(_ context.Context, staged LabStaged) error {
	ptr := p.pointerPath(staged.Slug)
	if raw, err := os.ReadFile(ptr); err == nil {
		var cur struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(raw, &cur) == nil && cur.Version == staged.ArtifactHash {
			_ = os.Remove(ptr)
		}
	}
	if p.pointerFor(staged.Slug) == "" {
		_ = os.RemoveAll(staged.VersionPath)
	}
	return nil
}

// portfolioPublisher is the filesystem PortfolioCatalogPublisher.
type portfolioPublisher struct {
	root   string
	origin string
}

// NewPortfolioCatalogPublisher returns a PortfolioCatalogPublisher rooted at
// root exposing entries below origin (PORTFOLIO_PUBLIC_ORIGIN).
func NewPortfolioCatalogPublisher(root, origin string) PortfolioCatalogPublisher {
	return &portfolioPublisher{root: root, origin: strings.TrimSuffix(strings.TrimSpace(origin), "/")}
}

// catalogDir is the immutable version root for catalog entries.
func (p *portfolioPublisher) catalogDir() string {
	return filepath.Join(p.root, "var", "portfolio", "catalog", "versions")
}

// pointerPath is the per-slug pointer file naming the visible entry version.
func (p *portfolioPublisher) pointerPath(slug string) string {
	return filepath.Join(p.root, "var", "portfolio", "catalog", "pointers", slug+".json")
}

// Stage writes the immutable catalog entry version for hash. Repeat stages
// are no-ops.
func (p *portfolioPublisher) Stage(_ context.Context, req publishing.PublicationRequest) (PortfolioStaged, error) {
	hash, err := normalizeHash(req.Submission.ArtifactSHA256)
	if err != nil {
		return PortfolioStaged{}, err
	}
	slug, err := normalizeSlug(req.Project.Slug)
	if err != nil {
		return PortfolioStaged{}, err
	}
	if strings.TrimSpace(p.root) == "" {
		return PortfolioStaged{}, errors.New("deployment: portfolio artifact store root is not configured")
	}

	versionDir := filepath.Join(p.catalogDir(), hash)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		return PortfolioStaged{}, fmt.Errorf("deployment: create catalog version dir: %w", err)
	}

	staged := PortfolioStaged{ArtifactHash: hash, Slug: slug, VersionPath: versionDir}
	payload, err := json.Marshal(map[string]any{
		"slug":            slug,
		"title":           req.Project.Title,
		"artifact_sha256": hash,
		"metadata":        req.Submission.PortfolioMetadata,
	})
	if err != nil {
		return PortfolioStaged{}, fmt.Errorf("deployment: marshal catalog entry: %w", err)
	}
	sentinel := filepath.Join(versionDir, "entry.json")
	f, err := os.OpenFile(sentinel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return staged, nil
		}
		return PortfolioStaged{}, fmt.Errorf("deployment: stage catalog entry: %w", err)
	}
	if _, err := f.Write(payload); err != nil {
		_ = f.Close()
		return PortfolioStaged{}, fmt.Errorf("deployment: stage catalog entry: %w", err)
	}
	if err := f.Close(); err != nil {
		return PortfolioStaged{}, fmt.Errorf("deployment: stage catalog entry: %w", err)
	}
	return staged, nil
}

// pointerFor returns the catalog version currently visible for slug.
func (p *portfolioPublisher) pointerFor(slug string) string {
	raw, err := os.ReadFile(p.pointerPath(slug))
	if err != nil {
		return ""
	}
	var ptr struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &ptr); err != nil {
		return ""
	}
	return ptr.Version
}

// MakeVisible atomically points the catalog slug at the staged version.
func (p *portfolioPublisher) MakeVisible(_ context.Context, staged PortfolioStaged) (PublishedDestination, error) {
	if strings.TrimSpace(p.origin) == "" {
		return PublishedDestination{}, errors.New("deployment: PORTFOLIO_PUBLIC_ORIGIN is not configured")
	}
	if _, err := normalizeHash(staged.ArtifactHash); err != nil {
		return PublishedDestination{}, err
	}
	slug, err := normalizeSlug(staged.Slug)
	if err != nil {
		return PublishedDestination{}, err
	}
	if _, err := os.Stat(staged.VersionPath); err != nil {
		return PublishedDestination{}, fmt.Errorf("deployment: catalog version not staged: %w", err)
	}
	if p.pointerFor(slug) == staged.ArtifactHash {
		return p.destination(staged), nil
	}
	if err := writePointerFile(p.pointerPath(slug), staged.ArtifactHash); err != nil {
		return PublishedDestination{}, fmt.Errorf("deployment: swap catalog pointer: %w", err)
	}
	return p.destination(staged), nil
}

func (p *portfolioPublisher) destination(staged PortfolioStaged) PublishedDestination {
	return PublishedDestination{
		URL:          p.origin + "/design-lab/" + staged.Slug,
		DeploymentID: "portfolio-" + staged.ArtifactHash[:12],
	}
}

// Rollback removes the catalog pointer only if it names this staged version,
// and the version directory only when unpointed.
func (p *portfolioPublisher) Rollback(_ context.Context, staged PortfolioStaged) error {
	ptr := p.pointerPath(staged.Slug)
	if raw, err := os.ReadFile(ptr); err == nil {
		var cur struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(raw, &cur) == nil && cur.Version == staged.ArtifactHash {
			_ = os.Remove(ptr)
		}
	}
	if p.pointerFor(staged.Slug) == "" {
		_ = os.RemoveAll(staged.VersionPath)
	}
	return nil
}

// atomicPublisher implements AtomicPublisher over one LabPublisher and one
// PortfolioCatalogPublisher with a two-phase commit: both destinations stage
// before either goes visible, so a stage failure publishes nothing; a
// visibility failure rolls back the destination that already flipped.
type atomicPublisher struct {
	lab       LabPublisher
	portfolio PortfolioCatalogPublisher

	mu        sync.Mutex
	published map[string]publishing.PublicationOutput
}

// NewAtomicPublisher returns an AtomicPublisher. It performs no I/O itself;
// retry idempotency is keyed in memory on submission ID + artifact hash.
func NewAtomicPublisher(lab LabPublisher, portfolio PortfolioCatalogPublisher) AtomicPublisher {
	return &atomicPublisher{
		lab:       lab,
		portfolio: portfolio,
		published: make(map[string]publishing.PublicationOutput),
	}
}

// Ruling: idempotency key is submission ID + artifact hash — the publishing
// service already replays whole ApproveAndPublish results by idempotency key,
// so the deployment layer only needs to collapse duplicate Publish calls for
// the same immutable payload — cost if wrong: a same-submission new-hash
// revision would collide (it can't: hash is part of the key).
func idempotencyKey(req publishing.PublicationRequest) string {
	return req.Submission.ID + "\x00" + req.Submission.ArtifactSHA256
}

// Publish stages both destinations, then flips both pointers.
func (a *atomicPublisher) Publish(ctx context.Context, req publishing.PublicationRequest) (publishing.PublicationOutput, error) {
	if err := ctx.Err(); err != nil {
		return publishing.PublicationOutput{}, err
	}
	if a.lab == nil || a.portfolio == nil {
		return publishing.PublicationOutput{}, errors.New("deployment: lab and portfolio publishers are required")
	}

	key := idempotencyKey(req)
	a.mu.Lock()
	if out, ok := a.published[key]; ok {
		a.mu.Unlock()
		return out, nil
	}
	a.mu.Unlock()

	// Phase 1: stage both destinations. Neither is visible yet.
	labStaged, err := a.lab.Stage(ctx, req)
	if err != nil {
		return publishing.PublicationOutput{}, fmt.Errorf("deployment: stage lab: %w", err)
	}
	portfolioStaged, err := a.portfolio.Stage(ctx, req)
	if err != nil {
		_ = rollbackLab(context.Background(), a.lab, labStaged)
		return publishing.PublicationOutput{}, fmt.Errorf("deployment: stage portfolio: %w", err)
	}

	// Phase 2: final pointer swaps. Public visibility changes only here.
	labPub, err := a.lab.MakeVisible(ctx, labStaged)
	if err != nil {
		_ = rollbackLab(context.Background(), a.lab, labStaged)
		_ = rollbackPortfolio(context.Background(), a.portfolio, portfolioStaged)
		return publishing.PublicationOutput{}, fmt.Errorf("deployment: publish lab: %w", err)
	}
	portfolioPub, err := a.portfolio.MakeVisible(ctx, portfolioStaged)
	if err != nil {
		_ = rollbackLab(context.Background(), a.lab, labStaged)
		_ = rollbackPortfolio(context.Background(), a.portfolio, portfolioStaged)
		return publishing.PublicationOutput{}, fmt.Errorf("deployment: publish portfolio: %w", err)
	}

	// Ruling: outputs bind both URLs and both deployment IDs to the same
	// artifact hash (deployment IDs embed the hash prefix) — the Task 15
	// acceptance proof that one immutable artifact reached both destinations
	// — cost if wrong: Task 16 could not verify Lab and portfolio serve the
	// same content.
	out := publishing.PublicationOutput{
		LabURL:          labPub.URL,
		PortfolioURL:    portfolioPub.URL,
		DeploymentIDs:   []string{labPub.DeploymentID, portfolioPub.DeploymentID},
		DestinationURLs: []string{labPub.URL, portfolioPub.URL},
	}

	a.mu.Lock()
	a.published[key] = out
	a.mu.Unlock()
	return out, nil
}

// writePointerFile swaps slug's visible version atomically: write temp +
// rename, fsync-free like the preview runner's staging-rename (durable
// enough for a pointer file; content loss only reverts to the prior
// version).
func writePointerFile(path, version string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]string{"version": version})
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".pointer-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
