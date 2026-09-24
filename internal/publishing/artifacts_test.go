package publishing

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactStoreContentAddressesAndDeduplicates(t *testing.T) {
	root := t.TempDir()
	store := NewLocalArtifactStore(root, AssetPolicy{MaxBytes: 1024})
	content := []byte(`{"name":"portfolio"}`)
	metadata := AssetMetadata{Name: "manifest.json", MIMEType: "application/json"}

	first, err := store.Put(context.Background(), bytes.NewReader(content), metadata)
	if err != nil {
		t.Fatalf("first Put: %v", err)
	}
	second, err := store.Put(context.Background(), bytes.NewReader(content), metadata)
	if err != nil {
		t.Fatalf("duplicate Put: %v", err)
	}
	wantHash := sha256.Sum256(content)
	if first.SHA256 != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("SHA256 = %q, want %q", first.SHA256, hex.EncodeToString(wantHash[:]))
	}
	if second != first {
		t.Fatalf("duplicate ref = %#v, want %#v", second, first)
	}

	artifactPath := filepath.Join(root, "var", "portfolio", "artifacts", "sha256", first.SHA256[:2], first.SHA256)
	info, err := os.Stat(artifactPath)
	if err != nil {
		t.Fatalf("stat artifact: %v", err)
	}
	if info.Mode().Perm()&0222 != 0 {
		t.Fatalf("artifact mode = %v, want immutable permissions", info.Mode().Perm())
	}
	entries, err := os.ReadDir(filepath.Dir(artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("artifact count = %d, want one deduplicated file", len(entries))
	}

	opened, openedRef, err := store.Open(context.Background(), first.SHA256)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer opened.Close()
	got, err := io.ReadAll(opened)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) || openedRef != first {
		t.Fatalf("Open = %q, %#v; want %q, %#v", got, openedRef, content, first)
	}
}

func TestArtifactValidationEnforcesMaximumSize(t *testing.T) {
	store := NewLocalArtifactStore(t.TempDir(), AssetPolicy{MaxBytes: 4})
	_, err := store.Put(context.Background(), strings.NewReader("12345"), AssetMetadata{Name: "data.json", MIMEType: "application/json"})
	if !errors.Is(err, ErrAssetTooLarge) {
		t.Fatalf("Put error = %v, want ErrAssetTooLarge", err)
	}
}

func TestArtifactValidationSniffsMIMEAndRejectsMismatch(t *testing.T) {
	store := NewLocalArtifactStore(t.TempDir(), AssetPolicy{MaxBytes: 1024})
	_, err := store.Put(context.Background(), bytes.NewReader([]byte("\x89PNG\r\n\x1a\nnot-really-a-png")), AssetMetadata{Name: "image.png", MIMEType: "image/jpeg"})
	if !errors.Is(err, ErrMIMEMismatch) {
		t.Fatalf("Put error = %v, want ErrMIMEMismatch", err)
	}
}

func TestArtifactValidationRejectsExecutablesAndSVGScript(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		metadata AssetMetadata
	}{
		{name: "ELF executable", content: append([]byte("\x7fELF"), make([]byte, 32)...), metadata: AssetMetadata{Name: "app", MIMEType: "application/octet-stream"}},
		{name: "PE executable", content: append([]byte("MZ"), make([]byte, 32)...), metadata: AssetMetadata{Name: "app.exe", MIMEType: "application/octet-stream"}},
		{name: "scripted SVG", content: []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), metadata: AssetMetadata{Name: "x.svg", MIMEType: "image/svg+xml"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewLocalArtifactStore(t.TempDir(), AssetPolicy{MaxBytes: 1024})
			if _, err := store.Put(context.Background(), bytes.NewReader(tt.content), tt.metadata); !errors.Is(err, ErrAssetTypeNotAllowed) {
				t.Fatalf("Put error = %v, want ErrAssetTypeNotAllowed", err)
			}
		})
	}
}

func TestArtifactValidationRejectsUnsafeArchives(t *testing.T) {
	tests := []struct {
		name string
		zip  []byte
	}{
		{name: "parent traversal", zip: makeZip(t, "../escape.html", 0644)},
		{name: "symlink", zip: makeZip(t, "link", os.ModeSymlink|0644)},
		{name: "scripted SVG in archive", zip: makeZipWithContent(t, "assets/logo.svg", 0644, []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewLocalArtifactStore(t.TempDir(), AssetPolicy{MaxBytes: 4096})
			_, err := store.Put(context.Background(), bytes.NewReader(tt.zip), AssetMetadata{Name: "site.zip", MIMEType: "application/zip"})
			if !errors.Is(err, ErrUnsafeArchive) {
				t.Fatalf("Put error = %v, want ErrUnsafeArchive", err)
			}
		})
	}
}

