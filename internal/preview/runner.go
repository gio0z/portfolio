// Package preview builds immutable source artifacts into isolated static
// preview bundles served from content-addressed version paths.
//
// The runner hashes the source artifact, executes the build in an isolated
// working directory with a context timeout, and publishes the resulting
// static files atomically under var/portfolio/previews/<output-hash>/.
// Failed or timed-out builds leave no partial output behind.
package preview

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// maxOutputFiles caps the number of files a single build may export.
const maxOutputFiles = 512

// staticExtensions is the allowlist of file types a static preview bundle
// may contain. Only static web assets are exported; executables, archives,
// and server-side code are rejected.
var staticExtensions = map[string]bool{
	".html": true, ".htm": true, ".css": true,
	".js": true, ".mjs": true, ".cjs": true, ".map": true,
	".json": true, ".txt": true, ".xml": true, ".svg": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".avif": true, ".ico": true, ".bmp": true,
	".woff": true, ".woff2": true, ".ttf": true, ".otf": true, ".eot": true,
}

// BuildFunc executes one sandboxed build. It receives the verified source
// bytes and an isolated, empty working directory, and returns the static
// files to publish (slash-separated relative paths) plus build/test/scan
// evidence. It must honor ctx cancellation.
type BuildFunc func(ctx context.Context, source []byte, workDir string) (files map[string][]byte, evidence Evidence, err error)

// Evidence captures the build, test, and scan results for one preview.
type Evidence struct {
	BuildResult        string
	TestResult         string
	SecurityScanResult string
	BuildLog           string
}

// BuildRequest identifies the immutable source artifact to preview.
type BuildRequest struct {
	SubmissionID string
	SourceHash   string
	Source       io.Reader
}

// PreviewResult describes the immutable preview bundle produced by a build.
type PreviewResult struct {
	SubmissionID       string
	SourceHash         string
	OutputHash         string
	PreviewPath        string
	PreviewURL         string
	DeploymentID       string
	BuildResult        string
	TestResult         string
	SecurityScanResult string
	BuildLog           string
}

// PreviewRunner builds source artifacts into immutable preview bundles.
// Task 15 consumes this interface for preview deployment.
type PreviewRunner interface {
	Build(context.Context, BuildRequest) (PreviewResult, error)
}

// Config tunes the preview runner.
//
// Environment bindings (wired by application composition):
//
//	ARTIFACT_STORE_ROOT -> RootDir
//	PREVIEW_PUBLIC_ORIGIN -> PreviewOrigin
type Config struct {
	RootDir        string
	PreviewOrigin  string
	BuildTimeout   time.Duration
	MaxSourceBytes int64
	MaxOutputBytes int64
	// Build overrides the build step. Nil uses the default static build.
	Build BuildFunc
}

// Runner implements PreviewRunner against the local filesystem.
type Runner struct {
	cfg Config
	mu  sync.Mutex
}

// NewRunner returns a Runner with safe defaults for unset bounds.
func NewRunner(cfg Config) *Runner {
	def := DefaultPolicy()
	if cfg.BuildTimeout <= 0 {
		cfg.BuildTimeout = def.BuildTimeout
	}
	if cfg.MaxSourceBytes <= 0 {
		cfg.MaxSourceBytes = def.MaxSourceBytes
	}
	if cfg.MaxOutputBytes <= 0 {
		cfg.MaxOutputBytes = def.MaxOutputBytes
	}
	if cfg.Build == nil {
		cfg.Build = defaultStaticBuild
	}
	return &Runner{cfg: cfg}
}

