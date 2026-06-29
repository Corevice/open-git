package entity

import (
	"time"

	"github.com/google/uuid"
)

type WorkflowState string

const (
	WorkflowStateActive   WorkflowState = "active"
	WorkflowStateDisabled WorkflowState = "disabled"
)

type ParseStatus string

const (
	ParseStatusValid   ParseStatus = "valid"
	ParseStatusInvalid ParseStatus = "invalid"
	ParseStatusPending ParseStatus = "pending"
)

type DiagnosticSeverity string

const (
	SeverityError   DiagnosticSeverity = "error"
	SeverityWarning DiagnosticSeverity = "warning"
	SeverityInfo    DiagnosticSeverity = "info"
)

type Workflow struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Name           string
	Path           string
	State          WorkflowState
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type WorkflowRevision struct {
	ID             uuid.UUID
	WorkflowID     uuid.UUID
	CommitSHA      string
	RawContentHash string
	ParseStatus    ParseStatus
	IR             string
	ParsedAt       *time.Time
}

type WorkflowDiagnostic struct {
	ID                 uuid.UUID
	WorkflowRevisionID uuid.UUID
	Line               int
	Col                int
	Severity           DiagnosticSeverity
	Message            string
}
