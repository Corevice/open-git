package repository

import (
	"context"
	"encoding/binary"
	"errors"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	repo "github.com/open-git/backend/internal/repository"
)

var ErrOwnerNotFound = errors.New("owner not found")

type ListRepositoriesInput struct {
	RequestUserID uuid.UUID
	OwnerLogin    string
	Page          int
	PerPage       int
}

type ListRepositoriesUsecase struct {
	repos repo.IRepositoryRepository
	users repo.IUserRepository
}

func NewListRepositoriesUsecase(repos repo.IRepositoryRepository, users repo.IUserRepository) *ListRepositoriesUsecase {
	return &ListRepositoriesUsecase{repos: repos, users: users}
}

func (u *ListRepositoriesUsecase) Execute(ctx context.Context, input ListRepositoriesInput) ([]*entity.Repository, int, error) {
	owner, err := u.users.GetByLogin(ctx, input.OwnerLogin)
	if err != nil || owner == nil {
		return nil, 0, ErrOwnerNotFound
	}

	orgID := int64ToUUID(owner.ID)

	page := input.Page
	if page < 1 {
		page = 1
	}
	perPage := input.PerPage
	if perPage < 1 {
		perPage = 30
	}

	repositories, err := u.repos.ListByOrg(ctx, orgID, page, perPage)
	if err != nil {
		return nil, 0, err
	}

	isOwner := input.RequestUserID != uuid.Nil && input.RequestUserID == orgID

	visible := make([]*entity.Repository, 0, len(repositories))
	for _, r := range repositories {
		if r.Visibility != entity.VisibilityPrivate {
			visible = append(visible, r)
			continue
		}
		if isOwner || (input.RequestUserID != uuid.Nil && input.RequestUserID == r.OwnerID) {
			visible = append(visible, r)
		}
	}

	return visible, len(visible), nil
}

func int64ToUUID(id int64) uuid.UUID {
	if id == 0 {
		return uuid.Nil
	}
	var u uuid.UUID
	binary.BigEndian.PutUint64(u[8:], uint64(id))
	return u
}
