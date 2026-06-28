package entity

import (
	"time"

	"github.com/google/uuid"
)

type ImportJobStatus string

const (
	StatusQueued    ImportJobStatus = "queued"
	StatusRunning   ImportJobStatus = "running"
	StatusPaused    ImportJobStatus = "paused"
	StatusCompleted ImportJobStatus = "completed"
	StatusFailed    ImportJobStatus = "failed"
	StatusCancelled ImportJobStatus = "cancelled"
)

type ImportJobPhase string

const (
	PhaseClone        ImportJobPhase = "clone"
	PhaseMetadata     ImportJobPhase = "metadata"
	PhaseIssues       ImportJobPhase = "issues"
	PhasePullRequests ImportJobPhase = "pull_requests"
	PhaseWiki         ImportJobPhase = "wiki"
	PhaseDone         ImportJobPhase = "done"
)

type PhaseProgress struct {
	Done  int
	Total int
}

type ImportProgress map[string]PhaseProgress

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
	GithubLogin       string
	GithubDisplayName string
	LocalUserID       *uuid.UUID
}

type ImportPhaseCheckpoint struct {
	ImportJobID uuid.UUID
	Phase       ImportJobPhase
	LastCursor  string
	Completed   bool
}
