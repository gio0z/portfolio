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
		{name: "symlink", zip: makeZip(t, "link", os.ModeSymlink|0777)},
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

func makeZip(t *testing.T, name string, mode os.FileMode) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	header := &zip.FileHeader{Name: name, Method: zip.Store}
	header.SetMode(mode)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("content")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
