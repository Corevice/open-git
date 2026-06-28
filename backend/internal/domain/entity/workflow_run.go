package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	WorkflowStatusCompleted   = "completed"
	WorkflowConclusionSuccess = "success"
)

type WorkflowRun struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	WorkflowID     uuid.UUID
	Workflow       string
	RunNumber      int
	Event          string
	HeadBranch     string
	HeadSHA        string
	Status         string
	Conclusion     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
