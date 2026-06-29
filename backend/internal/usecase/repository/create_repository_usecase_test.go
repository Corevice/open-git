package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	gitinfra "github.com/open-git/backend/internal/infrastructure/git"
	"github.com/open-git/backend/internal/usecase/repository"
)

var (
	testOwnerID    = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testOrgID      = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testOwnerLogin = "testuser"
)

type mockRepositoryRepo struct {
	byOwnerAndName map[string]*entity.Repository
	created        []*entity.Repository
	createErr      error
	deletedIDs     []uuid.UUID
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

func (m *mockRepositoryRepo) ListByOwner(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *mockRepositoryRepo) CountByOwner(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockRepositoryRepo) CountByOrg(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockRepositoryRepo) UpdateVisibility(context.Context, uuid.UUID, string) error {
	return nil
}

func (m *mockRepositoryRepo) UpdateName(context.Context, uuid.UUID, string) error {
	return nil
}

func (m *mockRepositoryRepo) UpdateDefaultBranch(context.Context, uuid.UUID, string) error {
	return nil
}

func (m *mockRepositoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.deletedIDs = append(m.deletedIDs, id)
	if m.byOwnerAndName != nil {
		for key, repo := range m.byOwnerAndName {
			if repo.ID == id {
				delete(m.byOwnerAndName, key)
				break
			}
		}
	}
	return nil
}

type mockGitInitService struct {
	called   bool
	lastPath string
	lastOpts gitinfra.AutoInitOpts
	err      error
}

func (m *mockGitInitService) AutoInitRepository(bareRepoPath string, opts gitinfra.AutoInitOpts) error {
	m.called = true
	m.lastPath = bareRepoPath
	m.lastOpts = opts
	return m.err
}

func TestDuplicateName(t *testing.T) {
	repos := &mockRepositoryRepo{
		byOwnerAndName: map[string]*entity.Repository{
			repoKey(testOwnerID, "existing"): {OwnerID: testOwnerID, Name: "existing"},
		},
	}
	uc := repository.NewCreateRepositoryUsecase(repos, repository.WithGitDataRoot("/data/git"))

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
	uc := repository.NewCreateRepositoryUsecase(repos, repository.WithGitDataRoot("/data/git"))

	_, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID:        testOwnerID,
		OrganizationID: testOrgID,
		Name:           "invalid name!",
	})
	if !errors.Is(err, repository.ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}

func TestInvalidNameWithDotDot(t *testing.T) {
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*entity.Repository{}}
	uc := repository.NewCreateRepositoryUsecase(repos, repository.WithGitDataRoot("/data/git"))

	_, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID:        testOwnerID,
		OrganizationID: testOrgID,
		Name:           "..",
	})
	if !errors.Is(err, repository.ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}

func TestAutoInitRequiresOwnerLogin(t *testing.T) {
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*entity.Repository{}}
	gitInit := &mockGitInitService{}
	uc := repository.NewCreateRepositoryUsecase(repos, repository.WithGitDataRoot("/data/git"), repository.WithGitInitService(gitInit))

	_, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID:        testOwnerID,
		OrganizationID: testOrgID,
		Name:           "my-repo",
		AutoInit:       true,
	})
	if !errors.Is(err, repository.ErrOwnerLoginRequired) {
		t.Fatalf("expected ErrOwnerLoginRequired, got %v", err)
	}
	if gitInit.called {
		t.Fatal("expected git init service not to be called")
	}
	if len(repos.deletedIDs) != 1 {
		t.Fatalf("expected repository to be deleted on owner login error, got %d deletes", len(repos.deletedIDs))
	}
}

func TestAutoInitFailureDeletesRepository(t *testing.T) {
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*entity.Repository{}}
	gitInit := &mockGitInitService{err: errors.New("init failed")}
	uc := repository.NewCreateRepositoryUsecase(repos, repository.WithGitDataRoot("/data/git"), repository.WithGitInitService(gitInit))

	_, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID:        testOwnerID,
		OrganizationID: testOrgID,
		Name:           "my-repo",
		AutoInit:       true,
		OwnerLogin:     testOwnerLogin,
	})
	if err == nil {
		t.Fatal("expected auto init error")
	}
	if !gitInit.called {
		t.Fatal("expected git init service to be called")
	}
	if len(repos.deletedIDs) != 1 {
		t.Fatalf("expected repository to be deleted on auto init failure, got %d deletes", len(repos.deletedIDs))
	}
	if len(repos.created) != 1 {
		t.Fatal("expected one create attempt")
	}
}

func TestValidCreate(t *testing.T) {
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*entity.Repository{}}
	uc := repository.NewCreateRepositoryUsecase(repos, repository.WithGitDataRoot("/data/git"))

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

func TestAutoInitCallsGitInitService(t *testing.T) {
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*entity.Repository{}}
	gitInit := &mockGitInitService{}
	uc := repository.NewCreateRepositoryUsecase(repos, repository.WithGitDataRoot("/data/git"), repository.WithGitInitService(gitInit))

	repo, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID:           testOwnerID,
		OrganizationID:    testOrgID,
		Name:              "my-repo",
		AutoInit:          true,
		GitIgnoreTemplate: "Go",
		LicenseTemplate:   "mit",
		OwnerLogin:        testOwnerLogin,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gitInit.called {
		t.Fatal("expected git init service to be called")
	}
	if gitInit.lastPath != "/data/git/testuser/my-repo.git" {
		t.Fatalf("expected git path /data/git/testuser/my-repo.git, got %q", gitInit.lastPath)
	}
	if gitInit.lastOpts.Readme != "my-repo" {
		t.Fatalf("expected readme option my-repo, got %q", gitInit.lastOpts.Readme)
	}
	if gitInit.lastOpts.GitIgnoreTemplate != "Go" {
		t.Fatalf("expected gitignore template Go, got %q", gitInit.lastOpts.GitIgnoreTemplate)
	}
	if gitInit.lastOpts.LicenseTemplate != "mit" {
		t.Fatalf("expected license template mit, got %q", gitInit.lastOpts.LicenseTemplate)
	}
	if repo.GitPath != "/data/git/testuser/my-repo.git" {
		t.Fatalf("expected git path on repository, got %q", repo.GitPath)
	}
}

func TestAutoInitFalseDoesNotCallGitInitService(t *testing.T) {
	repos := &mockRepositoryRepo{byOwnerAndName: map[string]*entity.Repository{}}
	gitInit := &mockGitInitService{}
	uc := repository.NewCreateRepositoryUsecase(repos, repository.WithGitDataRoot("/data/git"), repository.WithGitInitService(gitInit))

	repo, err := uc.Execute(context.Background(), repository.CreateRepositoryInput{
		OwnerID:        testOwnerID,
		OrganizationID: testOrgID,
		Name:           "my-repo",
		AutoInit:       false,
		OwnerLogin:     testOwnerLogin,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gitInit.called {
		t.Fatal("expected git init service not to be called")
	}
	if repo.GitPath != "" {
		t.Fatalf("expected empty git path, got %q", repo.GitPath)
	}
}
