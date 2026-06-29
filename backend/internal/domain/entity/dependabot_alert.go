package entity

import (
	"time"

	"github.com/google/uuid"
)

type DependabotAlertState string

const (
	DependabotAlertStateOpen      DependabotAlertState = "open"
	DependabotAlertStateDismissed DependabotAlertState = "dismissed"
	DependabotAlertStateFixed     DependabotAlertState = "fixed"
)

type DependabotAlert struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	RepositoryID    uuid.UUID
	AlertNumber     int
	AdvisoryID      uuid.UUID
	ManifestPath    string
	State           DependabotAlertState
	DismissedReason *DismissedReason
	AutoDismissedAt *time.Time
}
