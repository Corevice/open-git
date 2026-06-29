// Package service defines domain-level service interfaces that abstract
// infrastructure concerns (such as Git operations) from the usecase layer.
package service

import (
	"context"

	"github.com/google/uuid"
)

// GitService abstracts Git repository operations required by usecases.
type GitService interface {
	// BranchExists reports whether the given ref exists in the repository.
	BranchExists(ctx context.Context, repositoryID uuid.UUID, ref string) (bool, error)
	// ResolveRef resolves a ref (branch, tag, or revision) to its commit SHA.
	ResolveRef(ctx context.Context, repositoryID uuid.UUID, ref string) (string, error)
	// Merge merges the head ref into the base ref using the given merge method
	// ("merge", "squash", or "rebase").
	Merge(ctx context.Context, repositoryID uuid.UUID, baseRef, headRef, mergeMethod string) error
}
