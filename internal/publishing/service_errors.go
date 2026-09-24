package publishing

import (
	"errors"
	"fmt"
)

// Stable error codes returned by the publishing service.
const (
	ErrCodeValidationFailed  = "validation_failed"
	ErrCodeForbidden         = "forbidden"
	ErrCodeInvalidState      = "invalid_state"
	ErrCodeArtifactMismatch  = "artifact_mismatch"
	ErrCodeScanFailed        = "scan_failed"
	ErrCodeSlugConflict      = "slug_conflict"
	ErrCodePublicationFailed = "publication_failed"
	ErrCodeNotFound          = "not_found"
)

// ServiceError represents a structured, domain-specific publishing error.
type ServiceError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("publishing [%s]: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("publishing [%s]: %s", e.Code, e.Message)
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}

// ErrorCode returns the stable error code from err if it is or wraps a ServiceError,
// or maps standard repository and storage errors to stable codes.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var se *ServiceError
	if errors.As(err, &se) {
		return se.Code
	}
	if errors.Is(err, ErrConflict) || errors.Is(err, ErrDuplicateIdempotencyKey) {
		return ErrCodeSlugConflict
	}
	if errors.Is(err, ErrNotFound) {
		return ErrCodeNotFound
	}
	if errors.Is(err, ErrAssetTooLarge) || errors.Is(err, ErrAssetTypeNotAllowed) ||
		errors.Is(err, ErrMIMEMismatch) || errors.Is(err, ErrUnsafeArchive) ||
		errors.Is(err, ErrUnsafePath) {
		return ErrCodeValidationFailed
	}
	return ""
}

func NewValidationError(msg string, err error) error {
	return &ServiceError{Code: ErrCodeValidationFailed, Message: msg, Err: err}
}

func NewForbiddenError(msg string) error {
	return &ServiceError{Code: ErrCodeForbidden, Message: msg}
}

func NewInvalidStateError(msg string, err error) error {
	return &ServiceError{Code: ErrCodeInvalidState, Message: msg, Err: err}
}

func NewArtifactMismatchError(msg string) error {
	return &ServiceError{Code: ErrCodeArtifactMismatch, Message: msg}
}

func NewScanFailedError(msg string) error {
	return &ServiceError{Code: ErrCodeScanFailed, Message: msg}
}

func NewSlugConflictError(msg string, err error) error {
	return &ServiceError{Code: ErrCodeSlugConflict, Message: msg, Err: err}
}

func NewPublicationFailedError(msg string, err error) error {
	return &ServiceError{Code: ErrCodePublicationFailed, Message: msg, Err: err}
}

func NewNotFoundError(msg string, err error) error {
	return &ServiceError{Code: ErrCodeNotFound, Message: msg, Err: err}
}
