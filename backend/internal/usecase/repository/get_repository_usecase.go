package repository

import (
	"context"
	"errors"

	"github.com/open-git/backend/internal/domain"
	repo "github.com/open-git/backend/internal/repository"
)

var ErrNotFound = errors.New("not found")

type GetRepositoryInput struct {
	RequestUserID int64
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

func (u *GetRepositoryUsecase) Execute(ctx context.Context, input GetRepositoryInput) (*domain.Repository, error) {
	owner, err := u.users.GetByLogin(ctx, input.OwnerLogin)
	if err != nil || owner == nil {
		return nil, ErrNotFound
	}

	repository, err := u.repos.GetByOwnerAndName(ctx, owner.ID, input.Name)
	if err != nil || repository == nil {
		return nil, ErrNotFound
	}

	if repository.Visibility == domain.VisibilityPrivate {
		if input.RequestUserID == 0 {
			return nil, ErrNotFound
		}
		hasAccess, err := u.memberships.HasReadAccess(ctx, input.RequestUserID, repository.OrganizationID)
		if err != nil || !hasAccess {
			return nil, ErrNotFound
		}
	}

	repository.OwnerLogin = owner.Login
	return repository, nil
}
