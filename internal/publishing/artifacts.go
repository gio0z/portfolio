package publishing

import (
	"context"
	"errors"
	"io"
)

var (
	ErrAssetTooLarge       = errors.New("publishing: asset exceeds maximum size")
	ErrAssetTypeNotAllowed = errors.New("publishing: asset type is not allowed")
	ErrMIMEMismatch        = errors.New("publishing: declared MIME type does not match content")
	ErrUnsafeArchive       = errors.New("publishing: archive contains an unsafe entry")
	ErrUnsafePath          = errors.New("publishing: unsafe path")
	ErrArtifactCorrupt     = errors.New("publishing: artifact content does not match its hash")
)

// AssetMetadata describes an asset supplied for immutable storage.
type AssetMetadata struct {
	Name     string `json:"name"`
	MIMEType string `json:"mime_type"`
}

// AssetPolicy constrains accepted asset content. A zero MaxBytes uses a safe
// development default.
type AssetPolicy struct {
	MaxBytes int64
}

// PublicationRef identifies a promoted publication manifest.
type PublicationRef struct {
	Path           string `json:"path"`
	ArtifactSHA256 string `json:"artifact_sha256"`
}

// ValidatedAsset is the result of validating and hashing an asset stream.
type ValidatedAsset struct {
	SHA256   string
	MIMEType string
	Size     int64
}

// ArtifactStore persists immutable reviewed content and promotion manifests.
type ArtifactStore interface {
	Put(context.Context, io.Reader, AssetMetadata) (ArtifactRef, error)
	Open(context.Context, string) (io.ReadCloser, ArtifactRef, error)
	Promote(context.Context, ArtifactRef, string) (PublicationRef, error)
	DeletePreview(context.Context, string) error
}
