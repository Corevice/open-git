package entity

import (
	"time"

	"github.com/google/uuid"
)

type ImportJobStatus string

const (
	ImportJobStatusQueued    ImportJobStatus = "queued"
	ImportJobStatusRunning   ImportJobStatus = "running"
	ImportJobStatusPaused    ImportJobStatus = "paused"
	ImportJobStatusCompleted ImportJobStatus = "completed"
	ImportJobStatusFailed    ImportJobStatus = "failed"
	ImportJobStatusCancelled ImportJobStatus = "cancelled"
)

type ImportJobPhase string

const (
	ImportJobPhaseClone        ImportJobPhase = "clone"
	ImportJobPhaseMetadata     ImportJobPhase = "metadata"
	ImportJobPhaseIssues       ImportJobPhase = "issues"
	ImportJobPhasePullRequests ImportJobPhase = "pull_requests"
	ImportJobPhaseWiki         ImportJobPhase = "wiki"
	ImportJobPhaseDone         ImportJobPhase = "done"
)

type ImportPhaseProgress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

type ImportProgress map[string]ImportPhaseProgress

type ImportJob struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	CreatedBy          uuid.UUID
	SourceURL          string
	TargetRepositoryID *uuid.UUID
	TargetName         string
	Include            []string
	Status             ImportJobStatus
	Phase              ImportJobPhase
	Progress           ImportProgress
	TokenSecretRef     *string
	Error              *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ImportUserMapping struct {
	ID                uuid.UUID
	ImportJobID       uuid.UUID
	GitHubLogin       string
	GitHubDisplayName string
	LocalUserID       *uuid.UUID
}

type ImportPhaseCheckpoint struct {
	ImportJobID uuid.UUID
	Phase       ImportJobPhase
	LastCursor  string
	Completed   bool
}
