package milestone

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type ListMilestonesInput struct {
	RepositoryID uuid.UUID
	State        string
	Page         int
	PerPage      int
}

type ListMilestonesOutput struct {
	Milestones []*entity.Milestone
	Total      int
	Page       int
	PerPage    int
}

type ListMilestonesUsecase struct {
	milestoneRepo repository.IMilestoneRepository
}

func NewListMilestonesUsecase(milestoneRepo repository.IMilestoneRepository) *ListMilestonesUsecase {
	return &ListMilestonesUsecase{milestoneRepo: milestoneRepo}
}

func (uc *ListMilestonesUsecase) Execute(ctx context.Context, input ListMilestonesInput) (*ListMilestonesOutput, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}
	perPage := input.PerPage
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	milestones, total, err := uc.milestoneRepo.ListByRepo(ctx, input.RepositoryID, input.State, page, perPage)
	if err != nil {
		return nil, err
	}

	return &ListMilestonesOutput{
		Milestones: milestones,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
	}, nil
}
