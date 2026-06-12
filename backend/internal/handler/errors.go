package handler

import (
	"errors"

	"github.com/open-git/backend/internal/domain"
)

const (
	CodeInvalidRequest         = "invalid_request"
	CodeUnauthorized           = "unauthorized"
	CodeForbidden              = "forbidden"
	CodeNotFound               = "not_found"
	CodeConflict               = "conflict"
	CodeUnsupportedMediaType   = "unsupported_media_type"
	CodeValidationFailed       = "validation_failed"
	CodeRateLimited            = "rate_limited"
	CodeInternal               = "internal_error"
	CodeServiceUnavailable     = "service_unavailable"
)

func DomainErrorToHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return 404, CodeNotFound
	case errors.Is(err, domain.ErrConflict):
		return 409, CodeConflict
	case errors.Is(err, domain.ErrValidation):
		return 422, CodeValidationFailed
	case errors.Is(err, domain.ErrUnauthorized):
		return 401, CodeUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		return 403, CodeForbidden
	default:
		return 500, CodeInternal
	}
}
