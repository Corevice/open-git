package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
)

const (
	MaxRepositoryPerPage  = 100
	maxRepositoriesPerOrg   = 1000
	listRepositoriesBatch = 100
)

var ErrOwnerNotFound = errors.New("owner not found")

func NormalizeRepositoryPagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	if perPage > MaxRepositoryPerPage {
		perPage = MaxRepositoryPerPage
	}
	return page, perPage
}

type ListRepositoriesInput struct {
	RequestUserID  uuid.UUID
	OwnerLogin     string
	OrganizationID uuid.UUID
	Page           int
	PerPage        int
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
	owner, ownerLogin, err := u.resolveOwner(ctx, input)
	if err != nil {
		return nil, err
	}

	orgID := input.OrganizationID
	if orgID == uuid.Nil {
		orgID = middleware.Int64ToUUID(owner.ID)
	}

	page, perPage := NormalizeRepositoryPagination(input.Page, input.PerPage)

	targetStart := (page - 1) * perPage
	targetEnd := targetStart + perPage
	pageRepos := make([]*entity.Repository, 0, perPage)
	total := 0
	dbPage := 1
	membershipAccess := make(map[uuid.UUID]bool)

	for dbPage <= (maxRepositoriesPerOrg+listRepositoriesBatch-1)/listRepositoriesBatch {
		batch, err := u.repos.ListByOrg(ctx, orgID, dbPage, listRepositoriesBatch)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}

		for _, r := range batch {
			if !u.canViewRepository(ctx, input.RequestUserID, r, membershipAccess) {
				continue
			}
			if total >= targetStart && total < targetEnd {
				pageRepos = append(pageRepos, r)
			}
			total++
		}

		if len(batch) < listRepositoriesBatch {
			break
		}
		dbPage++
	}

	return &ListRepositoriesResult{
		Repositories: pageRepos,
		Total:        total,
		OwnerLogin:   ownerLogin,
	}, nil
}

func (u *ListRepositoriesUsecase) resolveOwner(ctx context.Context, input ListRepositoriesInput) (*domain.User, string, error) {
	if input.OwnerLogin == "" {
		return nil, "", ErrOwnerNotFound
	}

	owner, err := u.users.GetByLogin(ctx, input.OwnerLogin)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, "", ErrOwnerNotFound
		}
		return nil, "", err
	}
	if owner == nil {
		return nil, "", ErrOwnerNotFound
	}
	return owner, owner.Login, nil
}

func (u *ListRepositoriesUsecase) canViewRepository(
	ctx context.Context,
	requestUserID uuid.UUID,
	r *entity.Repository,
	membershipAccess map[uuid.UUID]bool,
) bool {
	if r.Visibility != entity.VisibilityPrivate {
		return true
	}
	if requestUserID == uuid.Nil {
		return false
	}
	if requestUserID == r.OwnerID {
		return true
	}
	if requestUserID == r.OrganizationID {
		return true
	}
	if allowed, ok := membershipAccess[r.OrganizationID]; ok {
		return allowed
	}
	hasAccess, err := u.memberships.HasReadAccess(ctx, requestUserID, r.OrganizationID)
	allowed := err == nil && hasAccess
	membershipAccess[r.OrganizationID] = allowed
	return allowed
}
