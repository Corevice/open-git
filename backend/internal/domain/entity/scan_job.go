package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	ScanTypeDepend = "dependency"
	ScanTypeSecret = "secret"
)

const (
	ScanStatusQueued     = "queued"
	ScanStatusRunning    = "running"
	ScanStatusCompleted  = "completed"
	ScanStatusFailed     = "scan_failed"
	ScanStatusParseError = "parse_error"
)

type ScanJob struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Type           string
	Status         string
	RetryCount     int
	StartedAt      *time.Time
	FinishedAt     *time.Time
	Error          string
}