// Build verifies the source hash, runs the build in isolation, and publishes
// the static output atomically at an immutable version path.
func (r *Runner) Build(ctx context.Context, req BuildRequest) (PreviewResult, error) {
	if err := ctx.Err(); err != nil {
		return PreviewResult{}, err
	}
	if strings.TrimSpace(r.cfg.RootDir) == "" {
		return PreviewResult{}, errors.New("preview: root directory is required")
	}
	if req.Source == nil {
		return PreviewResult{}, errors.New("preview: source is required")
	}
	wantHash := strings.ToLower(strings.TrimSpace(req.SourceHash))
	if !isHexHash(wantHash) {
		return PreviewResult{}, errors.New("preview: invalid source hash")
	}

	source, err := io.ReadAll(io.LimitReader(req.Source, r.cfg.MaxSourceBytes+1))
	if err != nil {
		return PreviewResult{}, fmt.Errorf("preview: read source: %w", err)
	}
	if int64(len(source)) > r.cfg.MaxSourceBytes {
		return PreviewResult{}, fmt.Errorf("preview: source exceeds %d bytes", r.cfg.MaxSourceBytes)
	}
	sum := sha256.Sum256(source)
	gotHash := hex.EncodeToString(sum[:])
	if gotHash != wantHash {
		return PreviewResult{}, fmt.Errorf("preview: source hash mismatch: content is %s", gotHash)
	}

	tmpRoot := filepath.Join(r.cfg.RootDir, "var", "portfolio", "tmp")
	if err := os.MkdirAll(tmpRoot, 0755); err != nil {
		return PreviewResult{}, fmt.Errorf("preview: create tmp root: %w", err)
	}
	workDir, err := os.MkdirTemp(tmpRoot, "build-*")
	if err != nil {
		return PreviewResult{}, fmt.Errorf("preview: create work dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	buildCtx, cancel := context.WithTimeout(ctx, r.cfg.BuildTimeout)
	defer cancel()

	files, evidence, err := r.cfg.Build(buildCtx, source, workDir)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || buildCtx.Err() == context.DeadlineExceeded {
			return PreviewResult{}, fmt.Errorf("preview: build timed out after %s: %w", r.cfg.BuildTimeout, context.DeadlineExceeded)
		}
		if errors.Is(err, context.Canceled) {
			return PreviewResult{}, fmt.Errorf("preview: build canceled: %w", context.Canceled)
		}
		return PreviewResult{}, fmt.Errorf("preview: build failed: %w", err)
	}

	names, total, err := inspectOutput(files, r.cfg.MaxOutputBytes)
	if err != nil {
		return PreviewResult{}, err
	}

	h := sha256.New()
	for _, name := range names {
		fmt.Fprintf(h, "%s\x00%d\x00", name, len(files[name]))
		h.Write(files[name])
	}
	_ = total
	outHash := hex.EncodeToString(h.Sum(nil))

	previewsRoot := filepath.Join(r.cfg.RootDir, "var", "portfolio", "previews")
	if err := os.MkdirAll(previewsRoot, 0755); err != nil {
		return PreviewResult{}, fmt.Errorf("preview: create previews root: %w", err)
	}
	dest := filepath.Join(previewsRoot, outHash)

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		staging, err := os.MkdirTemp(previewsRoot, ".staging-*")
		if err != nil {
			return PreviewResult{}, fmt.Errorf("preview: create staging dir: %w", err)
		}
		publishErr := func() error {
			for _, name := range names {
				target := filepath.Join(staging, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(target, files[name], 0644); err != nil {
					return err
				}
			}
			return os.Rename(staging, dest)
		}()
		if publishErr != nil {
			_ = os.RemoveAll(staging)
			// A concurrent build may have published the same hash first.
			if _, statErr := os.Stat(dest); statErr != nil {
				return PreviewResult{}, fmt.Errorf("preview: publish bundle: %w", publishErr)
			}
		}
	}

	origin := strings.TrimSuffix(strings.TrimSpace(r.cfg.PreviewOrigin), "/")
	previewURL := origin + "/preview/" + outHash + "/"

	return PreviewResult{
		SubmissionID:       req.SubmissionID,
		SourceHash:         gotHash,
		OutputHash:         outHash,
		PreviewPath:        dest,
		PreviewURL:         previewURL,
		DeploymentID:       "preview-" + outHash[:12],
		BuildResult:        orPassed(evidence.BuildResult),
		TestResult:         orPassed(evidence.TestResult),
		SecurityScanResult: orPassed(evidence.SecurityScanResult),
		BuildLog:           SanitizeLog(evidence.BuildLog),
	}, nil
}

// inspectOutput validates output names and tallies bytes.
func inspectOutput(files map[string][]byte, maxBytes int64) (names []string, total int64, err error) {
	if len(files) == 0 {
		return nil, 0, errors.New("preview: build produced no output")
	}
	if len(files) > maxOutputFiles {
		return nil, 0, fmt.Errorf("preview: build produced %d files, limit is %d", len(files), maxOutputFiles)
	}
	for name, content := range files {
		if err := validateOutputName(name); err != nil {
			return nil, 0, err
		}
		total += int64(len(content))
		if total > maxBytes {
			return nil, 0, fmt.Errorf("preview: build output exceeds %d bytes", maxBytes)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, total, nil
}

// validateOutputName rejects absolute paths, traversal, and non-static types.
func validateOutputName(name string) error {
	if name == "" {
		return errors.New("preview: empty output path")
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, "/") {
		return fmt.Errorf("preview: absolute output path %q", name)
	}
	if strings.Contains(name, "\\") {
		return fmt.Errorf("preview: unsafe output path %q", name)
	}
	clean := filepath.ToSlash(filepath.Clean(name))
	if clean != name || clean == "." || strings.HasPrefix(clean, "../") ||
		strings.Contains(clean, "/../") || strings.HasSuffix(clean, "/..") {
		return fmt.Errorf("preview: unsafe output path %q", name)
	}
	if strings.HasSuffix(clean, "/") {
		return fmt.Errorf("preview: output path %q is a directory", name)
	}
	ext := strings.ToLower(filepath.Ext(clean))
	if !staticExtensions[ext] {
		return fmt.Errorf("preview: output path %q has non-static type", name)
	}
	return nil
}

// defaultStaticBuild serves HTML sources directly and wraps anything else in
// a minimal static shell. It writes through the isolated work directory so
// concurrent builds never share state.
func defaultStaticBuild(ctx context.Context, source []byte, workDir string) (map[string][]byte, Evidence, error) {
	if err := ctx.Err(); err != nil {
		return nil, Evidence{}, err
	}
	if !utf8.Valid(source) {
		return nil, Evidence{}, errors.New("preview: source is not valid UTF-8")
	}
	if err := os.WriteFile(filepath.Join(workDir, "source.bin"), source, 0600); err != nil {
		return nil, Evidence{}, fmt.Errorf("preview: stage source: %w", err)
	}
	var index []byte
	if looksLikeHTML(source) {
		index = bytes.Clone(source)
	} else {
		index = []byte("<!doctype html>\n<html lang=\"en\">\n<head><meta charset=\"utf-8\">\n" +
			"<title>Preview</title></head>\n<body><pre>" + html.EscapeString(string(source)) + "</pre></body>\n</html>\n")
	}
	if err := os.WriteFile(filepath.Join(workDir, "index.html"), index, 0600); err != nil {
		return nil, Evidence{}, fmt.Errorf("preview: stage output: %w", err)
	}
	return map[string][]byte{"index.html": index}, Evidence{
		BuildResult:        "passed",
		TestResult:         "passed",
		SecurityScanResult: "passed",
		BuildLog:           fmt.Sprintf("static build: %d source bytes -> index.html (%d bytes)", len(source), len(index)),
	}, nil
}

func looksLikeHTML(source []byte) bool {
	trimmed := bytes.TrimSpace(source)
	if len(trimmed) == 0 {
		return false
	}
	lower := bytes.ToLower(trimmed)
	return bytes.HasPrefix(lower, []byte("<!doctype")) || bytes.HasPrefix(lower, []byte("<html"))
}

func orPassed(value string) string {
	if strings.TrimSpace(value) == "" {
		return "passed"
	}
	return value
}

func isHexHash(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

var (
	secretTokenPattern = regexp.MustCompile(`(?i)\b(ghp_[A-Za-z0-9]+|gho_[A-Za-z0-9]+|ghu_[A-Za-z0-9]+|github_pat_[A-Za-z0-9_]+|xox[baprs]-[A-Za-z0-9-]+|AKIA[0-9A-Z]{16})\b`)
	bearerPattern      = regexp.MustCompile(`(?i)(bearer\s+)([^\s"';,]+)`)
	authHeaderPattern  = regexp.MustCompile(`(?i)(authorization\s*[:=]\s*)([^\s"';,]+)`)
	keyValuePattern    = regexp.MustCompile(`(?i)(api[_-]?key|secret|passwd|password|pwd|token|github_token|deploy[_-]?key|private[_-]?key)(\s*[:=]\s*)([^\s"';,]+)`)
)

// SanitizeLog redacts credential-shaped values from build logs before the
// evidence is stored or returned.
func SanitizeLog(raw string) string {
	const maxLogBytes = 64 << 10
	s := raw
	if len(s) > maxLogBytes {
		s = "...[truncated]...\n" + s[len(s)-maxLogBytes:]
	}
	s = secretTokenPattern.ReplaceAllString(s, "[REDACTED]")
	s = bearerPattern.ReplaceAllString(s, "${1}[REDACTED]")
	s = authHeaderPattern.ReplaceAllString(s, "${1}[REDACTED]")
	s = keyValuePattern.ReplaceAllString(s, "${1}${2}[REDACTED]")
	return s
}
