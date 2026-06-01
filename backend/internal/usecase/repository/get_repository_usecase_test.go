package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/usecase/repository"
)

type getMockRepositoryRepo struct {
	repos map[string]*domain.Repository
}

func (m *getMockRepositoryRepo) Create(context.Context, *domain.Repository) error {
	return nil
}

func (m *getMockRepositoryRepo) GetByOwnerAndName(_ context.Context, ownerID int64, name string) (*domain.Repository, error) {
	if m.repos == nil {
		return nil, errors.New("not found")
	}
	if repo, ok := m.repos[repoKey(ownerID, name)]; ok {
		return repo, nil
	}
	return nil, errors.New("not found")
}

func (m *getMockRepositoryRepo) GetByOwnerLoginAndName(context.Context, string, string) (*domain.Repository, error) {
	return nil, errors.New("not found")
}

func (m *getMockRepositoryRepo) ListByOrg(context.Context, int64) ([]*domain.Repository, error) {
	return nil, nil
}

func (m *getMockRepositoryRepo) UpdateVisibility(context.Context, int64, domain.Visibility) error {
	return nil
}

func (m *getMockRepositoryRepo) Delete(context.Context, int64) error {
	return nil
}

type getMockUserRepo struct {
	users map[string]*domain.User
}

func (m *getMockUserRepo) Create(context.Context, *domain.User) error {
	return nil
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
	readAccess map[int64]bool
}

func (m *getMockMembershipRepo) HasReadAccess(_ context.Context, userID, _ int64) (bool, error) {
	if m.readAccess == nil {
		return false, nil
	}
	return m.readAccess[userID], nil
}

func TestPrivateRepoNoAuth(t *testing.T) {
	repos := &getMockRepositoryRepo{
		repos: map[string]*domain.Repository{
			repoKey(1, "secret"): {
				ID:             10,
				OrganizationID: 1,
				OwnerID:        1,
				Name:           "secret",
				Visibility:     domain.VisibilityPrivate,
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
		RequestUserID: 0,
		OwnerLogin:    "alice",
		Name:          "secret",
	})
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
