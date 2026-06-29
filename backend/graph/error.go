package graph

import (
	"context"
	"errors"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
)

const (
	errorTypeNotFound              = "NOT_FOUND"
	errorTypeForbidden             = "FORBIDDEN"
	errorTypeUnprocessable         = "UNPROCESSABLE"
	errorTypeInsufficientScopes    = "INSUFFICIENT_SCOPES"
	errorTypeMaxNodeLimitExceeded  = "MAX_NODE_LIMIT_EXCEEDED"
)

var ErrInsufficientScopes = errors.New("insufficient scopes")

func GitHubErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	gqlErr := graphql.DefaultErrorPresenter(ctx, err)
	gqlErr.Message = sanitizeErrorMessage(gqlErr.Message)

	githubType := mapErrorType(err)
	if githubType == "" {
		githubType = inferErrorType(gqlErr.Message)
	}

	return &gqlerror.Error{
		Message:    gqlErr.Message,
		Path:       gqlErr.Path,
		Locations:  gqlErr.Locations,
		Extensions: map[string]any{"type": githubType},
	}
}

func mapErrorType(err error) string {
	switch {
	case errors.Is(err, apperror.ErrNotFound), errors.Is(err, domain.ErrNotFound):
		return errorTypeNotFound
	case errors.Is(err, domain.ErrForbidden):
		return errorTypeForbidden
	case errors.Is(err, domain.ErrUnauthorized):
		return errorTypeForbidden
	case errors.Is(err, apperror.ErrValidation), errors.Is(err, domain.ErrValidation):
		return errorTypeUnprocessable
	case errors.Is(err, ErrInsufficientScopes):
		return errorTypeInsufficientScopes
	default:
		var depthErr *domainDepthError
		if errors.As(err, &depthErr) {
			return errorTypeMaxNodeLimitExceeded
		}
		var domainErr *domain.DomainError
		if errors.As(err, &domainErr) {
			switch strings.ToUpper(domainErr.Code) {
			case errorTypeNotFound:
				return errorTypeNotFound
			case errorTypeForbidden:
				return errorTypeForbidden
			case errorTypeUnprocessable:
				return errorTypeUnprocessable
			case errorTypeInsufficientScopes:
				return errorTypeInsufficientScopes
			case errorTypeMaxNodeLimitExceeded:
				return errorTypeMaxNodeLimitExceeded
			}
		}
	}
	return ""
}

func inferErrorType(message string) string {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "complexity"):
		return errorTypeMaxNodeLimitExceeded
	case strings.Contains(lower, "depth limit"):
		return errorTypeMaxNodeLimitExceeded
	case strings.Contains(lower, "introspection"):
		return errorTypeForbidden
	case strings.Contains(lower, "unauthorized"), strings.Contains(lower, "forbidden"):
		return errorTypeForbidden
	default:
		return errorTypeUnprocessable
	}
}

func sanitizeErrorMessage(message string) string {
	lower := strings.ToLower(message)
	if strings.Contains(lower, "bearer ") || strings.Contains(lower, "token ") {
		return "request could not be processed"
	}
	return message
}
