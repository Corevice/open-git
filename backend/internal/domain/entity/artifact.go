package entity

import (
	"time"

	"github.com/google/uuid"
)

type ArtifactStatus string

const (
	ArtifactStatusPending   ArtifactStatus = "pending"
	ArtifactStatusUploading ArtifactStatus = "uploading"
	ArtifactStatusCompleted ArtifactStatus = "completed"
	ArtifactStatusFailed    ArtifactStatus = "failed"
	ArtifactStatusExpired   ArtifactStatus = "expired"
)

type Artifact struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RunID          uuid.UUID
	Name           string
	SizeBytes      int64
	StorageKey     string
	ExpiresAt      time.Time
	CreatedAt      time.Time
}
