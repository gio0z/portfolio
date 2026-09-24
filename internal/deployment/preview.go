// Package deployment publishes approved Lab artifacts to their public
// destinations.
//
// Preview, Lab, and portfolio publication all share one rule: immutable
// version paths plus a final pointer swap. Nothing becomes publicly visible
// until every destination has staged successfully, and a failure in any
// destination rolls back the ones that already moved.
package deployment

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"portfolio/internal/preview"
)

// PreviewDeployer deploys an immutable source artifact to the sandboxed
// preview runtime. It is a thin adapter over preview.PreviewRunner so Task 4
// wiring depends on this package, not on the preview internals.
type PreviewDeployer interface {
	Deploy(context.Context, preview.BuildRequest) (preview.PreviewResult, error)
}

type previewDeployer struct {
	runner preview.PreviewRunner
}

// NewPreviewDeployer returns a PreviewDeployer backed by runner.
func NewPreviewDeployer(runner preview.PreviewRunner) PreviewDeployer {
	return &previewDeployer{runner: runner}
}

// Deploy builds the source artifact into an immutable preview bundle.
func (d *previewDeployer) Deploy(ctx context.Context, req preview.BuildRequest) (preview.PreviewResult, error) {
	if err := ctx.Err(); err != nil {
		return preview.PreviewResult{}, err
	}
	if d.runner == nil {
		return preview.PreviewResult{}, errors.New("deployment: preview runner is not configured")
	}
	return d.runner.Build(ctx, req)
}

// NewPreviewDeployerFromConfig wires preview.NewRunner with the artifact-store
// bindings below, keeping environment handling in application composition.
func NewPreviewDeployerFromConfig(cfg preview.Config) PreviewDeployer {
	return NewPreviewDeployer(preview.NewRunner(cfg))
}

// Config carries the deployment bindings. Application composition fills it
// from the environment:
//
//	ARTIFACT_STORE_ROOT -> ArtifactStoreRoot
//	PREVIEW_PUBLIC_ORIGIN -> PreviewOrigin
//	LAB_PUBLIC_ORIGIN -> LabOrigin
//	PORTFOLIO_PUBLIC_ORIGIN -> PortfolioOrigin
type Config struct {
	ArtifactStoreRoot string
	PreviewOrigin     string
	LabOrigin         string
	PortfolioOrigin   string
}

// LoadConfigFromEnv reads the deployment bindings from the environment.
// Empty values are reported lazily by Stage/MakeVisible so misconfiguration
// surfaces at publish time, not at wiring time.
func LoadConfigFromEnv() Config {
	return Config{
		ArtifactStoreRoot: os.Getenv("ARTIFACT_STORE_ROOT"),
		PreviewOrigin:     os.Getenv("PREVIEW_PUBLIC_ORIGIN"),
		LabOrigin:         os.Getenv("LAB_PUBLIC_ORIGIN"),
		PortfolioOrigin:   os.Getenv("PORTFOLIO_PUBLIC_ORIGIN"),
	}
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// normalizeHash lowercases and validates a SHA-256 artifact hash.
func normalizeHash(hash string) (string, error) {
	h := strings.ToLower(strings.TrimSpace(hash))
	if len(h) != sha256HexLen {
		return "", fmt.Errorf("deployment: invalid artifact hash %q: want 64 hex characters", hash)
	}
	if _, err := hex.DecodeString(h); err != nil {
		return "", fmt.Errorf("deployment: invalid artifact hash %q: not hex", hash)
	}
	return h, nil
}

const sha256HexLen = 64

// normalizeSlug validates a public slug. It mirrors the publishing slug rule
// so malformed slugs fail here before touching the filesystem.
func normalizeSlug(slug string) (string, error) {
	s := strings.TrimSpace(slug)
	if !slugPattern.MatchString(s) {
		return "", fmt.Errorf("deployment: invalid slug %q", slug)
	}
	return s, nil
}
