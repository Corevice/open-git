package entity

import (
	"time"

	"github.com/google/uuid"
)

type SecretScanningAlert struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	SecretType     string
	CommitSHA      string
	FilePath       string
	Line           int
	State          string
	CreatedAt      time.Time
}
