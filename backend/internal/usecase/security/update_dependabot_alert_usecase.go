package security

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

var ErrInvalidDependabotTransition = errors.New("invalid dependabot alert state transition")

var validDependabotStateTransitions = map[entity.DependabotAlertState][]entity.DependabotAlertState{
	entity.DependabotAlertStateOpen:      {entity.DependabotAlertStateDismissed},
	entity.DependabotAlertStateDismissed: {entity.DependabotAlertStateOpen},
	entity.DependabotAlertStateFixed:     {},
}

type UpdateDependabotAlertInput struct {
	OrganizationID  uuid.UUID
	RepositoryID    uuid.UUID
	AlertNumber     int
	State           entity.DependabotAlertState
	DismissedReason *entity.DismissedReason
}

type UpdateDependabotAlertUsecase struct {
	alertRepo repository.IDependabotAlertRepository
}

func NewUpdateDependabotAlertUsecase(alertRepo repository.IDependabotAlertRepository) *UpdateDependabotAlertUsecase {
	return &UpdateDependabotAlertUsecase{alertRepo: alertRepo}
}

func (uc *UpdateDependabotAlertUsecase) Execute(ctx context.Context, input UpdateDependabotAlertInput) (*entity.DependabotAlert, error) {
	if input.State != entity.DependabotAlertStateOpen && input.State != entity.DependabotAlertStateDismissed {
		return nil, apperror.ErrValidation
	}

	alert, err := uc.alertRepo.GetByAlertNumber(ctx, input.OrganizationID, input.RepositoryID, input.AlertNumber)
	if err != nil {
		return nil, err
	}
	if alert == nil {
		return nil, apperror.ErrNotFound
	}

	if !isValidDependabotTransition(alert.State, input.State) {
		return nil, fmt.Errorf(
			"%w: cannot transition from %s to %s",
			ErrInvalidDependabotTransition,
			alert.State,
			input.State,
		)
	}

	if input.State == entity.DependabotAlertStateDismissed {
		if input.DismissedReason == nil || *input.DismissedReason == "" {
			return nil, apperror.ErrValidation
		}
	}

	return uc.alertRepo.UpdateState(ctx, input.OrganizationID, input.RepositoryID, input.AlertNumber, input.State, input.DismissedReason)
}

func isValidDependabotTransition(from, to entity.DependabotAlertState) bool {
	allowed, ok := validDependabotStateTransitions[from]
	if !ok {
		return false
	}
	for _, state := range allowed {
		if state == to {
			return true
		}
	}
	return false
}
