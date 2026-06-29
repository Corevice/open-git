package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	repo "github.com/open-git/backend/internal/repository"
	"github.com/open-git/backend/internal/validator"
)

const MaxRepositoriesPerPage = 100

var ErrOwnerNotFound = errors.New("owner not found")

type repositoryListQuerier interface {
	repo.IRepositoryRepository
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
	repos       repositoryListQuerier
	users       repo.IUserRepository
	memberships repo.IMembershipRepository
}

func NewListRepositoriesUsecase(
	repos repositoryListQuerier,
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

	page, perPage := NormalizeRepositoryPagination(input.Page, input.PerPage)
	ownerUUID := int64ToUserUUID(owner.ID)

	var (
		total     int
		pageRepos []*entity.Repository
	)

	if input.RequestUserID == ownerUUID {
		total, err = u.repos.CountByOrg(ctx, orgID)
		if err != nil {
			return nil, err
		}
		pageRepos, err = u.repos.ListByOrg(ctx, orgID, page, perPage)
	} else {
		total, err = u.repos.CountVisibleByOrg(ctx, orgID, input.RequestUserID)
		if err != nil {
			return nil, err
		}
		pageRepos, err = u.repos.ListVisibleByOrg(ctx, orgID, input.RequestUserID, page, perPage)
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
		ownerID, convErr := UserUUIDToInt64(input.RequestUserID)
		if convErr != nil {
			return nil, "", uuid.Nil, ErrOwnerNotFound
		}
		owner, err = u.users.GetByID(ctx, ownerID)
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

	orgID := int64ToUserUUID(owner.ID)
	return owner, owner.Login, orgID, nil
}
