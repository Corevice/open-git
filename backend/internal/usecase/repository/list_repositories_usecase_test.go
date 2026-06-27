package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/usecase/repository"
)

type listMockRepositoryRepo struct {
	repos []*entity.Repository
}

func (m *listMockRepositoryRepo) Create(context.Context, *entity.Repository) error {
	return nil
}

func (m *listMockRepositoryRepo) GetByOwnerAndName(context.Context, uuid.UUID, string) (*entity.Repository, error) {
	return nil, errors.New("not found")
}

func (m *listMockRepositoryRepo) GetByOwnerLoginAndName(context.Context, string, string) (*entity.Repository, error) {
	return nil, errors.New("not found")
}

func (m *listMockRepositoryRepo) ListByOrg(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.Repository, error) {
	return m.repos, nil
}

func (m *listMockRepositoryRepo) UpdateVisibility(context.Context, uuid.UUID, string) error {
	return nil
}

func (m *listMockRepositoryRepo) UpdateDiskPath(context.Context, uuid.UUID, string) error {
	return nil
}

func (m *listMockRepositoryRepo) Delete(context.Context, uuid.UUID) error {
	return nil
}

type listMockUserRepo struct {
	users map[string]*domain.User
}

func (m *listMockUserRepo) Create(context.Context, *domain.User) error {
	return nil
}

func (m *listMockUserRepo) GetByID(context.Context, int64) (*domain.User, error) {
	return nil, errors.New("not found")
}

func (m *listMockUserRepo) GetByLogin(_ context.Context, login string) (*domain.User, error) {
	if m.users == nil {
		return nil, errors.New("not found")
	}
	if user, ok := m.users[login]; ok {
		return user, nil
	}
	return nil, errors.New("not found")
}

func (m *listMockUserRepo) GetByEmail(context.Context, string) (*domain.User, error) {
	return nil, errors.New("not found")
}

type listMockMembershipRepo struct{}

func (m *listMockMembershipRepo) HasReadAccess(_ context.Context, userID, _ uuid.UUID) (bool, error) {
	return userID != uuid.Nil, nil
}

func (m *listMockMembershipRepo) HasWriteAccess(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

func TestListRepositoriesUsecase_FiltersPrivate(t *testing.T) {
	ownerID := testOwnerID
	publicRepo := &entity.Repository{
		ID:             uuid.MustParse("00000000-0000-0000-0000-000000000010"),
		OrganizationID: testOrgID,
		OwnerID:        ownerID,
		Name:           "public-repo",
		Visibility:     entity.VisibilityPublic,
	}
	privateRepo := &entity.Repository{
		ID:             uuid.MustParse("00000000-0000-0000-0000-000000000011"),
		OrganizationID: testOrgID,
		OwnerID:        ownerID,
		Name:           "private-repo",
		Visibility:     entity.VisibilityPrivate,
	}

	repos := &listMockRepositoryRepo{
		repos: []*entity.Repository{publicRepo, privateRepo},
	}
	users := &listMockUserRepo{
		users: map[string]*domain.User{
			"alice": {ID: 1, Login: "alice"},
		},
	}
	uc := repository.NewListRepositoriesUsecase(repos, users, &listMockMembershipRepo{})

	result, err := uc.Execute(context.Background(), repository.ListRepositoriesInput{
		RequestUserID: uuid.Nil,
		OwnerLogin:    "alice",
		Page:          1,
		PerPage:       30,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if len(result.Repositories) != 1 {
		t.Fatalf("expected 1 visible repository, got %d", len(result.Repositories))
	}
	if result.Repositories[0].Name != "public-repo" {
		t.Fatalf("expected public-repo, got %s", result.Repositories[0].Name)
	}

	resultAsOwner, err := uc.Execute(context.Background(), repository.ListRepositoriesInput{
		RequestUserID: ownerID,
		OwnerLogin:    "alice",
		Page:          1,
		PerPage:       30,
	})
	if err != nil {
		t.Fatalf("unexpected error for owner: %v", err)
	}
	if resultAsOwner.Total != 2 {
		t.Fatalf("expected total 2 for owner, got %d", resultAsOwner.Total)
	}
	if len(resultAsOwner.Repositories) != 2 {
		t.Fatalf("expected 2 visible repositories for owner, got %d", len(resultAsOwner.Repositories))
	}
}
