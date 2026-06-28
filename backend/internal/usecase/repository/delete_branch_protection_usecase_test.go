package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/usecase/repository"
)

type deleteMockBranchProtectionRepo struct {
	byPattern    map[string]*entity.BranchProtection
	deleteCalled bool
}

func (m *deleteMockBranchProtectionRepo) GetByPattern(_ context.Context, orgID, repoID uuid.UUID, pattern string) (*entity.BranchProtection, error) {
	if m.byPattern == nil {
		return nil, apperror.ErrNotFound
	}
	rule, ok := m.byPattern[branchProtectionKey(orgID, repoID, pattern)]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return rule, nil
}

func (m *deleteMockBranchProtectionRepo) Upsert(context.Context, uuid.UUID, uuid.UUID, *entity.BranchProtection) (*entity.BranchProtection, error) {
	return nil, nil
}

func (m *deleteMockBranchProtectionRepo) DeleteByPattern(_ context.Context, orgID, repoID uuid.UUID, pattern string) error {
	m.deleteCalled = true
	if m.byPattern != nil {
		delete(m.byPattern, branchProtectionKey(orgID, repoID, pattern))
	}
	return nil
}

type deleteMockAuditLogRepo struct {
	calls []upsertAuditLogCall
}

func (m *deleteMockAuditLogRepo) Record(_ context.Context, _, _ uuid.UUID, action, _ string, _ uuid.UUID, _ map[string]any) error {
	m.calls = append(m.calls, upsertAuditLogCall{action: action})
	return nil
}

func TestDeleteBranchProtectionUsecase(t *testing.T) {
	t.Parallel()

	t.Run("happy path delete calls audit log", func(t *testing.T) {
		t.Parallel()

		pattern := "main"
		branchProtectionRepo := &deleteMockBranchProtectionRepo{
			byPattern: map[string]*entity.BranchProtection{
				branchProtectionKey(testBranchProtectionOrgID, testBranchProtectionRepoID, pattern): {
					Pattern: pattern,
				},
			},
		}
		auditLogRepo := &deleteMockAuditLogRepo{}
		uc := repository.NewDeleteBranchProtectionUsecase(branchProtectionRepo, auditLogRepo)

		err := uc.Execute(
			context.Background(),
			testBranchProtectionOrgID,
			testBranchProtectionRepoID,
			testBranchProtectionActorID,
			pattern,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !branchProtectionRepo.deleteCalled {
			t.Fatal("expected DeleteByPattern to be called")
		}
		if len(auditLogRepo.calls) != 1 {
			t.Fatalf("expected 1 audit log call, got %d", len(auditLogRepo.calls))
		}
		if auditLogRepo.calls[0].action != "branch_protection.delete" {
			t.Fatalf("expected audit action branch_protection.delete, got %q", auditLogRepo.calls[0].action)
		}
	})

	t.Run("missing pattern returns ErrNotFound without calling audit log", func(t *testing.T) {
		t.Parallel()

		branchProtectionRepo := &deleteMockBranchProtectionRepo{
			byPattern: map[string]*entity.BranchProtection{},
		}
		auditLogRepo := &deleteMockAuditLogRepo{}
		uc := repository.NewDeleteBranchProtectionUsecase(branchProtectionRepo, auditLogRepo)

		err := uc.Execute(
			context.Background(),
			testBranchProtectionOrgID,
			testBranchProtectionRepoID,
			testBranchProtectionActorID,
			"missing",
		)
		if !errors.Is(err, apperror.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
		if branchProtectionRepo.deleteCalled {
			t.Fatal("expected DeleteByPattern not to be called")
		}
		if len(auditLogRepo.calls) != 0 {
			t.Fatalf("expected no audit log calls, got %d", len(auditLogRepo.calls))
		}
	})
}
