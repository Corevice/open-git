package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending = "pending"
	StatusSuccess = "success"
	StatusFailed  = "failed"
)

type WebhookDelivery struct {
	ID               uuid.UUID
	WebhookID        uuid.UUID
	OrganizationID   uuid.UUID
	Event            string
	Status           string
	StatusCode       *int
	RequestHeaders   map[string][]string
	RequestBody      string
	ResponseHeaders  map[string][]string
	ResponseBody     *string
	DurationMs       *int
	Attempt          int
	Redelivery       bool
	ParentDeliveryID *uuid.UUID
	DeliveredAt      *time.Time
	CreatedAt        time.Time
}
