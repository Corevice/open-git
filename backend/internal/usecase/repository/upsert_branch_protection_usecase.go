package repository

import (
	"context"
	"errors"
	"path"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	repo "github.com/open-git/backend/internal/repository"
)

type BranchProtectionRule struct {
	Pattern                      string
	RequiredApprovingReviewCount int
}

type BranchProtectionRepository interface {
	GetByPattern(ctx context.Context, orgID, repoID uuid.UUID, pattern string) (*BranchProtectionRule, error)
	Upsert(ctx context.Context, orgID, repoID uuid.UUID, rule *BranchProtectionRule) (*BranchProtectionRule, error)
	DeleteByPattern(ctx context.Context, orgID, repoID uuid.UUID, pattern string) error
}

type UpsertBranchProtectionUsecase struct {
	branchProtectionRepo BranchProtectionRepository
	auditLogRepo         repo.IAuditLogRepository
}

func NewUpsertBranchProtectionUsecase(
	branchProtectionRepo BranchProtectionRepository,
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
	rule *BranchProtectionRule,
) (*BranchProtectionRule, error) {
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
