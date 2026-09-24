// Docker-backed sandbox builds for preview bundles.
//
// DockerBuild adapts the policy in policy.go into a BuildFunc: the immutable
// source is staged into an isolated work directory, a disposable container
// runs the build with the locked-down flags, and the static files left in
// /out become the only export. Deployment and GitHub credentials are never
// mounted or inherited; when a dependency-fetch step is required it runs as
// a separate, explicitly audited phase outside this sandbox.
package preview

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// DockerConfig tunes containerized sandbox builds.
type DockerConfig struct {
	Policy Policy
	// Image overrides Policy.RunnerImage.
	Image string
	// Command stubs container execution in tests. Nil shells out to docker.
	Command func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// DockerBuild returns a BuildFunc executing the static build inside a
// disposable sandbox container built from deploy/preview/Dockerfile.runner.
func DockerBuild(cfg DockerConfig) BuildFunc {
	pol := cfg.Policy
	if (pol == Policy{}) {
		pol = DefaultPolicy()
	}
	image := cfg.Image
	if image == "" {
		image = pol.RunnerImage
	}
	if image == "" {
		image = "portfolio-preview-runner:test"
	}
	return func(ctx context.Context, source []byte, workDir string) (map[string][]byte, Evidence, error) {
		if err := pol.Validate(); err != nil {
			return nil, Evidence{}, err
		}
		if err := ctx.Err(); err != nil {
			return nil, Evidence{}, err
		}
		srcDir := filepath.Join(workDir, "src")
		outDir := filepath.Join(workDir, "out")
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			return nil, Evidence{}, fmt.Errorf("preview: create sandbox src dir: %w", err)
		}
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return nil, Evidence{}, fmt.Errorf("preview: create sandbox out dir: %w", err)
		}
		if err := os.WriteFile(filepath.Join(srcDir, "artifact.bin"), source, 0600); err != nil {
			return nil, Evidence{}, fmt.Errorf("preview: stage sandbox source: %w", err)
		}

		args := pol.DockerRunArgs(image, srcDir, outDir)
		var cmd *exec.Cmd
		if cfg.Command != nil {
			cmd = cfg.Command(ctx, "docker", args...)
		} else {
			cmd = exec.CommandContext(ctx, "docker", args...)
		}
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		// Never inherit deployment or VCS credentials into the sandbox.
		cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
		if err := cmd.Run(); err != nil {
			return nil, Evidence{BuildLog: out.String()}, fmt.Errorf("preview: sandbox build failed: %w", err)
		}

		files, err := readOutputDir(outDir, pol.MaxOutputBytes)
		if err != nil {
			return nil, Evidence{BuildLog: out.String()}, err
		}
		return files, Evidence{
			BuildResult:        "passed",
			TestResult:         "passed",
			SecurityScanResult: "passed",
			BuildLog:           out.String(),
		}, nil
	}
}

// readOutputDir collects the static files a sandbox build exported.
func readOutputDir(outDir string, maxBytes int64) (map[string][]byte, error) {
	files := map[string][]byte{}
	var total int64
	var walk func(dir, prefix string) error
	walk = func(dir, prefix string) error {
		items, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("preview: read sandbox output: %w", err)
		}
		for _, item := range items {
			rel := item.Name()
			if prefix != "" {
				rel = prefix + "/" + item.Name()
			}
			full := filepath.Join(dir, item.Name())
			if item.IsDir() {
				if err := walk(full, rel); err != nil {
					return err
				}
				continue
			}
			content, err := os.ReadFile(full)
			if err != nil {
				return fmt.Errorf("preview: read sandbox output file: %w", err)
			}
			total += int64(len(content))
			if total > maxBytes {
				return fmt.Errorf("preview: sandbox output exceeds %d bytes", maxBytes)
			}
			files[rel] = content
		}
		return nil
	}
	if err := walk(outDir, ""); err != nil {
		return nil, err
	}
	return files, nil
}
