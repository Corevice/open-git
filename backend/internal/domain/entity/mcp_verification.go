package entity

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type RunStatus string

const (
	RunStatusQueued    RunStatus = "queued"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusErrored   RunStatus = "errored"
)

type OverallStatus string

const (
	OverallStatusCompatible   OverallStatus = "compatible"
	OverallStatusPartial      OverallStatus = "partial"
	OverallStatusIncompatible OverallStatus = "incompatible"
)

type CheckStatus string

const (
	CheckStatusPass CheckStatus = "pass"
	CheckStatusFail CheckStatus = "fail"
	CheckStatusSkip CheckStatus = "skip"
)

type CheckCategory string

const (
	CheckCategoryGraphQL CheckCategory = "graphql"
	CheckCategoryREST    CheckCategory = "rest"
	CheckCategoryAuth    CheckCategory = "auth"
)

type MCPVerificationRun struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	RepositoryID       *uuid.UUID
	RepositoryFullName   string
	TriggeredBy        *uuid.UUID
	Status             RunStatus
	OverallStatus      *OverallStatus
	Targets            json.RawMessage
	StartedAt          *time.Time
	FinishedAt         *time.Time
	CreatedAt          time.Time
}

type MCPVerificationCheck struct {
	ID             uuid.UUID
	RunID          uuid.UUID
	OrganizationID uuid.UUID
	CheckID        string
	Category       CheckCategory
	Status         CheckStatus
	Expected       json.RawMessage
	Actual         json.RawMessage
	Error          *string
	DurationMS     int
	CreatedAt      time.Time
}

func (r *MCPVerificationRun) Validate() error {
	if r.RepositoryFullName == "" {
		return errors.New("repository is required")
	}
	return nil
}

func ComputeOverallStatus(checks []*MCPVerificationCheck) OverallStatus {
	if len(checks) == 0 {
		return OverallStatusCompatible
	}

	hasSkip := false
	for _, check := range checks {
		if check == nil {
			continue
		}
		switch check.Status {
		case CheckStatusFail:
			return OverallStatusIncompatible
		case CheckStatusSkip:
			hasSkip = true
		}
	}

	if hasSkip {
		return OverallStatusPartial
	}

	return OverallStatusCompatible
}
