package entity

import (
	"time"

	"github.com/google/uuid"
)

type Milestone struct {
	ID             uuid.UUID
	RepositoryID   uuid.UUID
	OrganizationID uuid.UUID
	Number         int
	Title          string
	Description    string
	State          string
	DueOn          *time.Time
	OpenIssues     int
	ClosedIssues   int
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ClosedAt       *time.Time
}
