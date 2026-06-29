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
	testOrgID   = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	testGHSAPID = "GHSA-test-0001"
)

type mockSecurityAdvisoryRepo struct {
	advisory    *entity.SecurityAdvisory
	updateCalls int
}

func (m *mockSecurityAdvisoryRepo) ListByOrg(context.Context, uuid.UUID, string, string, int, int) ([]*entity.SecurityAdvisory, int, error) {
	return nil, 0, nil
}

func (m *mockSecurityAdvisoryRepo) GetByGHSAPID(_ context.Context, orgID uuid.UUID, ghsaID string) (*entity.SecurityAdvisory, error) {
	if m.advisory == nil || m.advisory.OrganizationID != orgID || m.advisory.GHSAPID != ghsaID {
		return nil, nil
	}
	copy := *m.advisory
	return &copy, nil
}

func (m *mockSecurityAdvisoryRepo) UpdateState(_ context.Context, orgID uuid.UUID, ghsaID string, state entity.AdvisoryState, reason *entity.DismissedReason) (*entity.SecurityAdvisory, error) {
	m.updateCalls++
	updated := *m.advisory
	updated.State = state
	updated.DismissedReason = reason
	m.advisory = &updated
	return &updated, nil
}

var _ repository.ISecurityAdvisoryRepository = (*mockSecurityAdvisoryRepo)(nil)

func TestUpdateAdvisoryStateUsecase(t *testing.T) {
	t.Parallel()

	reasonNoBandwidth := entity.DismissedReasonNoBandwidth
	emptyReason := entity.DismissedReason("")

	tests := []struct {
		name            string
		initialState    entity.AdvisoryState
		newState        entity.AdvisoryState
		dismissedReason *entity.DismissedReason
		wantErr         error
		wantUpdate      bool
	}{
		{
			name:         "open to acknowledged without reason",
			initialState: entity.AdvisoryStateOpen,
			newState:     entity.AdvisoryStateAcknowledged,
			wantUpdate:   true,
		},
		{
			name:            "acknowledged to dismissed with reason",
			initialState:    entity.AdvisoryStateAcknowledged,
			newState:        entity.AdvisoryStateDismissed,
			dismissedReason: &reasonNoBandwidth,
			wantUpdate:      true,
		},
		{
			name:         "open to resolved without reason",
			initialState: entity.AdvisoryStateOpen,
			newState:     entity.AdvisoryStateResolved,
			wantUpdate:   true,
		},
		{
			name:         "resolved to open invalid transition",
			initialState: entity.AdvisoryStateResolved,
			newState:     entity.AdvisoryStateOpen,
			wantErr:      securityusecase.ErrInvalidTransition,
		},
		{
			name:            "acknowledged to dismissed with empty reason",
			initialState:    entity.AdvisoryStateAcknowledged,
			newState:        entity.AdvisoryStateDismissed,
			dismissedReason: &emptyReason,
			wantErr:         apperror.ErrValidation,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &mockSecurityAdvisoryRepo{
				advisory: &entity.SecurityAdvisory{
					OrganizationID: testOrgID,
					GHSAPID:        testGHSAPID,
					State:          tt.initialState,
				},
			}
			uc := securityusecase.NewUpdateAdvisoryStateUsecase(repo)

			_, err := uc.Execute(context.Background(), securityusecase.UpdateAdvisoryStateInput{
				OrganizationID:  testOrgID,
				GHSAPID:         testGHSAPID,
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
			if !tt.wantUpdate && repo.updateCalls != 0 {
				t.Fatalf("expected no update calls, got %d", repo.updateCalls)
			}
			if tt.wantUpdate && repo.updateCalls != 1 {
				t.Fatalf("expected 1 update call, got %d", repo.updateCalls)
			}
			if tt.wantUpdate && repo.advisory.State != tt.newState {
				t.Fatalf("expected state %s, got %s", tt.newState, repo.advisory.State)
			}
		})
	}
}
