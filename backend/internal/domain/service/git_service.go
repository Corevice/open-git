package service

import (
	"context"

	"github.com/google/uuid"
)

type GitService interface {
	BranchExists(ctx context.Context, repoID uuid.UUID, branch string) (bool, error)
	Merge(ctx context.Context, repoID uuid.UUID, base, head, method string) error
	ResolveRef(ctx context.Context, repoID uuid.UUID, ref string) (string, error)
}
