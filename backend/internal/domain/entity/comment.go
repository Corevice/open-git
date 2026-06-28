package entity

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID             uuid.UUID
	IssueID        uuid.UUID
	AuthorID       uuid.UUID
	OrganizationID uuid.UUID
	Body           string
	CreatedAt      time.Time
}
