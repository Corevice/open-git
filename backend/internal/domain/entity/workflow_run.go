package entity

import (
	"time"

	"github.com/google/uuid"
)

// WorkflowRun represents a single execution of a workflow for a commit.
type WorkflowRun struct {
	ID           uuid.UUID
	RepositoryID uuid.UUID
	Workflow     string
	HeadSHA      string
	Status       string
	Conclusion   string
	CreatedAt    time.Time
}
