package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	RunnerTypeAct      = "act"
	RunnerTypeOfficial = "official"

	RunnerStatusOnline  = "online"
	RunnerStatusOffline = "offline"
	RunnerStatusBusy    = "busy"
)

type Runner struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Labels         []string
	OS             string
	Arch           string
	RunnerType     string
	Status         string
	LastSeenAt     *time.Time
	Ephemeral      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