func TestArtifactStorePromoteRejectsTraversalAndWritesPublication(t *testing.T) {
	root := t.TempDir()
	store := NewLocalArtifactStore(root, AssetPolicy{MaxBytes: 1024})
	ref, err := store.Put(context.Background(), strings.NewReader(`{"ok":true}`), AssetMetadata{Name: "site.json", MIMEType: "application/json"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Promote(context.Background(), ref, "../outside/1.json"); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("Promote traversal error = %v, want ErrUnsafePath", err)
	}
	publication, err := store.Promote(context.Background(), ref, "project-1/2.json")
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	wantPath := filepath.Join(root, "var", "portfolio", "publications", "project-1", "2.json")
	if publication.Path != wantPath {
		t.Fatalf("publication path = %q, want %q", publication.Path, wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("stat publication: %v", err)
	}
}

func TestArtifactStorePromoteRejectsMissingArtifact(t *testing.T) {
	root := t.TempDir()
	store := NewLocalArtifactStore(root, AssetPolicy{MaxBytes: 1024})
	ref := ArtifactRef{
		SHA256: strings.Repeat("a", 64),
	}
	if _, err := store.Promote(context.Background(), ref, "project-1/release.json"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Promote missing artifact error = %v, want ErrNotFound", err)
	}
}

func TestArtifactStoreHashValidation(t *testing.T) {
	root := t.TempDir()
	store := NewLocalArtifactStore(root, AssetPolicy{MaxBytes: 1024})

	invalidHashes := []string{
		"too-short",
		strings.Repeat("a", 63),
		strings.Repeat("a", 65),
		strings.Repeat("A", 64), // uppercase not allowed
		strings.Repeat("g", 64), // non-hex
		"../" + strings.Repeat("a", 61),
	}

	for _, invalid := range invalidHashes {
		if _, _, err := store.Open(context.Background(), invalid); err == nil {
			t.Errorf("Open(%q) expected error, got nil", invalid)
		}

		ref := ArtifactRef{SHA256: invalid}
		if _, err := store.Promote(context.Background(), ref, "project-1/release.json"); err == nil {
			t.Errorf("Promote(%q) expected error, got nil", invalid)
		}
	}
}

func TestArtifactStoreDeletePreview(t *testing.T) {
	root := t.TempDir()
	store := NewLocalArtifactStore(root, AssetPolicy{MaxBytes: 1024})

	// Traversal and invalid hash rejection
	invalidHashes := []string{
		"",
		"../escape",
		"../../etc",
		strings.Repeat("A", 64),
		strings.Repeat("g", 64),
		"short",
	}
	for _, invalid := range invalidHashes {
		err := store.DeletePreview(context.Background(), invalid)
		if !errors.Is(err, ErrUnsafePath) {
			t.Errorf("DeletePreview(%q) error = %v, want ErrUnsafePath", invalid, err)
		}
	}

	// Valid preview cleanup
	validHash := strings.Repeat("b", 64)
	previewDir := filepath.Join(root, "var", "portfolio", "previews", validHash)
	if err := os.MkdirAll(previewDir, 0755); err != nil {
		t.Fatal(err)
	}
	sampleFile := filepath.Join(previewDir, "index.html")
	if err := os.WriteFile(sampleFile, []byte("<h1>preview</h1>"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(previewDir); err != nil {
		t.Fatalf("preview dir should exist: %v", err)
	}

	if err := store.DeletePreview(context.Background(), validHash); err != nil {
		t.Fatalf("DeletePreview failed: %v", err)
	}

	if _, err := os.Stat(previewDir); !os.IsNotExist(err) {
		t.Fatalf("preview dir should be removed, got err = %v", err)
	}

	// Idempotent deletion on non-existent preview
	if err := store.DeletePreview(context.Background(), validHash); err != nil {
		t.Fatalf("subsequent DeletePreview should succeed, got %v", err)
	}
}

func makeZip(t *testing.T, name string, mode os.FileMode) []byte {
	return makeZipWithContent(t, name, mode, []byte("content"))
}

func makeZipWithContent(t *testing.T, name string, mode os.FileMode, content []byte) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	header := &zip.FileHeader{Name: name, Method: zip.Store}
	header.SetMode(mode)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
