package entity

import (
	"time"

	"github.com/google/uuid"
)

// Comment is a comment authored on an issue (or pull request).
type Comment struct {
	ID        uuid.UUID
	IssueID   uuid.UUID
	AuthorID  uuid.UUID
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
