package entity

import (
	"errors"
	"unicode/utf8"

	"github.com/google/uuid"
)

type Issue struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Number         int
	Title          string
	Body           string
	State          string
	AuthorID       uuid.UUID
}

func (i *Issue) ValidateTitle() error {
	length := utf8.RuneCountInString(i.Title)
	if length < 1 || length > 256 {
		return errors.New("invalid title")
	}
	return nil
}
