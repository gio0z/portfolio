package publishing

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
)

const defaultMaxAssetBytes int64 = 32 << 20

var allowedMIMETypes = map[string]struct{}{
	"image/png":              {},
	"image/jpeg":             {},
	"image/webp":             {},
	"video/mp4":              {},
	"application/json":       {},
	"text/html":              {},
	"text/css":               {},
	"text/javascript":        {},
	"application/javascript": {},
	"application/zip":        {},
}

// ValidateAndHash validates an asset as it is copied to dst and returns its
// content identity. The source is read at most MaxBytes+1 bytes.
func ValidateAndHash(ctx context.Context, dst io.Writer, src io.Reader, metadata AssetMetadata, policy AssetPolicy) (ValidatedAsset, error) {
	if err := ctx.Err(); err != nil {
		return ValidatedAsset{}, err
	}
	if dst == nil || src == nil {
		return ValidatedAsset{}, fmt.Errorf("publishing: asset reader and writer are required")
	}
	maxBytes := policy.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxAssetBytes
	}
	content, err := io.ReadAll(io.LimitReader(src, maxBytes+1))
	if err != nil {
		return ValidatedAsset{}, fmt.Errorf("publishing: read asset: %w", err)
	}
	if int64(len(content)) > maxBytes {
		return ValidatedAsset{}, ErrAssetTooLarge
	}

	detected := detectMIME(content, metadata.Name)
	declared := canonicalMIME(metadata.MIMEType)
	if _, allowed := allowedMIMETypes[detected]; !allowed {
		return ValidatedAsset{}, fmt.Errorf("%w: %s", ErrAssetTypeNotAllowed, detected)
	}
	if declared == "" || !mimeMatches(declared, detected) {
		return ValidatedAsset{}, fmt.Errorf("%w: declared %q, detected %q", ErrMIMEMismatch, declared, detected)
	}
	if isExecutable(content) {
		return ValidatedAsset{}, ErrAssetTypeNotAllowed
	}
	if detected == "application/zip" {
		if err := validateStaticArchive(content); err != nil {
			return ValidatedAsset{}, err
		}
	}

	hash := sha256.Sum256(content)
	if _, err := dst.Write(content); err != nil {
		return ValidatedAsset{}, fmt.Errorf("publishing: write validated asset: %w", err)
	}
	return ValidatedAsset{SHA256: hex.EncodeToString(hash[:]), MIMEType: detected, Size: int64(len(content))}, nil
}

func detectMIME(content []byte, name string) string {
	detected := canonicalMIME(http.DetectContentType(content))
	ext := strings.ToLower(path.Ext(name))

	// DetectContentType intentionally labels textual web assets as text/plain.
	if detected == "text/plain" {
		switch ext {
		case ".json":
			if json.Valid(content) {
				return "application/json"
			}
		case ".css":
			return "text/css"
		case ".js", ".mjs":
			return "text/javascript"
		}
	}
	if ext == ".html" || ext == ".htm" {
		trimmed := strings.ToLower(strings.TrimSpace(string(content)))
		if strings.HasPrefix(trimmed, "<!doctype html") || strings.HasPrefix(trimmed, "<html") {
			return "text/html"
		}
	}
	return detected
}

func canonicalMIME(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return value
	}
	return mediaType
}

func mimeMatches(declared, detected string) bool {
	if declared == detected {
		return true
	}
	return (declared == "application/javascript" && detected == "text/javascript") ||
		(declared == "text/javascript" && detected == "application/javascript")
}

func isExecutable(content []byte) bool {
	return bytes.HasPrefix(content, []byte("\x7fELF")) ||
		bytes.HasPrefix(content, []byte("MZ")) ||
		bytes.HasPrefix(content, []byte("\xfe\xed\xfa\xce")) ||
		bytes.HasPrefix(content, []byte("\xfe\xed\xfa\xcf")) ||
		bytes.HasPrefix(content, []byte("\xcf\xfa\xed\xfe")) ||
		bytes.HasPrefix(content, []byte("\xca\xfe\xba\xbe"))
}

func validateStaticArchive(content []byte) error {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return fmt.Errorf("%w: invalid zip: %v", ErrUnsafeArchive, err)
	}
	for _, file := range reader.File {
		name := strings.ReplaceAll(file.Name, "\\", "/")
		clean := path.Clean(name)
		if name == "" || strings.HasPrefix(name, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
			return fmt.Errorf("%w: path %q", ErrUnsafeArchive, file.Name)
		}
		if file.Mode()&os.ModeSymlink != 0 || file.Mode()&0o111 != 0 {
			return fmt.Errorf("%w: non-static entry %q", ErrUnsafeArchive, file.Name)
		}
		if file.FileInfo().IsDir() {
			continue
		}
		entry, err := file.Open()
		if err != nil {
			return fmt.Errorf("%w: open %q: %v", ErrUnsafeArchive, file.Name, err)
		}
		entryContent, readErr := io.ReadAll(io.LimitReader(entry, defaultMaxAssetBytes+1))
		closeErr := entry.Close()
		if readErr != nil || closeErr != nil || int64(len(entryContent)) > defaultMaxAssetBytes || isExecutable(entryContent) {
			return fmt.Errorf("%w: invalid entry %q", ErrUnsafeArchive, file.Name)
		}
		lower := bytes.ToLower(entryContent)
		if strings.EqualFold(path.Ext(clean), ".svg") && (bytes.Contains(lower, []byte("<script")) || bytes.Contains(lower, []byte("javascript:")) || bytes.Contains(lower, []byte("onload="))) {
			return fmt.Errorf("%w: scripted SVG %q", ErrUnsafeArchive, file.Name)
		}
	}
	return nil
}
