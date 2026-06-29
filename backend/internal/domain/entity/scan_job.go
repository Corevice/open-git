package entity

import (
	"time"

	"github.com/google/uuid"
)

type ScanJobType string

const (
	ScanJobTypeDependency ScanJobType = "dependency"
	ScanJobTypeSecret     ScanJobType = "secret"
)

type ScanJobStatus string

const (
	ScanJobStatusQueued     ScanJobStatus = "queued"
	ScanJobStatusRunning    ScanJobStatus = "running"
	ScanJobStatusCompleted  ScanJobStatus = "completed"
	ScanJobStatusScanFailed ScanJobStatus = "scan_failed"
	ScanJobStatusParseError ScanJobStatus = "parse_error"
)

type ScanJob struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Type           ScanJobType
	Status         ScanJobStatus
	RetryCount     int
	StartedAt      *time.Time
	FinishedAt     *time.Time
	Error          string
}
