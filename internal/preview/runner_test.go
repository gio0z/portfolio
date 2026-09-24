package preview_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"portfolio/internal/preview"
)

func testConfig(root string) preview.Config {
	return preview.Config{
		RootDir:        root,
		PreviewOrigin:  "https://preview.example.test",
		BuildTimeout:   5 * time.Second,
		MaxSourceBytes: 1 << 20,
		MaxOutputBytes: 2 << 20,
	}
}

func sourceFixture(content string) (hash string, data []byte) {
	data = []byte(content)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), data
}

func TestBuild_ProducesIsolatedBundleAtImmutableVersionPath(t *testing.T) {
	root := t.TempDir()
	r := preview.NewRunner(testConfig(root))

	hash, data := sourceFixture("<html><body>hello lab</body></html>")
	res, err := r.Build(context.Background(), preview.BuildRequest{
		SubmissionID: "submission_1",
		SourceHash:   hash,
		Source:       bytes.NewReader(data),
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if res.SourceHash != hash {
		t.Fatalf("SourceHash = %q, want %q", res.SourceHash, hash)
	}
	if len(res.OutputHash) != 64 {
		t.Fatalf("OutputHash = %q, want 64-char hex", res.OutputHash)
	}
	if !strings.Contains(res.PreviewPath, res.OutputHash) {
		t.Fatalf("PreviewPath %q does not contain output hash %q", res.PreviewPath, res.OutputHash)
	}
	wantURL := "https://preview.example.test/preview/" + res.OutputHash + "/"
	if res.PreviewURL != wantURL {
		t.Fatalf("PreviewURL = %q, want %q", res.PreviewURL, wantURL)
	}
	if res.DeploymentID == "" {
		t.Fatal("DeploymentID is empty")
	}
	index := filepath.Join(res.PreviewPath, "index.html")
	if _, err := os.Stat(index); err != nil {
		t.Fatalf("expected bundle index.html: %v", err)
	}
	if res.BuildResult != "passed" || res.TestResult != "passed" || res.SecurityScanResult != "passed" {
		t.Fatalf("unexpected evidence: %+v", res)
	}
}

func TestBuild_RejectsSourceHashMismatch(t *testing.T) {
	root := t.TempDir()
	r := preview.NewRunner(testConfig(root))

	_, data := sourceFixture("<html><body>real</body></html>")
	mismatch := strings.Repeat("0", 64)
	_, err := r.Build(context.Background(), preview.BuildRequest{
		SubmissionID: "submission_2",
		SourceHash:   mismatch,
		Source:       bytes.NewReader(data),
	})
	if err == nil {
		t.Fatal("expected hash mismatch error, got nil")
	}
	entries, _ := os.ReadDir(filepath.Join(root, "var", "portfolio", "previews"))
	if len(entries) != 0 {
		t.Fatalf("expected no preview output, found %d entries", len(entries))
	}
}

func TestBuild_ConcurrentBuildsAreIsolated(t *testing.T) {
	root := t.TempDir()
	r := preview.NewRunner(testConfig(root))

	const n = 8
	results := make([]preview.PreviewResult, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			hash, data := sourceFixture("<html><body>project project-" + string(rune('a'+i)) + "</body></html>")
			res, err := r.Build(context.Background(), preview.BuildRequest{
				SubmissionID: "submission_concurrent",
				SourceHash:   hash,
				Source:       bytes.NewReader(data),
			})
			results[i] = res
			errs[i] = err
		}(i)
	}
	wg.Wait()
	seen := map[string]bool{}
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("build %d failed: %v", i, errs[i])
		}
		if seen[results[i].OutputHash] {
			t.Fatalf("duplicate output hash %q: builds not isolated", results[i].OutputHash)
		}
		seen[results[i].OutputHash] = true
		if _, err := os.Stat(filepath.Join(results[i].PreviewPath, "index.html")); err != nil {
			t.Fatalf("build %d bundle missing: %v", i, err)
		}
	}
}

