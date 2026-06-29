package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/usecase/repository"
)

func repoLoginKey(ownerLogin, name string) string {
	return fmt.Sprintf("%s:%s", ownerLogin, name)
}

type getMockRepositoryRepo struct {
	reposByLogin map[string]*entity.Repository
}

func (m *getMockRepositoryRepo) Create(context.Context, *entity.Repository) error {
	return nil
}

func (m *getMockRepositoryRepo) GetByOwnerAndName(context.Context, uuid.UUID, string) (*entity.Repository, error) {
	return nil, errors.New("not found")
}

func (m *getMockRepositoryRepo) GetByOwnerLoginAndName(_ context.Context, ownerLogin, name string) (*entity.Repository, error) {
	if m.reposByLogin == nil {
		return nil, errors.New("not found")
	}
	if repo, ok := m.reposByLogin[repoLoginKey(ownerLogin, name)]; ok {
		return repo, nil
	}
	return nil, errors.New("not found")
}

func (m *getMockRepositoryRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *getMockRepositoryRepo) CountByOrg(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (m *getMockRepositoryRepo) ListByOwner(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *getMockRepositoryRepo) CountByOwner(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (m *getMockRepositoryRepo) UpdateVisibility(context.Context, uuid.UUID, string) error {
	return nil
}

func (m *getMockRepositoryRepo) UpdateName(context.Context, uuid.UUID, string) error {
	return nil
}

func (m *getMockRepositoryRepo) UpdateDefaultBranch(context.Context, uuid.UUID, string) error {
	return nil
}

func (m *getMockRepositoryRepo) Delete(context.Context, uuid.UUID) error {
	return nil
}

type getMockUserRepo struct {
	users map[string]*domain.User
}

func (m *getMockUserRepo) Create(context.Context, *domain.User) error {
	return nil
}

func (m *getMockUserRepo) GetByID(context.Context, int64) (*domain.User, error) {
	return nil, errors.New("not found")
}

func (m *getMockUserRepo) GetByLogin(_ context.Context, login string) (*domain.User, error) {
	if m.users == nil {
		return nil, errors.New("not found")
	}
	if user, ok := m.users[login]; ok {
		return user, nil
	}
	return nil, errors.New("not found")
}

func (m *getMockUserRepo) GetByEmail(context.Context, string) (*domain.User, error) {
	return nil, errors.New("not found")
}

type getMockMembershipRepo struct {
	readAccess map[uuid.UUID]bool
}

func (m *getMockMembershipRepo) HasReadAccess(_ context.Context, userID, _ uuid.UUID) (bool, error) {
	if m.readAccess == nil {
		return false, nil
	}
	return m.readAccess[userID], nil
}

func (m *getMockMembershipRepo) HasWriteAccess(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	return false, nil
}

func TestPrivateRepoNoAuth(t *testing.T) {
	repos := &getMockRepositoryRepo{
		reposByLogin: map[string]*entity.Repository{
			repoLoginKey("alice", "secret"): {
				ID:             uuid.MustParse("00000000-0000-0000-0000-000000000010"),
				OrganizationID: testOrgID,
				OwnerID:        testOwnerID,
				Name:           "secret",
				Visibility:     entity.VisibilityPrivate,
			},
		},
	}
	users := &getMockUserRepo{
		users: map[string]*domain.User{
			"alice": {ID: 1, Login: "alice"},
		},
	}
	uc := repository.NewGetRepositoryUsecase(repos, users, &getMockMembershipRepo{})

	_, err := uc.Execute(context.Background(), repository.GetRepositoryInput{
		RequestUserID: uuid.Nil,
		OwnerLogin:    "alice",
		Name:          "secret",
	})
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
