package entity

import (
	"fmt"
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
	RepositoryID   uuid.UUID
	WorkflowRunID  uuid.UUID
	Name           string
	StorageKey     string
	SizeInBytes    int64
	Status         ArtifactStatus
	RetentionDays  int
	CreatedAt      time.Time
	ExpiresAt      time.Time
	DeletedAt      *time.Time
}

func (a *Artifact) IsExpired() bool {
	return time.Now().After(a.ExpiresAt) || a.Status == ArtifactStatusExpired
}

func ArtifactStorageKey(orgLogin, repoName, runID, artifactID, name string) string {
	return fmt.Sprintf("org/%s/repo/%s/runs/%s/%s/%s.zip", orgLogin, repoName, runID, artifactID, name)
}
