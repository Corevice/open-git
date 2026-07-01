package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	repo "github.com/open-git/backend/internal/repository"
)

// ErrNotFound aliases apperror.ErrNotFound so that repositories not found here
// are translated to a 404 by the central HTTP error handler (which recognizes
// apperror sentinels) instead of falling through to a generic 500. Existing
// errors.Is(err, ErrNotFound) checks keep working since it is the same sentinel.
var ErrNotFound = apperror.ErrNotFound

type GetRepositoryInput struct {
	RequestUserID uuid.UUID
	OwnerLogin    string
	Name          string
}

type GetRepositoryUsecase struct {
	repos       repo.IRepositoryRepository
	users       repo.IUserRepository
	memberships repo.IMembershipRepository
}

func NewGetRepositoryUsecase(
	repos repo.IRepositoryRepository,
	users repo.IUserRepository,
	memberships repo.IMembershipRepository,
) *GetRepositoryUsecase {
	return &GetRepositoryUsecase{repos: repos, users: users, memberships: memberships}
}

func (u *GetRepositoryUsecase) Execute(ctx context.Context, input GetRepositoryInput) (*entity.Repository, error) {
	repository, err := u.repos.GetByOwnerLoginAndName(ctx, input.OwnerLogin, input.Name)
	if err != nil || repository == nil {
		return nil, ErrNotFound
	}

	if repository.Visibility == entity.VisibilityPrivate {
		if input.RequestUserID == uuid.Nil {
			return nil, ErrNotFound
		}
		hasAccess, err := u.memberships.HasReadAccess(ctx, input.RequestUserID, repository.OrganizationID)
		if err != nil || !hasAccess {
			return nil, ErrNotFound
		}
	}

	return repository, nil
}
