package entity

import (
	"time"

	"github.com/google/uuid"
)

type DependabotAlert struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	RepositoryID     uuid.UUID
	AlertNumber      int
	AdvisoryID       uuid.UUID
	ManifestPath     string
	State            string
	AutoDismissedAt  *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
