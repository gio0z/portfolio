// Policy defines the sandbox bounds every preview build must satisfy:
// non-root execution, a read-only base filesystem, default-deny network,
// bounded CPU/memory/PIDs/disk/time, no host or credential mounts, and
// static output as the only export.
package preview

import (
	"errors"
	"strconv"
	"time"
)

// Policy bounds sandboxed preview builds.
type Policy struct {
	// User is the unprivileged account the build runs as. Never root.
	User string
	// ReadOnlyRootFS mounts the container base filesystem read-only.
	ReadOnlyRootFS bool
	// AllowNetwork must stay false: builds get no outbound network except
	// an explicit, separately audited dependency-fetch step.
	AllowNetwork bool
	// CPUs caps build CPU (docker --cpus value, e.g. "1.0").
	CPUs string
	// Memory caps build memory (docker --memory value, e.g. "512m").
	Memory string
	// PIDsLimit caps the build process count (docker --pids-limit).
	PIDsLimit int64
	// BuildTimeout kills hung builds.
	BuildTimeout time.Duration
	// MaxSourceBytes caps the immutable source artifact size.
	MaxSourceBytes int64
	// MaxOutputBytes caps the total static output size.
	MaxOutputBytes int64
	// RunnerImage is the sandbox image used for containerized builds.
	RunnerImage string
}

// DefaultPolicy returns the locked-down sandbox defaults.
func DefaultPolicy() Policy {
	return Policy{
		User:           "nonroot",
		ReadOnlyRootFS: true,
		AllowNetwork:   false,
		CPUs:           "1.0",
		Memory:         "512m",
		PIDsLimit:      256,
		BuildTimeout:   10 * time.Minute,
		MaxSourceBytes: 8 << 20,
		MaxOutputBytes: 32 << 20,
		RunnerImage:    "portfolio-preview-runner:test",
	}
}

// Validate rejects any policy that would weaken the sandbox.
func (p Policy) Validate() error {
	if p.User == "" || p.User == "root" || p.User == "0" || p.User == "0:0" {
		return errors.New("preview: sandbox must run as a non-root user")
	}
	if !p.ReadOnlyRootFS {
		return errors.New("preview: sandbox requires a read-only base filesystem")
	}
	if p.AllowNetwork {
		return errors.New("preview: sandbox requires default-deny network")
	}
	if p.CPUs == "" {
		return errors.New("preview: sandbox requires a CPU limit")
	}
	if p.Memory == "" {
		return errors.New("preview: sandbox requires a memory limit")
	}
	if p.PIDsLimit <= 0 {
		return errors.New("preview: sandbox requires a PID limit")
	}
	if p.BuildTimeout <= 0 {
		return errors.New("preview: sandbox requires a build timeout")
	}
	if p.MaxSourceBytes <= 0 {
		return errors.New("preview: sandbox requires a source size limit")
	}
	if p.MaxOutputBytes <= 0 {
		return errors.New("preview: sandbox requires an output size limit")
	}
	return nil
}

// DockerRunArgs renders the `docker run` argv enforcing the policy. The
// command mounts the immutable source read-only at /src and collects static
// output from /out as the only export. It never mounts the Docker socket,
// host home directories, or credential material, and never grants privileges.
func (p Policy) DockerRunArgs(image, srcDir, outDir string) []string {
	if image == "" {
		image = p.RunnerImage
	}
	if image == "" {
		image = "portfolio-preview-runner:test"
	}
	return []string{
		"run", "--rm",
		"--read-only",
		"--network=none",
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--pids-limit=" + strconv.FormatInt(p.PIDsLimit, 10),
		"--memory=" + p.Memory,
		"--cpus=" + p.CPUs,
		"--user=" + p.User,
		"--tmpfs=/tmp:rw,noexec,nosuid,size=64m",
		"--volume=" + srcDir + ":/src:ro",
		"--volume=" + outDir + ":/out",
		image,
	}
}
