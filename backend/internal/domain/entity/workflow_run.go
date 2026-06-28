package entity

import "github.com/google/uuid"

type WorkflowRun struct {
	ID             uuid.UUID
	RepositoryID   uuid.UUID
	OrganizationID uuid.UUID
	Workflow       string
	Status         string
	Conclusion     string
	HeadSHA        string
}
