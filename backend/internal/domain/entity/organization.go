package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	PlanFree = "free"
	PlanPro  = "pro"
)

type Organization struct {
	ID          uuid.UUID
	Login       string
	Name        string
	Description string
	PlanTier    string
	CreatedAt   time.Time
}
