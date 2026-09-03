package publishing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// LocalArtifactStore implements ArtifactStore on the local filesystem using
// content-addressed SHA-256 storage and safe atomic writes.
type LocalArtifactStore struct {
	root   string
	policy AssetPolicy
	mu     sync.RWMutex
}

// NewLocalArtifactStore initializes a local artifact store under root.
func NewLocalArtifactStore(root string, policy AssetPolicy) *LocalArtifactStore {
	if policy.MaxBytes <= 0 {
		policy.MaxBytes = defaultMaxAssetBytes
	}
	return &LocalArtifactStore{
		root:   root,
		policy: policy,
	}
}

func (s *LocalArtifactStore) artifactDir(hash string) string {
	return filepath.Join(s.root, "var", "portfolio", "artifacts", "sha256", hash[:2])
}

func (s *LocalArtifactStore) artifactPath(hash string) string {
	return filepath.Join(s.artifactDir(hash), hash)
}

func (s *LocalArtifactStore) metaPath(hash string) string {
	return filepath.Join(s.root, "var", "portfolio", "metadata", hash[:2], hash+".json")
}

func (s *LocalArtifactStore) publicationsDir(projectID string) string {
	return filepath.Join(s.root, "var", "portfolio", "publications", projectID)
}

func (s *LocalArtifactStore) previewsDir(hash string) string {
	return filepath.Join(s.root, "var", "portfolio", "previews", hash)
}

// Put validates an incoming asset and writes it to content-addressed storage.
// Repeated puts with identical content return the existing reference without
// rewriting the file.
func (s *LocalArtifactStore) Put(ctx context.Context, src io.Reader, metadata AssetMetadata) (ArtifactRef, error) {
	if err := ctx.Err(); err != nil {
		return ArtifactRef{}, err
	}
	if src == nil {
		return ArtifactRef{}, errors.New("publishing: asset source is required")
	}

	tempDir := filepath.Join(s.root, "var", "portfolio", "tmp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return ArtifactRef{}, fmt.Errorf("publishing: create tmp dir: %w", err)
	}

	tempFile, err := os.CreateTemp(tempDir, "artifact-*")
	if err != nil {
		return ArtifactRef{}, fmt.Errorf("publishing: create temp artifact: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	validated, err := ValidateAndHash(ctx, tempFile, src, metadata, s.policy)
	if err != nil {
		_ = tempFile.Close()
		return ArtifactRef{}, err
	}

	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return ArtifactRef{}, fmt.Errorf("publishing: sync temp artifact: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return ArtifactRef{}, fmt.Errorf("publishing: close temp artifact: %w", err)
	}

	targetPath := s.artifactPath(validated.SHA256)
	targetDir := s.artifactDir(validated.SHA256)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return ArtifactRef{}, fmt.Errorf("publishing: create artifact dir: %w", err)
	}

	ref := ArtifactRef{
		SHA256:   validated.SHA256,
		MIMEType: validated.MIMEType,
		Size:     validated.Size,
	}

	if _, err := os.Stat(targetPath); err == nil {
		return ref, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return ArtifactRef{}, fmt.Errorf("publishing: stat target artifact: %w", err)
	}

	if err := os.Chmod(tempPath, 0444); err != nil {
		return ArtifactRef{}, fmt.Errorf("publishing: set read-only perm: %w", err)
	}

	if err := os.Rename(tempPath, targetPath); err != nil {
		return ArtifactRef{}, fmt.Errorf("publishing: commit artifact: %w", err)
	}

	metaBytes, err := json.Marshal(ref)
	if err == nil {
		metaDir := filepath.Dir(s.metaPath(validated.SHA256))
		if err := os.MkdirAll(metaDir, 0755); err == nil {
			_ = os.WriteFile(s.metaPath(validated.SHA256), metaBytes, 0644)
		}
	}

	return ref, nil
}

func isValidHexHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for i := 0; i < len(hash); i++ {
		c := hash[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// Open opens an immutable artifact for reading after verifying its content hash.
func (s *LocalArtifactStore) Open(ctx context.Context, hash string) (io.ReadCloser, ArtifactRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, ArtifactRef{}, err
	}
	if !isValidHexHash(hash) {
		return nil, ArtifactRef{}, fmt.Errorf("publishing: invalid artifact hash")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.artifactPath(hash)
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ArtifactRef{}, ErrNotFound
		}
		return nil, ArtifactRef{}, fmt.Errorf("publishing: open artifact: %w", err)
	}

	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		_ = file.Close()
		return nil, ArtifactRef{}, fmt.Errorf("publishing: verify artifact: %w", err)
	}

	calculated := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(calculated, hash) {
		_ = file.Close()
		return nil, ArtifactRef{}, ErrArtifactCorrupt
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, ArtifactRef{}, fmt.Errorf("publishing: rewind artifact: %w", err)
	}

	ref := ArtifactRef{
		SHA256: hash,
		Size:   size,
	}

	if metaBytes, err := os.ReadFile(s.metaPath(hash)); err == nil {
		_ = json.Unmarshal(metaBytes, &ref)
	}

	return file, ref, nil
}

