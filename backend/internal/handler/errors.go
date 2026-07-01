package handler

import (
	"errors"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
)

const (
	CodeInvalidRequest       = "invalid_request"
	CodeUnauthorized         = "unauthorized"
	CodeForbidden            = "forbidden"
	CodeNotFound             = "not_found"
	CodeConflict             = "conflict"
	CodeGone                 = "gone"
	CodeUnsupportedMediaType = "unsupported_media_type"
	CodeValidationFailed     = "validation_failed"
	CodeRateLimited          = "rate_limited"
	CodeInternal             = "internal_error"
	CodeServiceUnavailable   = "service_unavailable"
)

// DomainErrorToHTTP maps sentinel errors from both the domain layer
// (domain.Err*) and the application layer (apperror.Err*) to HTTP status
// codes. The central error handler relies on this for any handler that does
// not translate these sentinels itself; anything unrecognized becomes a 500.
func DomainErrorToHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, apperror.ErrNotFound):
		return 404, CodeNotFound
	case errors.Is(err, domain.ErrConflict),
		errors.Is(err, apperror.ErrConflict),
		errors.Is(err, apperror.ErrAlreadyMerged):
		return 409, CodeConflict
	case errors.Is(err, domain.ErrValidation),
		errors.Is(err, apperror.ErrValidation),
		errors.Is(err, apperror.ErrProtectionNotSatisfied):
		return 422, CodeValidationFailed
	case errors.Is(err, apperror.ErrGone):
		return 410, CodeGone
	case errors.Is(err, domain.ErrUnauthorized):
		return 401, CodeUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		return 403, CodeForbidden
	default:
		return 500, CodeInternal
	}
}
