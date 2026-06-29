package entity

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
)

const (
	VisibilityPrivate  = "private"
	VisibilityInternal = "internal"
	VisibilityPublic   = "public"
)

var repositoryNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,100}$`)

type Repository struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	OwnerID        uuid.UUID
	Name           string
	Visibility     string
	DefaultBranch  string
	DiskPath       string `json:"-"`
	CreatedAt      time.Time
}

func (r *Repository) ValidateName() error {
	if !repositoryNameRegex.MatchString(r.Name) {
		return errors.New("invalid repository name")
	}
	return nil
}
