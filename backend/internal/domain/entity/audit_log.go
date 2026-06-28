package entity

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	ActorLogin     string
	Action         string
	TargetType     string
	TargetID       string
	Metadata       map[string]any
	IPAddress      string
	UserAgent      string
	CreatedAt      time.Time
}
