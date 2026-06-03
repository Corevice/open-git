package entity

import "github.com/google/uuid"

type BranchProtection struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Pattern        string
	RequiredReviews int
	RequiredChecks []string
}
