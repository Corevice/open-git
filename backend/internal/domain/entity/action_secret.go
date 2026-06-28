package entity

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
)

type SecretVisibility string

const (
	VisibilityAll      SecretVisibility = "all"
	VisibilityPrivate  SecretVisibility = "private"
	VisibilitySelected SecretVisibility = "selected"

	MaxActionSecretNameLength = 255
)

var actionSecretNamePattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

type ActionSecret struct {
	ID                    uuid.UUID
	OrganizationID        uuid.UUID
	RepositoryID          *uuid.UUID
	Name                  string
	EncryptedValue        []byte
	KeyID                 string
	Visibility            SecretVisibility
	SelectedRepositoryIDs []uuid.UUID
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (s *ActionSecret) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("%w: name is required", apperror.ErrValidation)
	}
	if len(s.Name) > MaxActionSecretNameLength {
		return fmt.Errorf("%w: name exceeds maximum length", apperror.ErrValidation)
	}
	if !actionSecretNamePattern.MatchString(s.Name) {
		return fmt.Errorf("%w: name must match naming convention", apperror.ErrValidation)
	}
	if strings.HasPrefix(s.Name, "GITHUB_") {
		return fmt.Errorf("%w: GITHUB_ prefix is reserved", apperror.ErrValidation)
	}
	switch s.Visibility {
	case VisibilityAll, VisibilityPrivate:
	case VisibilitySelected:
		if len(s.SelectedRepositoryIDs) == 0 {
			return fmt.Errorf("%w: selected visibility requires at least one repository", apperror.ErrValidation)
		}
	default:
		return fmt.Errorf("%w: invalid visibility", apperror.ErrValidation)
	}
	return nil
}
