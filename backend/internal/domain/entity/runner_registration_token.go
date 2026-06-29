package entity

import (
	"time"

	"github.com/google/uuid"
)

type RunnerRegistrationToken struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	TokenHash      string
	ExpiresAt      time.Time
	UsedAt         *time.Time
}
