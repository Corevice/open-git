package repository

import (
	"context"

	"github.com/google/uuid"
	repo "github.com/open-git/backend/internal/repository"
)

type DeleteBranchProtectionUsecase struct {
	branchProtectionRepo BranchProtectionRepository
	auditLogRepo         repo.IAuditLogRepository
}

func NewDeleteBranchProtectionUsecase(
	branchProtectionRepo BranchProtectionRepository,
	auditLogRepo repo.IAuditLogRepository,
) *DeleteBranchProtectionUsecase {
	return &DeleteBranchProtectionUsecase{
		branchProtectionRepo: branchProtectionRepo,
		auditLogRepo:         auditLogRepo,
	}
}

func (u *DeleteBranchProtectionUsecase) Execute(
	ctx context.Context,
	orgID, repoID, actorID uuid.UUID,
	pattern string,
) error {
	if _, err := u.branchProtectionRepo.GetByPattern(ctx, orgID, repoID, pattern); err != nil {
		return err
	}

	if err := u.branchProtectionRepo.DeleteByPattern(ctx, orgID, repoID, pattern); err != nil {
		return err
	}

	return u.auditLogRepo.Record(ctx, orgID, actorID, "branch_protection.delete", "repository", repoID, map[string]any{
		"pattern": pattern,
	})
}
