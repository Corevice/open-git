package importjob

import "errors"

var (
	ErrInvalidSourceURL   = errors.New("invalid source url")
	ErrTargetNameConflict = errors.New("target name conflict")
	ErrNotFound           = errors.New("not found")
	ErrInvalidTransition  = errors.New("invalid transition")
	ErrForbidden          = errors.New("forbidden")
)
