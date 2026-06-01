package repository

import (
	"context"

	"github.com/open-git/backend/internal/domain"
	repo "github.com/open-git/backend/internal/repository"
)

type ListRepositoriesInput struct {
	OrganizationID int64
	RequestUserID  int64
}

type ListRepositoriesUsecase struct {
	repos       repo.IRepositoryRepository
	memberships repo.IMembershipRepository
}

func NewListRepositoriesUsecase(
	repos repo.IRepositoryRepository,
	memberships repo.IMembershipRepository,
) *ListRepositoriesUsecase {
	return &ListRepositoriesUsecase{repos: repos, memberships: memberships}
}

func (u *ListRepositoriesUsecase) Execute(ctx context.Context, input ListRepositoriesInput) ([]*domain.Repository, error) {
	repositories, err := u.repos.ListByOrg(ctx, input.OrganizationID)
	if err != nil {
		return nil, err
	}

	visible := make([]*domain.Repository, 0, len(repositories))
	for _, r := range repositories {
		if r.Visibility != domain.VisibilityPrivate {
			visible = append(visible, r)
			continue
		}
		if input.RequestUserID == 0 {
			continue
		}
		hasAccess, err := u.memberships.HasReadAccess(ctx, input.RequestUserID, r.OrganizationID)
		if err != nil || !hasAccess {
			continue
		}
		visible = append(visible, r)
	}

	return visible, nil
}
