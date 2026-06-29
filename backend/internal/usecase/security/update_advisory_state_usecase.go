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

var ErrInvalidTransition = errors.New("invalid advisory state transition")

type UpdateAdvisoryStateInput struct {
	OrganizationID  uuid.UUID
	GHSAPID         string
	State           entity.AdvisoryState
	DismissedReason *entity.DismissedReason
}

type UpdateAdvisoryStateUsecase struct {
	advisoryRepo repository.ISecurityAdvisoryRepository
}

func NewUpdateAdvisoryStateUsecase(advisoryRepo repository.ISecurityAdvisoryRepository) *UpdateAdvisoryStateUsecase {
	return &UpdateAdvisoryStateUsecase{advisoryRepo: advisoryRepo}
}

func (uc *UpdateAdvisoryStateUsecase) Execute(ctx context.Context, input UpdateAdvisoryStateInput) (*entity.SecurityAdvisory, error) {
	advisory, err := uc.advisoryRepo.GetByGHSAPID(ctx, input.OrganizationID, input.GHSAPID)
	if err != nil {
		return nil, err
	}
	if advisory == nil {
		return nil, apperror.ErrNotFound
	}

	if !isValidTransition(advisory.State, input.State) {
		return nil, fmt.Errorf(
			"%w: cannot transition from %s to %s",
			ErrInvalidTransition,
			advisory.State,
			input.State,
		)
	}

	if input.State == entity.AdvisoryStateDismissed {
		if input.DismissedReason == nil || *input.DismissedReason == "" {
			return nil, apperror.ErrValidation
		}
	}

	return uc.advisoryRepo.UpdateState(ctx, input.OrganizationID, input.GHSAPID, input.State, input.DismissedReason)
}

func isValidTransition(from, to entity.AdvisoryState) bool {
	allowed, ok := entity.ValidStateTransitions[from]
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
