package entity

import "time"

const (
	WorkflowJobStatusQueued     = "queued"
	WorkflowJobStatusInProgress = "in_progress"
	WorkflowJobStatusCompleted  = "completed"
	WorkflowJobStatusFailed     = "failed"
)

type WorkflowJob struct {
	ID             string
	WorkflowRunID  string
	OrganizationID string
	RepositoryID   string
	Name           string
	Status         string
	Conclusion     string
	CreatedAt      time.Time
	CompletedAt    *time.Time
}
