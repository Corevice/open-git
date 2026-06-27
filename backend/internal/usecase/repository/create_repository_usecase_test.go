package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/usecase/repository"
)

type mockRepositoryRepo struct {
	byOwnerAndName map[string]*domain.Repository
	created        []*domain.Repository
	createErr      error
	nextNumber     int64
}

func repoKey(ownerID int64, name string) string {
	return fmt.Sprintf("%d:%s", ownerID, name)
}

func (m *mockRepositoryRepo) Create(_ context.Context, repo *domain.Repository) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.byOwnerAndName == nil {
		m.byOwnerAndName = map[string]*domain.Repository{}
	}
	key := repoKey(repo.OwnerID, repo.Name)
	if _, exists := m.byOwnerAndName[key]; exists {
		return errors.New("duplicate")
	}
	m.created = append(m.created, repo)
	repo.ID = int64(len(m.created))
	m.byOwnerAndName[key] = repo
	return nil
}

func (m *mockRepositoryRepo) GetByOwnerAndName(_ context.Context, ownerID int64, name string) (*domain.Repository, error) {
	if m.byOwnerAndName == nil {
		return nil, errors.New("not found")
	}
	if repo, ok := m.byOwnerAndName[repoKey(ownerID, name)]; ok {
		return repo, nil
	}
	return nil, errors.New("not found")
}

func (m *mockRepositoryRepo) GetByOwnerLoginAndName(context.Context, string, string) (*domain.Repository, error) {
	return nil, errors.New("not found")
}

func (m *mockRepositoryRepo) ListByOrg(context.Context, int64) ([]*domain.Repository, error) {
	return nil, nil
}

func (m *mockRepositoryRepo) UpdateVisibility(context.Context, int64, domain.Visibility) error {
	return nil
}

func (m *mockRepositoryRepo) UpdateDiskPath(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}

func (m *mockRepositoryRepo) SetIsEmpty(context.Context, uuid.UUID, uuid.UUID, bool) error {
	return nil
}

func (m *mockRepositoryRepo) Delete(context.Context, int64) error {
	return nil
}

func (m *mockRepositoryRepo) NextNumber(context.Context, int64) (int64, error) {
	m.nextNumber++
	return m.nextNumber, nil
}

func TestDuplicateName(t *testing.T) {
	repos := &mockRepositoryRepo{
		byOwnerAndName: map[string]*domain.Repository{
			repoKey(1, "existing"): {OwnerID: 1, Name: "existing"},
		},
	}
	uc := repository.NewCreateRepositoryUsecase(repos)

	_, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID:        1,
		OrganizationID: 1,
		Name:           "existing",
	})
	if !errors.Is(err, repository.ErrDuplicateName) {
		t.Fatalf("expected ErrDuplicateName, got %v", err)
	}
}

func TestInvalidName(t *testing.T) {
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*domain.Repository{}}
	uc := repository.NewCreateRepositoryUsecase(repos)

	_, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID:        1,
		OrganizationID: 1,
		Name:           "invalid name!",
	})
	if !errors.Is(err, repository.ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}

func TestValidCreate(t *testing.T) {
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*domain.Repository{}}
	uc := repository.NewCreateRepositoryUsecase(repos)

	repo, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID:        1,
		OrganizationID: 1,
		Name:           "my-repo",
		Private:        true,
		Description:    "test repo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.Name != "my-repo" {
		t.Fatalf("expected name my-repo, got %s", repo.Name)
	}
	if repo.Visibility != domain.VisibilityPrivate {
		t.Fatal("expected private visibility")
	}
	if repo.DefaultBranch != "main" {
		t.Fatalf("expected default branch main, got %s", repo.DefaultBranch)
	}
	if len(repos.created) != 1 {
		t.Fatal("expected repository to be created")
	}
	if repos.nextNumber != 1 {
		t.Fatal("expected NextNumber to be called for default branch setup")
	}
}
