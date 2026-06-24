package apperror

import "errors"

var (
	ErrValidation             = errors.New("validation error")
	ErrGone                   = errors.New("gone")
	ErrNotFound               = errors.New("not found")
	ErrAlreadyMerged          = errors.New("already merged")
	ErrConflict               = errors.New("conflict")
	ErrProtectionNotSatisfied = errors.New("branch protection not satisfied")
)
