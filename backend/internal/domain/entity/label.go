package entity

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var labelColorPattern = regexp.MustCompile("^[0-9a-fA-F]{6}$")

type Label struct {
	ID             uuid.UUID
	RepositoryID   uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Color          string
	Description    string
	CreatedAt      time.Time
}

func (l *Label) ValidateColor() error {
	if !labelColorPattern.MatchString(l.Color) {
		return errors.New("invalid color")
	}
	return nil
}
