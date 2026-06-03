package entity

import (
	"time"

	"github.com/google/uuid"
)

type WorkflowRun struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Workflow       string
	Status         string
	Conclusion     string
	HeadSHA        string
	StartedAt      time.Time
	CompletedAt    *time.Time
}
