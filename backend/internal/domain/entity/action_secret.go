package entity

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var actionSecretNamePattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

type ActionSecret struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Name           string
	EncryptedValue string
	KeyID          string
	Visibility     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (s *ActionSecret) Validate() error {
	name := strings.TrimSpace(s.Name)
	if name == "" {
		return errors.New("secret name is required")
	}
	if !actionSecretNamePattern.MatchString(name) {
		return errors.New("secret name must match [A-Z_][A-Z0-9_]*")
	}
	if strings.HasPrefix(name, "GITHUB_") {
		return errors.New("secret names with GITHUB_ prefix are reserved")
	}
	return nil
}
