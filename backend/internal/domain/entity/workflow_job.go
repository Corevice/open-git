package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	WorkflowJobStatusQueued     = "queued"
	WorkflowJobStatusInProgress = "in_progress"
	WorkflowJobStatusCompleted  = "completed"
	WorkflowJobStatusFailed     = "failed"
	WorkflowJobStatusCancelled  = "cancelled"

	WorkflowJobConclusionSuccess       = "success"
	WorkflowJobConclusionFailure       = "failure"
	WorkflowJobConclusionSkipped       = "skipped"
	WorkflowJobConclusionCancelled     = "cancelled"
	WorkflowJobConclusionQuotaExceeded = "quota_exceeded"
)

type WorkflowJob struct {
	ID                 uuid.UUID
	WorkflowRunID      *uuid.UUID
	OrganizationID     uuid.UUID
	RepositoryID       uuid.UUID
	Name               string
	RunsOn             []string
	AssignedRunnerID   *uuid.UUID
	AcquireLockVersion int
	Status             string
	Conclusion         string
	StartedAt          *time.Time
	FinishedAt         *time.Time
	TimeoutMinutes     int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
