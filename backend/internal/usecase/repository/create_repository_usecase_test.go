package repository_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
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
	diskPaths      map[uuid.UUID]string
	deleted        []uuid.UUID
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

func (m *mockRepositoryRepo) UpdateDiskPath(_ context.Context, id uuid.UUID, diskPath string) error {
	if m.diskPaths == nil {
		m.diskPaths = map[uuid.UUID]string{}
	}
	m.diskPaths[id] = diskPath
	return nil
}

func (m *mockRepositoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.deleted = append(m.deleted, id)
	return nil
}

type mockUserRepo struct{}

func (m *mockUserRepo) Create(context.Context, *domain.User) error {
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id int64) (*domain.User, error) {
	if id == 1 {
		return &domain.User{ID: 1, Login: "alice"}, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) GetByLogin(_ context.Context, login string) (*domain.User, error) {
	if login == "alice" {
		return &domain.User{ID: 1, Login: "alice"}, nil
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) GetByEmail(context.Context, string) (*domain.User, error) {
	return nil, errors.New("not found")
}

func TestDuplicateName(t *testing.T) {
	repos := &mockRepositoryRepo{
		byOwnerAndName: map[string]*entity.Repository{
			repoKey(testOwnerID, "existing"): {OwnerID: testOwnerID, Name: "existing"},
		},
	}
	uc := repository.NewCreateRepositoryUsecase(repos, &mockUserRepo{}, t.TempDir())

	_, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID: testOwnerID,
		Name:    "existing",
	})
	if !errors.Is(err, repository.ErrDuplicateName) {
		t.Fatalf("expected ErrDuplicateName, got %v", err)
	}
}

func TestInvalidName(t *testing.T) {
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*entity.Repository{}}
	uc := repository.NewCreateRepositoryUsecase(repos, &mockUserRepo{}, t.TempDir())

	_, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID: testOwnerID,
		Name:    "invalid name!",
	})
	if !errors.Is(err, repository.ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}

func TestValidCreate(t *testing.T) {
	gitRoot := t.TempDir()
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*entity.Repository{}}
	uc := repository.NewCreateRepositoryUsecase(repos, &mockUserRepo{}, gitRoot)

	result, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID:     testOwnerID,
		Name:        "my-repo",
		Private:     true,
		Description: "test repo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	repo := result.Repository
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

func TestCreateRepositoryInitsBareRepo(t *testing.T) {
	gitRoot := t.TempDir()
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*entity.Repository{}}
	uc := repository.NewCreateRepositoryUsecase(repos, &mockUserRepo{}, gitRoot)

	ownerLogin := "alice"
	_, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID: testOwnerID,
		Name:    "my-repo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	diskPath := filepath.Join(gitRoot, ownerLogin, "my-repo.git")
	if _, err := os.Stat(diskPath); err != nil {
		t.Fatalf("expected bare repo directory at %s: %v", diskPath, err)
	}

	if _, err := gogit.PlainOpen(diskPath); err != nil {
		t.Fatalf("expected valid bare git repo: %v", err)
	}
}

func TestOwnerNotFound(t *testing.T) {
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*entity.Repository{}}
	uc := repository.NewCreateRepositoryUsecase(repos, &mockUserRepo{}, t.TempDir())

	unknownOwnerID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	_, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID: unknownOwnerID,
		Name:    "my-repo",
	})
	if !errors.Is(err, repository.ErrOwnerNotFound) {
		t.Fatalf("expected ErrOwnerNotFound, got %v", err)
	}
}

func TestCreateRepositoryRollbackOnInitBareFailure(t *testing.T) {
	gitRoot := t.TempDir()
	ownerLogin := "alice"
	blockedPath := filepath.Join(gitRoot, ownerLogin, "my-repo.git")
	if err := os.MkdirAll(filepath.Dir(blockedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blockedPath, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*entity.Repository{}}
	uc := repository.NewCreateRepositoryUsecase(repos, &mockUserRepo{}, gitRoot)

	_, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID: testOwnerID,
		Name:    "my-repo",
	})
	if err == nil {
		t.Fatal("expected error when bare repo init fails")
	}
	if len(repos.created) != 1 {
		t.Fatalf("expected one created repository before rollback, got %d", len(repos.created))
	}
	if len(repos.deleted) != 1 {
		t.Fatalf("expected DB rollback delete, got %d deletes", len(repos.deleted))
	}
	if repos.deleted[0] != repos.created[0].ID {
		t.Fatalf("expected rollback to delete created repository id %s, got %s", repos.created[0].ID, repos.deleted[0])
	}
}
