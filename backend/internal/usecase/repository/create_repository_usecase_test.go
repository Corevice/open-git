package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/usecase/repository"
)

var (
	testOwnerID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testOrgID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
)

type mockRepositoryRepo struct {
	byOwnerAndName map[string]*entity.Repository
	created        []*entity.Repository
	createErr      error
}

func repoKey(ownerID uuid.UUID, name string) string {
	return fmt.Sprintf("%s:%s", ownerID, name)
}

func (m *mockRepositoryRepo) Create(_ context.Context, repo *entity.Repository) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.byOwnerAndName == nil {
		m.byOwnerAndName = map[string]*entity.Repository{}
	}
	key := repoKey(repo.OwnerID, repo.Name)
	if _, exists := m.byOwnerAndName[key]; exists {
		return errors.New("duplicate")
	}
	m.created = append(m.created, repo)
	repo.ID = uuid.New()
	m.byOwnerAndName[key] = repo
	return nil
}

func (m *mockRepositoryRepo) GetByOwnerAndName(_ context.Context, ownerID uuid.UUID, name string) (*entity.Repository, error) {
	if m.byOwnerAndName == nil {
		return nil, errors.New("not found")
	}
	if repo, ok := m.byOwnerAndName[repoKey(ownerID, name)]; ok {
		return repo, nil
	}
	return nil, errors.New("not found")
}

func (m *mockRepositoryRepo) GetByOwnerLoginAndName(context.Context, string, string) (*entity.Repository, error) {
	return nil, errors.New("not found")
}

func (m *mockRepositoryRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *mockRepositoryRepo) UpdateVisibility(context.Context, uuid.UUID, string) error {
	return nil
}

func (m *mockRepositoryRepo) Delete(context.Context, uuid.UUID) error {
	return nil
}

func TestDuplicateName(t *testing.T) {
	repos := &mockRepositoryRepo{
		byOwnerAndName: map[string]*entity.Repository{
			repoKey(testOwnerID, "existing"): {OwnerID: testOwnerID, Name: "existing"},
		},
	}
	uc := repository.NewCreateRepositoryUsecase(repos)

	_, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID:        testOwnerID,
		OrganizationID: testOrgID,
		Name:           "existing",
	})
	if !errors.Is(err, repository.ErrDuplicateName) {
		t.Fatalf("expected ErrDuplicateName, got %v", err)
	}
}

func TestInvalidName(t *testing.T) {
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*entity.Repository{}}
	uc := repository.NewCreateRepositoryUsecase(repos)

	_, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID:        testOwnerID,
		OrganizationID: testOrgID,
		Name:           "invalid name!",
	})
	if !errors.Is(err, repository.ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}

func TestValidCreate(t *testing.T) {
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*entity.Repository{}}
	uc := repository.NewCreateRepositoryUsecase(repos)

	repo, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID:        testOwnerID,
		OrganizationID: testOrgID,
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
	if repo.Visibility != entity.VisibilityPrivate {
		t.Fatal("expected private visibility")
	}
	if repo.DefaultBranch != "main" {
		t.Fatalf("expected default branch main, got %s", repo.DefaultBranch)
	}
	if repo.OwnerID != testOwnerID {
		t.Fatalf("expected owner id %s, got %s", testOwnerID, repo.OwnerID)
	}
	if repo.OrganizationID != testOrgID {
		t.Fatalf("expected organization id %s, got %s", testOrgID, repo.OrganizationID)
	}
	if len(repos.created) != 1 {
		t.Fatal("expected repository to be created")
	}
}
