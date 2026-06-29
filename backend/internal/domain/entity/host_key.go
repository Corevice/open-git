package entity

import (
	"time"

	"github.com/google/uuid"
)

type HostKey struct {
	ID         uuid.UUID
	Algorithm  string
	PrivateKey string
	CreatedAt  time.Time
}
