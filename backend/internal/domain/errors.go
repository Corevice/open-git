package domain

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrValidation   = errors.New("validation failed")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrInternal     = errors.New("internal error")
)

type DomainError struct {
	Code    string
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Message != "" {
		return e.Code + ": " + e.Message
	}
	if e.Err != nil {
		return e.Code + ": " + e.Err.Error()
	}
	return e.Code
}

func (e *DomainError) Unwrap() error {
	return e.Err
}
