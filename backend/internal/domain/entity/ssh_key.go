package entity

import (
	"time"

	"github.com/google/uuid"
)

type SSHKey struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Title       string
	Fingerprint string
	PublicKey   string
	CreatedAt   time.Time
}
