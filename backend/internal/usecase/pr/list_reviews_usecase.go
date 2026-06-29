package pr

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type ListReviewsInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Number         int
}

type ListReviewsUsecase struct {
	prRepo     repository.IPullRequestRepository
	reviewRepo repository.IReviewRepository
}

func NewListReviewsUsecase(
	prRepo repository.IPullRequestRepository,
	reviewRepo repository.IReviewRepository,
) *ListReviewsUsecase {
	return &ListReviewsUsecase{
		prRepo:     prRepo,
		reviewRepo: reviewRepo,
	}
}

func (uc *ListReviewsUsecase) Execute(ctx context.Context, input ListReviewsInput) ([]*entity.Review, error) {
	pr, err := uc.prRepo.GetByNumber(ctx, input.RepositoryID, input.Number)
	if err != nil {
		return nil, err
	}
	if pr.OrganizationID != input.OrganizationID {
		return nil, apperror.ErrNotFound
	}

	reviews, err := uc.reviewRepo.ListByPR(ctx, pr.ID)
	if err != nil {
		return nil, err
	}
	if reviews == nil {
		reviews = []*entity.Review{}
	}

	return reviews, nil
}
