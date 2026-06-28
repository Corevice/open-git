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
	HeadSHA        string
	Workflow       string
	Status         string
	Conclusion     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
