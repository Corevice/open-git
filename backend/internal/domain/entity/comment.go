package entity

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID             uuid.UUID
	IssueID        uuid.UUID
	OrganizationID uuid.UUID
	AuthorID       uuid.UUID
	AuthorLogin    string
	Body           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
