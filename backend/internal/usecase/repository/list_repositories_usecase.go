package repository

import (
	"context"
	"encoding/binary"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	repo "github.com/open-git/backend/internal/repository"
)

type ListRepositoriesInput struct {
	OrganizationID uuid.UUID
	OwnerID        uuid.UUID
	OwnerLogin     string
	RequestUserID  uuid.UUID
	Page           int
	PerPage        int
}

type ListRepositoriesUsecase struct {
	repos       repo.IRepositoryRepository
	memberships repo.IMembershipRepository
	users       repo.IUserRepository
}

func NewListRepositoriesUsecase(
	repos repo.IRepositoryRepository,
	memberships repo.IMembershipRepository,
	users repo.IUserRepository,
) *ListRepositoriesUsecase {
	return &ListRepositoriesUsecase{repos: repos, memberships: memberships, users: users}
}

func int64ToUUID(id int64) uuid.UUID {
	if id == 0 {
		return uuid.Nil
	}
	var u uuid.UUID
	binary.BigEndian.PutUint64(u[8:], uint64(id))
	return u
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

	var (
		repositories []*entity.Repository
		err          error
	)

	switch {
	case input.OwnerID != uuid.Nil:
		repositories, err = u.repos.ListByOwner(ctx, input.OwnerID, page, perPage)
	case input.OwnerLogin != "" && u.users != nil:
		user, lookupErr := u.users.GetByLogin(ctx, input.OwnerLogin)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if user == nil {
			return []*entity.Repository{}, nil
		}
		repositories, err = u.repos.ListByOwner(ctx, int64ToUUID(user.ID), page, perPage)
	default:
		repositories, err = u.repos.ListByOrg(ctx, input.OrganizationID, page, perPage)
	}
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
