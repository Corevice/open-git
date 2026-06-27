package repository

import (
	"context"
	"encoding/binary"
	"errors"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	repo "github.com/open-git/backend/internal/repository"
)

const userUUIDPrefixLen = 8

const (
	MaxRepositoryPerPage  = 100
	maxRepositoriesPerOrg   = 1000
	listRepositoriesBatch = 100
)

var ErrOwnerNotFound = errors.New("owner not found")

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
		if requestUserID, ok := userIDFromUUID(input.RequestUserID); ok && requestUserID == owner.ID {
			orgID = input.RequestUserID
		} else {
			orgID = uuidFromInt64(owner.ID)
		}
	}

	page := input.Page
	if page < 1 {
		page = 1
	}
	perPage := input.PerPage
	if perPage < 1 {
		perPage = 30
	}
	if perPage > MaxRepositoryPerPage {
		perPage = MaxRepositoryPerPage
	}

	targetStart := (page - 1) * perPage
	targetEnd := targetStart + perPage
	pageRepos := make([]*entity.Repository, 0, perPage)
	total := 0
	dbPage := 1

	for dbPage <= (maxRepositoriesPerOrg+listRepositoriesBatch-1)/listRepositoriesBatch {
		batch, err := u.repos.ListByOrg(ctx, orgID, dbPage, listRepositoriesBatch)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}

		for _, r := range batch {
			if !u.canViewRepository(ctx, input.RequestUserID, r) {
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
	if input.OwnerLogin != "" {
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

	if input.OrganizationID == uuid.Nil {
		return nil, "", ErrOwnerNotFound
	}

	userID, ok := userIDFromUUID(input.OrganizationID)
	if !ok {
		return nil, "", ErrOwnerNotFound
	}

	owner, err := u.users.GetByID(ctx, userID)
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

func (u *ListRepositoriesUsecase) canViewRepository(ctx context.Context, requestUserID uuid.UUID, r *entity.Repository) bool {
	if r.Visibility != entity.VisibilityPrivate {
		return true
	}
	if requestUserID == uuid.Nil {
		return false
	}
	hasAccess, err := u.memberships.HasReadAccess(ctx, requestUserID, r.OrganizationID)
	return err == nil && hasAccess
}

func isUserUUID(id uuid.UUID) bool {
	for i := 0; i < userUUIDPrefixLen; i++ {
		if id[i] != 0 {
			return false
		}
	}
	return true
}

func userIDFromUUID(id uuid.UUID) (int64, bool) {
	if id == uuid.Nil || !isUserUUID(id) {
		return 0, false
	}
	return int64(binary.BigEndian.Uint64(id[userUUIDPrefixLen:])), true
}

func int64FromUUID(id uuid.UUID) int64 {
	userID, ok := userIDFromUUID(id)
	if !ok {
		return 0
	}
	return userID
}

func uuidFromInt64(id int64) uuid.UUID {
	if id <= 0 {
		return uuid.Nil
	}
	var u uuid.UUID
	binary.BigEndian.PutUint64(u[userUUIDPrefixLen:], uint64(id))
	return u
}
