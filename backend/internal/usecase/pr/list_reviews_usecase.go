package pr

import (
	"context"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	repointerface "github.com/open-git/backend/internal/repository"
)

type ListReviewsUsecase struct {
	repos      repointerface.IRepositoryRepository
	prRepo     domainrepo.IPullRequestRepository
	reviewRepo domainrepo.IReviewRepository
}

func NewListReviewsUsecase(
	repos repointerface.IRepositoryRepository,
	prRepo domainrepo.IPullRequestRepository,
	reviewRepo domainrepo.IReviewRepository,
) *ListReviewsUsecase {
	return &ListReviewsUsecase{
		repos:      repos,
		prRepo:     prRepo,
		reviewRepo: reviewRepo,
	}
}

func (uc *ListReviewsUsecase) Execute(ctx context.Context, owner, repo string, prNumber int) ([]*entity.Review, error) {
	repository, err := uc.repos.GetByOwnerLoginAndName(ctx, owner, repo)
	if err != nil || repository == nil {
		return nil, apperror.ErrNotFound
	}

	pr, err := uc.prRepo.GetByNumber(ctx, repository.ID, prNumber)
	if err != nil {
		return nil, err
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
