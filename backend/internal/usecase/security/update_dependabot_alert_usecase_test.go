package security_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	securityusecase "github.com/open-git/backend/internal/usecase/security"
)

var (
	testRepoID      = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	testAlertNumber = 1
)

type mockDependabotAlertRepo struct {
	alert       *entity.DependabotAlert
	updateCalls int
}

func (m *mockDependabotAlertRepo) ListByRepo(context.Context, uuid.UUID, uuid.UUID, string, int, int) ([]*entity.DependabotAlert, int, error) {
	return nil, 0, nil
}

func (m *mockDependabotAlertRepo) GetByAlertNumber(_ context.Context, orgID, repoID uuid.UUID, alertNumber int) (*entity.DependabotAlert, error) {
	if m.alert == nil || m.alert.OrganizationID != orgID || m.alert.RepositoryID != repoID || m.alert.AlertNumber != alertNumber {
		return nil, nil
	}
	copy := *m.alert
	return &copy, nil
}

func (m *mockDependabotAlertRepo) UpdateState(_ context.Context, orgID, repoID uuid.UUID, alertNumber int, state entity.DependabotAlertState, reason *entity.DismissedReason) (*entity.DependabotAlert, error) {
	m.updateCalls++
	updated := *m.alert
	updated.State = state
	updated.DismissedReason = reason
	m.alert = &updated
	return &updated, nil
}

var _ repository.IDependabotAlertRepository = (*mockDependabotAlertRepo)(nil)

func TestUpdateDependabotAlertUsecase(t *testing.T) {
	t.Parallel()

	reasonNoBandwidth := entity.DismissedReasonNoBandwidth

	tests := []struct {
		name            string
		initialState    entity.DependabotAlertState
		newState        entity.DependabotAlertState
		dismissedReason *entity.DismissedReason
		wantErr         error
		wantUpdate      bool
	}{
		{
			name:            "dismiss with reason",
			initialState:    entity.DependabotAlertStateOpen,
			newState:        entity.DependabotAlertStateDismissed,
			dismissedReason: &reasonNoBandwidth,
			wantUpdate:      true,
		},
		{
			name:         "dismiss without reason",
			initialState: entity.DependabotAlertStateOpen,
			newState:     entity.DependabotAlertStateDismissed,
			wantErr:      apperror.ErrValidation,
		},
		{
			name:         "reopen dismissed",
			initialState: entity.DependabotAlertStateDismissed,
			newState:     entity.DependabotAlertStateOpen,
			wantUpdate:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &mockDependabotAlertRepo{
				alert: &entity.DependabotAlert{
					OrganizationID: testOrgID,
					RepositoryID:   testRepoID,
					AlertNumber:    testAlertNumber,
					State:          tt.initialState,
				},
			}
			uc := securityusecase.NewUpdateDependabotAlertUsecase(repo)

			result, err := uc.Execute(context.Background(), securityusecase.UpdateDependabotAlertInput{
				OrganizationID:  testOrgID,
				RepositoryID:    testRepoID,
				AlertNumber:     testAlertNumber,
				State:           tt.newState,
				DismissedReason: tt.dismissedReason,
			})

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if repo.updateCalls != 0 {
					t.Fatalf("expected no update calls, got %d", repo.updateCalls)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantUpdate && repo.updateCalls != 1 {
				t.Fatalf("expected 1 update call, got %d", repo.updateCalls)
			}
			if tt.wantUpdate && result.State != tt.newState {
				t.Fatalf("expected state %s, got %s", tt.newState, result.State)
			}
		})
	}
}
