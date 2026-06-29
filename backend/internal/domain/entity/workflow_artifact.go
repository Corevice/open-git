package entity

import "github.com/google/uuid"

type WorkflowArtifact struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RunID          uuid.UUID
	Name           string
	SizeInBytes    int64
	Expired        bool
}
