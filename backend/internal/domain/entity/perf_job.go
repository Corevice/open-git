package entity

import (
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	JobQueued    JobStatus = "queued"
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
	JobTimeout   JobStatus = "timeout"
)

type PerfJob struct {
	ID          uuid.UUID
	Status      JobStatus
	TriggeredBy *uuid.UUID
	BenchmarkID *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
