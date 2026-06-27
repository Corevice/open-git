package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	repo "github.com/open-git/backend/internal/repository"
)

type ListRepositoriesInput struct {
	OrganizationID uuid.UUID
	RequestUserID  uuid.UUID
	Page           int
	PerPage        int
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

func (u *ListRepositoriesUsecase) Execute(ctx context.Context, input ListRepositoriesInput) ([]*entity.Repository, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}
	perPage := input.PerPage
	if perPage < 1 {
		perPage = 30
	}

	repositories, err := u.repos.ListByOrg(ctx, input.OrganizationID, page, perPage)
	if err != nil {
		return nil, err
	}

	visible := make([]*entity.Repository, 0, len(repositories))
	for _, r := range repositories {
		if r.Visibility != entity.VisibilityPrivate {
			visible = append(visible, r)
			continue
		}
		if input.RequestUserID == uuid.Nil {
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
