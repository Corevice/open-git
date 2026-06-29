package entity

import (
	"time"

	"github.com/google/uuid"
)

type AdvisoryState string

const (
	AdvisoryStateOpen         AdvisoryState = "open"
	AdvisoryStateAcknowledged AdvisoryState = "acknowledged"
	AdvisoryStateResolved     AdvisoryState = "resolved"
	AdvisoryStateDismissed    AdvisoryState = "dismissed"
)

type AdvisorySeverity string

const (
	AdvisorySeverityCritical AdvisorySeverity = "critical"
	AdvisorySeverityHigh     AdvisorySeverity = "high"
	AdvisorySeverityMedium   AdvisorySeverity = "medium"
	AdvisorySeverityLow      AdvisorySeverity = "low"
)

type DismissedReason string

const (
	DismissedReasonNoBandwidth   DismissedReason = "no_bandwidth"
	DismissedReasonTolerableRisk DismissedReason = "tolerable_risk"
	DismissedReasonInaccurate    DismissedReason = "inaccurate"
	DismissedReasonNotUsed       DismissedReason = "not_used"
)

var ValidStateTransitions = map[AdvisoryState][]AdvisoryState{
	AdvisoryStateOpen:         {AdvisoryStateAcknowledged, AdvisoryStateResolved},
	AdvisoryStateAcknowledged: {AdvisoryStateResolved, AdvisoryStateDismissed},
	AdvisoryStateResolved:     {},
	AdvisoryStateDismissed:    {},
}

type SecurityAdvisory struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	RepositoryID     *uuid.UUID
	GHSAPID          string
	CVEID            string
	Severity         AdvisorySeverity
	Summary          string
	Description      string
	AffectedPackage  string
	AffectedVersions string
	PatchedVersions  string
	State            AdvisoryState
	DismissedReason  *DismissedReason
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
