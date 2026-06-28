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
	IPAddress      string         `db:"ip_address" json:"ip_address"`
	Metadata       map[string]any
	CreatedAt      time.Time
}
