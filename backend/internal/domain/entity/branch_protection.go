package entity

import "github.com/google/uuid"

// BranchProtection holds the protection rules configured for a repository ref.
type BranchProtection struct {
	ID              uuid.UUID
	RepositoryID    uuid.UUID
	Ref             string
	RequiredReviews int
	RequiredChecks  []string
}
