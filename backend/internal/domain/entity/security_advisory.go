package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
)

const (
	StateOpen         = "open"
	StateAcknowledged = "acknowledged"
	StateResolved     = "resolved"
	StateDismissed    = "dismissed"
)

const (
	DismissReasonNoBandwidth   = "no_bandwidth"
	DismissReasonTolerableRisk = "tolerable_risk"
	DismissReasonInaccurate    = "inaccurate"
	DismissReasonNotUsed       = "not_used"
)

var ValidStateTransitions = map[string][]string{
	StateOpen:         {StateAcknowledged, StateResolved, StateDismissed},
	StateAcknowledged: {StateResolved, StateDismissed},
}

type SecurityAdvisory struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	RepositoryID     *uuid.UUID
	GHSAPID          string
	CVEID            string
	Severity         string
	Summary          string
	Description      string
	AffectedPackage  string
	AffectedVersions string
	PatchedVersions  string
	State            string
	DismissedReason  string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (a *SecurityAdvisory) CanTransitionTo(next string) bool {
	allowed, ok := ValidStateTransitions[a.State]
	if !ok {
		return false
	}
	for _, state := range allowed {
		if state == next {
			return true
		}
	}
	return false
}
