package entity

import (
	"time"

	"github.com/google/uuid"
)

type Workflow struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Name           string
	Path           string
	State          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type WorkflowStep struct {
	ID          uuid.UUID
	JobID       uuid.UUID
	Number      int
	Name        string
	Status      string
	Conclusion  string
	StartedAt   *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
}