// Promote writes an immutable publication manifest linking an artifact to a project release.
func (s *LocalArtifactStore) Promote(ctx context.Context, ref ArtifactRef, relPath string) (PublicationRef, error) {
	if err := ctx.Err(); err != nil {
		return PublicationRef{}, err
	}
	clean := filepath.Clean(filepath.ToSlash(strings.TrimSpace(relPath)))
	if clean == "." || clean == "" || strings.HasPrefix(clean, "../") || clean == ".." || filepath.IsAbs(relPath) {
		return PublicationRef{}, ErrUnsafePath
	}
	if !isValidHexHash(ref.SHA256) {
		return PublicationRef{}, errors.New("publishing: invalid artifact reference for promotion")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	artifactPath := s.artifactPath(ref.SHA256)
	if _, err := os.Stat(artifactPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return PublicationRef{}, ErrNotFound
		}
		return PublicationRef{}, fmt.Errorf("publishing: stat artifact: %w", err)
	}

	targetPath := filepath.Join(s.root, "var", "portfolio", "publications", filepath.FromSlash(clean))
	pubDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(pubDir, 0755); err != nil {
		return PublicationRef{}, fmt.Errorf("publishing: create publication dir: %w", err)
	}

	manifest := PublicationRef{
		Path:           targetPath,
		ArtifactSHA256: ref.SHA256,
	}

	payload, err := json.MarshalIndent(map[string]interface{}{
		"path":            clean,
		"artifact_sha256": ref.SHA256,
		"mime_type":       ref.MIMEType,
		"size":            ref.Size,
	}, "", "  ")
	if err != nil {
		return PublicationRef{}, fmt.Errorf("publishing: marshal publication manifest: %w", err)
	}

	tempFile, err := os.CreateTemp(pubDir, "manifest-*")
	if err != nil {
		return PublicationRef{}, fmt.Errorf("publishing: create temp manifest: %w", err)
	}
	tempName := tempFile.Name()
	defer func() { _ = os.Remove(tempName) }()

	if _, err := io.Copy(tempFile, bytes.NewReader(payload)); err != nil {
		_ = tempFile.Close()
		return PublicationRef{}, err
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return PublicationRef{}, err
	}
	if err := tempFile.Close(); err != nil {
		return PublicationRef{}, err
	}

	if err := os.Rename(tempName, manifest.Path); err != nil {
		return PublicationRef{}, fmt.Errorf("publishing: commit manifest: %w", err)
	}

	return manifest, nil
}

// DeletePreview safely cleans up ephemeral preview artifacts.
func (s *LocalArtifactStore) DeletePreview(ctx context.Context, hash string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !isValidHexHash(hash) {
		return ErrUnsafePath
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return os.RemoveAll(s.previewsDir(hash))
}
