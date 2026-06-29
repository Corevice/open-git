package repository

import (
	"context"
	"encoding/binary"
	"errors"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
	"github.com/open-git/backend/internal/validator"
)

const MaxRepositoriesPerPage = 100

var ErrOwnerNotFound = errors.New("owner not found")

type repositoryListQuerier interface {
	CountByOrg(ctx context.Context, organizationID uuid.UUID) (int, error)
	CountVisibleByOrg(ctx context.Context, organizationID, viewerID uuid.UUID) (int, error)
	ListVisibleByOrg(ctx context.Context, organizationID, viewerID uuid.UUID, page, perPage int) ([]*entity.Repository, error)
}

func NormalizeRepositoryPagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	if perPage > MaxRepositoriesPerPage {
		perPage = MaxRepositoriesPerPage
	}
	return page, perPage
}

type ListRepositoriesInput struct {
	RequestUserID uuid.UUID
	OwnerLogin    string
	Page          int
	PerPage       int
}

type ListRepositoriesResult struct {
	Repositories []*entity.Repository
	Total        int
	OwnerLogin   string
}

type ListRepositoriesUsecase struct {
	repos       repo.IRepositoryRepository
	users       repo.IUserRepository
	memberships repo.IMembershipRepository
}

func NewListRepositoriesUsecase(
	repos repo.IRepositoryRepository,
	users repo.IUserRepository,
	memberships repo.IMembershipRepository,
) *ListRepositoriesUsecase {
	return &ListRepositoriesUsecase{repos: repos, users: users, memberships: memberships}
}

func (u *ListRepositoriesUsecase) Execute(ctx context.Context, input ListRepositoriesInput) (*ListRepositoriesResult, error) {
	owner, ownerLogin, orgID, err := u.resolveOwnerAndOrg(ctx, input)
	if err != nil {
		return nil, err
	}

	querier, ok := u.repos.(repositoryListQuerier)
	if !ok {
		return nil, errors.New("repository list querier not configured")
	}

	page, perPage := NormalizeRepositoryPagination(input.Page, input.PerPage)
	ownerUUID := middleware.Int64ToUUID(owner.ID)

	var (
		total     int
		pageRepos []*entity.Repository
	)

	if input.RequestUserID == ownerUUID {
		total, err = querier.CountByOrg(ctx, orgID)
		if err != nil {
			return nil, err
		}
		pageRepos, err = u.repos.ListByOrg(ctx, orgID, page, perPage)
	} else {
		total, err = querier.CountVisibleByOrg(ctx, orgID, input.RequestUserID)
		if err != nil {
			return nil, err
		}
		pageRepos, err = querier.ListVisibleByOrg(ctx, orgID, input.RequestUserID, page, perPage)
	}
	if err != nil {
		return nil, err
	}

	return &ListRepositoriesResult{
		Repositories: pageRepos,
		Total:        total,
		OwnerLogin:   ownerLogin,
	}, nil
}

func (u *ListRepositoriesUsecase) resolveOwnerAndOrg(ctx context.Context, input ListRepositoriesInput) (*domain.User, string, uuid.UUID, error) {
	var (
		owner *domain.User
		err   error
	)

	switch {
	case input.OwnerLogin != "":
		if err := validator.ValidateLogin(input.OwnerLogin); err != nil {
			return nil, "", uuid.Nil, ErrOwnerNotFound
		}
		owner, err = u.users.GetByLogin(ctx, input.OwnerLogin)
	case input.RequestUserID != uuid.Nil:
		owner, err = u.users.GetByID(ctx, uuidToInt64(input.RequestUserID))
	default:
		return nil, "", uuid.Nil, ErrOwnerNotFound
	}

	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, "", uuid.Nil, ErrOwnerNotFound
		}
		return nil, "", uuid.Nil, err
	}
	if owner == nil || owner.Login == "" {
		return nil, "", uuid.Nil, ErrOwnerNotFound
	}

	orgID := middleware.Int64ToUUID(owner.ID)
	return owner, owner.Login, orgID, nil
}

func uuidToInt64(id uuid.UUID) int64 {
	return int64(binary.BigEndian.Uint64(id[8:]))
}