func TestBuild_FailureLeavesNoPartialOutput(t *testing.T) {
	root := t.TempDir()
	cfg := testConfig(root)
	cfg.Build = func(ctx context.Context, source []byte, workDir string) (map[string][]byte, preview.Evidence, error) {
		return nil, preview.Evidence{}, errors.New("boom: fixture build failure")
	}
	r := preview.NewRunner(cfg)

	hash, data := sourceFixture("<html><body>doomed</body></html>")
	_, err := r.Build(context.Background(), preview.BuildRequest{
		SubmissionID: "submission_fail",
		SourceHash:   hash,
		Source:       bytes.NewReader(data),
	})
	if err == nil {
		t.Fatal("expected build error, got nil")
	}
	previews := filepath.Join(root, "var", "portfolio", "previews")
	var found []string
	_ = filepath.Walk(previews, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			found = append(found, p)
		}
		return nil
	})
	if len(found) != 0 {
		t.Fatalf("expected no partial output, found %v", found)
	}
}

func TestBuild_TimeoutKillsHungBuild(t *testing.T) {
	root := t.TempDir()
	cfg := testConfig(root)
	cfg.BuildTimeout = 100 * time.Millisecond
	cfg.Build = func(ctx context.Context, source []byte, workDir string) (map[string][]byte, preview.Evidence, error) {
		<-ctx.Done()
		return nil, preview.Evidence{}, ctx.Err()
	}
	r := preview.NewRunner(cfg)

	hash, data := sourceFixture("<html><body>hung</body></html>")
	start := time.Now()
	_, err := r.Build(context.Background(), preview.BuildRequest{
		SubmissionID: "submission_hung",
		SourceHash:   hash,
		Source:       bytes.NewReader(data),
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context timeout error, got %v", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("hung build not killed promptly: %v", elapsed)
	}
	previews := filepath.Join(root, "var", "portfolio", "previews")
	var found []string
	_ = filepath.Walk(previews, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			found = append(found, p)
		}
		return nil
	})
	if len(found) != 0 {
		t.Fatalf("expected no partial output after timeout, found %v", found)
	}
}

func TestDockerArgs_EnforceSandboxPolicy(t *testing.T) {
	pol := preview.DefaultPolicy()
	if err := pol.Validate(); err != nil {
		t.Fatalf("DefaultPolicy invalid: %v", err)
	}
	args := pol.DockerRunArgs("portfolio-preview-runner:test", "/src", "/out")
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"--read-only",
		"--network=none",
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--pids-limit",
		"--memory",
		"--cpus",
		"--user",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("docker args missing %q: %q", want, joined)
		}
	}
	for _, forbidden := range []string{"docker.sock", "/root", "/home/", "--privileged", "github_token", "GITHUB_TOKEN"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("docker args must not contain %q: %q", forbidden, joined)
		}
	}
}

func TestSanitizeLog_RedactsSecrets(t *testing.T) {
	raw := "build ok\ngithub_token=ghp_superSecretValue12345\nAuthorization: Bearer hunter2hunter2\nAKIAIOSFODNN7EXAMPLE done\n"
	clean := preview.SanitizeLog(raw)
	if strings.Contains(clean, "ghp_superSecretValue12345") || strings.Contains(clean, "hunter2hunter2") || strings.Contains(clean, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("secrets not redacted: %q", clean)
	}
	if !strings.Contains(clean, "build ok") {
		t.Fatalf("non-secret log content lost: %q", clean)
	}
}

func TestDefaultPolicy_IsBounded(t *testing.T) {
	pol := preview.DefaultPolicy()
	if pol.CPUs == "" || pol.Memory == "" || pol.PIDsLimit <= 0 {
		t.Fatalf("CPU/memory/PID bounds required: %+v", pol)
	}
	if pol.BuildTimeout <= 0 {
		t.Fatalf("time bound required: %+v", pol)
	}
	if pol.MaxSourceBytes <= 0 || pol.MaxOutputBytes <= 0 {
		t.Fatalf("disk bounds required: %+v", pol)
	}
	if !pol.ReadOnlyRootFS || pol.AllowNetwork {
		t.Fatalf("read-only base and default-deny network required: %+v", pol)
	}
	if pol.User == "" || pol.User == "root" || pol.User == "0" || pol.User == "0:0" {
		t.Fatalf("non-root user required: %+v", pol)
	}
}
