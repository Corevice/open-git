package milestone

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type UpdateMilestoneInput struct {
	RepositoryID uuid.UUID
	Number       int
	Title        *string
	Description  *string
	State        *string
	DueOn        *time.Time
}

type UpdateMilestoneUsecase struct {
	milestoneRepo repository.IMilestoneRepository
}

func NewUpdateMilestoneUsecase(milestoneRepo repository.IMilestoneRepository) *UpdateMilestoneUsecase {
	return &UpdateMilestoneUsecase{milestoneRepo: milestoneRepo}
}

func (uc *UpdateMilestoneUsecase) Execute(ctx context.Context, input UpdateMilestoneInput) (*entity.Milestone, error) {
	milestone, err := uc.milestoneRepo.GetByNumber(ctx, input.RepositoryID, input.Number)
	if err != nil {
		return nil, err
	}
	if milestone == nil {
		return nil, apperror.ErrNotFound
	}

	if input.Title != nil {
		milestone.Title = *input.Title
	}
	if input.Description != nil {
		milestone.Description = *input.Description
	}
	if input.State != nil {
		if *input.State == "closed" && milestone.State != "closed" {
			now := time.Now().UTC()
			milestone.ClosedAt = &now
		}
		milestone.State = *input.State
	}
	if input.DueOn != nil {
		milestone.DueOn = input.DueOn
	}

	if err := uc.milestoneRepo.Update(ctx, milestone); err != nil {
		return nil, err
	}

	return milestone, nil
}
