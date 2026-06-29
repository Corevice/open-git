package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusPass     = "pass"
	StatusPartial  = "partial"
	StatusFail     = "fail"
	StatusUntested = "untested"
	StatusError    = "error"
)

type ActionCompatibilityResult struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RepositoryID   *uuid.UUID
	ActionName     string
	ActionVersion  string
	Status         string
	Note           *string
	GoldenDiff     map[string]any
	VerifiedAt     *time.Time
	VerificationID uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

const (
	TriggerManual    = "manual"
	TriggerScheduled = "scheduled"
	TriggerPush      = "push"

	VerifStatusQueued    = "queued"
	VerifStatusRunning   = "running"
	VerifStatusCompleted = "completed"
	VerifStatusFailed    = "failed"
)

type ActionVerification struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Trigger        string
	Status         string
	RequestedBy    *uuid.UUID
	StartedAt      *time.Time
	FinishedAt     *time.Time
	CreatedAt      time.Time
}

type ActionCacheEntry struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ActionName     string
	ResolvedRef    string
	StoragePath    string
	CachedAt       time.Time
}
