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
	RepositoryID   uuid.UUID
	WorkflowID     uuid.UUID
	HeadSHA        string
	HeadBranch     string
	Workflow       string
	Event          string
	ActorLogin     string
	RunNumber      int
	Status         string
	Conclusion     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
