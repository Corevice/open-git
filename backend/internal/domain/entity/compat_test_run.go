package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	CompatStatusQueued    = "queued"
	CompatStatusRunning   = "running"
	CompatStatusCompleted = "completed"
	CompatStatusFailed    = "failed"
)

type CompatTestRun struct {
	ID              uuid.UUID
	Suite           string
	Status          string
	TriggeredBy     *uuid.UUID
	OrganizationID  uuid.UUID
	TotalEndpoints  int
	Passing         int
	Failing         int
	Unimplemented   int
	CoverageRate    float64
	StartedAt       *time.Time
	FinishedAt      *time.Time
	CreatedAt       time.Time
}
