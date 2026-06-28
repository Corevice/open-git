package repository

import (
	"context"
	"errors"
	"path"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	repo "github.com/open-git/backend/internal/repository"
)

type UpsertBranchProtectionUsecase struct {
	branchProtectionRepo repo.IBranchProtectionRepository
	auditLogRepo         repo.IAuditLogRepository
}

func NewUpsertBranchProtectionUsecase(
	branchProtectionRepo repo.IBranchProtectionRepository,
	auditLogRepo repo.IAuditLogRepository,
) *UpsertBranchProtectionUsecase {
	return &UpsertBranchProtectionUsecase{
		branchProtectionRepo: branchProtectionRepo,
		auditLogRepo:         auditLogRepo,
	}
}

func (u *UpsertBranchProtectionUsecase) Execute(
	ctx context.Context,
	orgID, repoID, actorID uuid.UUID,
	rule *entity.BranchProtection,
) (*entity.BranchProtection, error) {
	if rule.Pattern == "" {
		return nil, apperror.ErrValidation
	}
	if _, err := path.Match(rule.Pattern, "_"); err != nil {
		return nil, apperror.ErrValidation
	}
	if rule.RequiredApprovingReviewCount < 0 || rule.RequiredApprovingReviewCount > 6 {
		return nil, apperror.ErrValidation
	}

	action := "branch_protection.create"
	_, err := u.branchProtectionRepo.GetByPattern(ctx, orgID, repoID, rule.Pattern)
	if err != nil {
		if !errors.Is(err, apperror.ErrNotFound) {
			return nil, err
		}
	} else {
		action = "branch_protection.update"
	}

	result, err := u.branchProtectionRepo.Upsert(ctx, orgID, repoID, rule)
	if err != nil {
		return nil, err
	}

	if err := u.auditLogRepo.Record(ctx, orgID, actorID, action, "repository", repoID, map[string]any{
		"pattern": rule.Pattern,
	}); err != nil {
		return nil, err
	}

	return result, nil
}
