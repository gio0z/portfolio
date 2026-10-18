package preview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadOutputDir_RejectsSymlinks(t *testing.T) {
	tmp := t.TempDir()

	secretFile := filepath.Join(tmp, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("super-secret-host-data"), 0o600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	outDir := filepath.Join(tmp, "output")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("failed to create outDir: %v", err)
	}

	validFile := filepath.Join(outDir, "index.html")
	if err := os.WriteFile(validFile, []byte("<h1>Hello</h1>"), 0o644); err != nil {
		t.Fatalf("failed to write index.html: %v", err)
	}

	symlinkPath := filepath.Join(outDir, "leak.txt")
	if err := os.Symlink(secretFile, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	files, err := readOutputDir(outDir, 1024*1024)
	if err == nil {
		t.Fatalf("expected error for symlink, got nil")
	}

	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected error mentioning 'symlink', got %v", err)
	}

	for k, v := range files {
		if strings.Contains(string(v), "super-secret-host-data") {
			t.Fatalf("leak detected in files[%q]: host secret bytes returned", k)
		}
	}
}

func TestReadOutputDir_NestedTreeAndMaxBytes(t *testing.T) {
	outDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(outDir, "sub", "deep"), 0o755); err != nil {
		t.Fatalf("failed to create nested dirs: %v", err)
	}

	file1 := filepath.Join(outDir, "root.txt")
	file2 := filepath.Join(outDir, "sub", "file2.txt")
	file3 := filepath.Join(outDir, "sub", "deep", "file3.txt")

	if err := os.WriteFile(file1, []byte("12345"), 0o644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("67890"), 0o644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}
	if err := os.WriteFile(file3, []byte("abcde"), 0o644); err != nil {
		t.Fatalf("failed to write file3: %v", err)
	}

	files, err := readOutputDir(outDir, 100)
	if err != nil {
		t.Fatalf("readOutputDir failed: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	if string(files["root.txt"]) != "12345" {
		t.Errorf("expected root.txt to be '12345', got %q", string(files["root.txt"]))
	}
	if string(files["sub/file2.txt"]) != "67890" {
		t.Errorf("expected sub/file2.txt to be '67890', got %q", string(files["sub/file2.txt"]))
	}
	if string(files["sub/deep/file3.txt"]) != "abcde" {
		t.Errorf("expected sub/deep/file3.txt to be 'abcde', got %q", string(files["sub/deep/file3.txt"]))
	}

	_, err = readOutputDir(outDir, 10)
	if err == nil {
		t.Fatalf("expected maxBytes error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds 10 bytes") {
		t.Fatalf("expected error mentioning maxBytes limit, got %v", err)
	}
}
