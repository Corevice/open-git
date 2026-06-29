package entity

import (
	"github.com/google/uuid"
)

type BranchProtection struct {
	ID                       uuid.UUID
	RepositoryID             uuid.UUID
	Pattern                  string
	RequiredApprovingReviews int
	RequiredStatusChecks     []string
	DismissStaleReviews      bool
	RequireCodeOwnerReviews  bool
}
