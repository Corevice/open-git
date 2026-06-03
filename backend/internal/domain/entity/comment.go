package entity

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	IssueID        uuid.UUID
	AuthorID       uuid.UUID
	Body           string
	CreatedAt      time.Time
}
