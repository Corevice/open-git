// Package apperror defines sentinel application errors shared across the
// usecase and handler layers. Handlers map these to HTTP status codes.
package apperror

import "errors"

var (
	// ErrValidation indicates the request failed validation (HTTP 422).
	ErrValidation = errors.New("validation failed")
	// ErrNotFound indicates the requested resource does not exist (HTTP 404).
	ErrNotFound = errors.New("not found")
	// ErrConflict indicates a conflicting state, e.g. a merge conflict (HTTP 409).
	ErrConflict = errors.New("conflict")
	// ErrUnauthorized indicates the caller is not authenticated (HTTP 401).
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden indicates the caller lacks permission (HTTP 403).
	ErrForbidden = errors.New("forbidden")
	// ErrGone indicates the resource is no longer available (HTTP 410).
	ErrGone = errors.New("gone")
	// ErrAlreadyMerged indicates a pull request has already been merged (HTTP 405).
	ErrAlreadyMerged = errors.New("already merged")
	// ErrProtectionNotSatisfied indicates branch protection rules are unmet (HTTP 405).
	ErrProtectionNotSatisfied = errors.New("branch protection not satisfied")
	// ErrInternal indicates an unexpected internal error (HTTP 500).
	ErrInternal = errors.New("internal error")
)
