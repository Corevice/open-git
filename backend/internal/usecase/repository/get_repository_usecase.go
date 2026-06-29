package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	repo "github.com/open-git/backend/internal/repository"
)

var ErrNotFound = errors.New("not found")

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
