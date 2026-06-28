package entity

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	Action         string
	TargetType     string
	TargetID       string
	Metadata       map[string]any
	CreatedAt      time.Time
}
