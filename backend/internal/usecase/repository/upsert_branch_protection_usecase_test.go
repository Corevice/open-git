package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/usecase/repository"
)

var (
	testBranchProtectionOrgID  = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	testBranchProtectionRepoID = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	testBranchProtectionActorID = uuid.MustParse("00000000-0000-0000-0000-000000000012")
)

type upsertMockBranchProtectionRepo struct {
	byPattern   map[string]*entity.BranchProtection
	upsertCalls int
}

func branchProtectionKey(orgID, repoID uuid.UUID, pattern string) string {
	return fmt.Sprintf("%s:%s:%s", orgID, repoID, pattern)
}

func (m *upsertMockBranchProtectionRepo) GetByPattern(_ context.Context, orgID, repoID uuid.UUID, pattern string) (*entity.BranchProtection, error) {
	if m.byPattern == nil {
		return nil, apperror.ErrNotFound
	}
	rule, ok := m.byPattern[branchProtectionKey(orgID, repoID, pattern)]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return rule, nil
}

func (m *upsertMockBranchProtectionRepo) Upsert(_ context.Context, orgID, repoID uuid.UUID, rule *entity.BranchProtection) (*entity.BranchProtection, error) {
	m.upsertCalls++
	if m.byPattern == nil {
		m.byPattern = map[string]*entity.BranchProtection{}
	}
	m.byPattern[branchProtectionKey(orgID, repoID, rule.Pattern)] = rule
	return rule, nil
}

func (m *upsertMockBranchProtectionRepo) DeleteByPattern(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}

type upsertMockAuditLogRepo struct {
	calls []upsertAuditLogCall
}

type upsertAuditLogCall struct {
	action string
}

func (m *upsertMockAuditLogRepo) Record(_ context.Context, _, _ uuid.UUID, action, _ string, _ uuid.UUID, _ map[string]any) error {
	m.calls = append(m.calls, upsertAuditLogCall{action: action})
	return nil
}

func TestUpsertBranchProtectionUsecase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		rule          *entity.BranchProtection
		existing      *entity.BranchProtection
		wantErr       error
		wantAction    string
		wantUpsert    bool
		wantAuditLogs int
	}{
		{
			name: "happy path create",
			rule: &entity.BranchProtection{
				Pattern:                        "main",
				RequiredApprovingReviewCount:   1,
			},
			wantAction:    "branch_protection.create",
			wantUpsert:    true,
			wantAuditLogs: 1,
		},
		{
			name: "happy path update",
			rule: &entity.BranchProtection{
				Pattern:                        "release/*",
				RequiredApprovingReviewCount:   2,
			},
			existing: &entity.BranchProtection{
				Pattern:                        "release/*",
				RequiredApprovingReviewCount:   1,
			},
			wantAction:    "branch_protection.update",
			wantUpsert:    true,
			wantAuditLogs: 1,
		},
		{
			name: "empty pattern",
			rule: &entity.BranchProtection{
				Pattern:                      "",
				RequiredApprovingReviewCount: 0,
			},
			wantErr:       apperror.ErrValidation,
			wantUpsert:    false,
			wantAuditLogs: 0,
		},
		{
			name: "invalid glob pattern",
			rule: &entity.BranchProtection{
				Pattern:                      "[",
				RequiredApprovingReviewCount: 0,
			},
			wantErr:       apperror.ErrValidation,
			wantUpsert:    false,
			wantAuditLogs: 0,
		},
		{
			name: "review count -1",
			rule: &entity.BranchProtection{
				Pattern:                      "main",
				RequiredApprovingReviewCount: -1,
			},
			wantErr:       apperror.ErrValidation,
			wantUpsert:    false,
			wantAuditLogs: 0,
		},
		{
			name: "review count 7",
			rule: &entity.BranchProtection{
				Pattern:                      "main",
				RequiredApprovingReviewCount: 7,
			},
			wantErr:       apperror.ErrValidation,
			wantUpsert:    false,
			wantAuditLogs: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			branchProtectionRepo := &upsertMockBranchProtectionRepo{
				byPattern: map[string]*entity.BranchProtection{},
			}
			if tt.existing != nil {
				branchProtectionRepo.byPattern[branchProtectionKey(
					testBranchProtectionOrgID,
					testBranchProtectionRepoID,
					tt.existing.Pattern,
				)] = tt.existing
			}

			auditLogRepo := &upsertMockAuditLogRepo{}
			uc := repository.NewUpsertBranchProtectionUsecase(branchProtectionRepo, auditLogRepo)

			result, err := uc.Execute(
				context.Background(),
				testBranchProtectionOrgID,
				testBranchProtectionRepoID,
				testBranchProtectionActorID,
				tt.rule,
			)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if result != nil {
					t.Fatal("expected nil result on error")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantUpsert && branchProtectionRepo.upsertCalls != 1 {
				t.Fatalf("expected Upsert to be called once, got %d", branchProtectionRepo.upsertCalls)
			}
			if !tt.wantUpsert && branchProtectionRepo.upsertCalls != 0 {
				t.Fatalf("expected Upsert not to be called, got %d", branchProtectionRepo.upsertCalls)
			}

			if len(auditLogRepo.calls) != tt.wantAuditLogs {
				t.Fatalf("expected %d audit log calls, got %d", tt.wantAuditLogs, len(auditLogRepo.calls))
			}
			if tt.wantAction != "" {
				if len(auditLogRepo.calls) != 1 || auditLogRepo.calls[0].action != tt.wantAction {
					t.Fatalf("expected audit action %q, got %+v", tt.wantAction, auditLogRepo.calls)
				}
			}
		})
	}
}
