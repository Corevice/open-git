package repository_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

var repositoryRowColumns = []string{
	"id", "organization_id", "owner_id", "name", "description", "git_path", "owner_login", "visibility", "default_branch", "created_at",
}

func TestListByOrgIncludesOrganizationFilter(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	repo := repository.NewRepositoryRepository(sqlxDB)

	orgID := uuid.New()
	repoID := uuid.New()
	ownerID := uuid.New()
	now := time.Now().UTC()

	// The query MUST filter by organization_id for multi-tenant isolation.
	mock.ExpectQuery(`WHERE\s+organization_id\s*=\s*\$1`).
		WithArgs(orgID, 30, 0).
		WillReturnRows(sqlmock.NewRows(repositoryRowColumns).
			AddRow(repoID, orgID, ownerID, "demo", "", "", "", "private", "main", now))

	repos, err := repo.ListByOrg(context.Background(), orgID, 1, 30)
	if err != nil {
		t.Fatalf("ListByOrg: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	if repos[0].OrganizationID != orgID {
		t.Fatalf("expected org %s, got %s", orgID, repos[0].OrganizationID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestListByOrgPagination(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	repo := repository.NewRepositoryRepository(sqlxDB)

	orgID := uuid.New()

	// page=2, perPage=10 → LIMIT 10 OFFSET 10
	mock.ExpectQuery(regexp.QuoteMeta(`LIMIT $2 OFFSET $3`)).
		WithArgs(orgID, 10, 10).
		WillReturnRows(sqlmock.NewRows(repositoryRowColumns))

	if _, err := repo.ListByOrg(context.Background(), orgID, 2, 10); err != nil {
		t.Fatalf("ListByOrg: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestCreateAndGetByOwnerLoginAndNameReturnsOwnerLoginAndDescription(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	repo := repository.NewRepositoryRepository(sqlxDB)

	repoID := uuid.New()
	orgID := uuid.New()
	ownerID := uuid.New()
	now := time.Now().UTC()
	ownerLogin := "testuser"
	description := "A test repository"
	gitPath := "/data/git/testuser/myrepo.git"

	mock.ExpectExec(`INSERT INTO repositories`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(`WHERE\s+owner_login\s*=\s*\$1`).
		WithArgs(ownerLogin, "myrepo").
		WillReturnRows(sqlmock.NewRows(repositoryRowColumns).
			AddRow(repoID, orgID, ownerID, "myrepo", description, gitPath, ownerLogin, "private", "main", now))

	repoEntity := &entity.Repository{
		ID:             repoID,
		OrganizationID: orgID,
		OwnerID:        ownerID,
		Name:           "myrepo",
		Description:    description,
		GitPath:        gitPath,
		OwnerLogin:     ownerLogin,
		Visibility:     "private",
		DefaultBranch:  "main",
		CreatedAt:      now,
	}

	if err := repo.Create(context.Background(), repoEntity); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByOwnerLoginAndName(context.Background(), ownerLogin, "myrepo")
	if err != nil {
		t.Fatalf("GetByOwnerLoginAndName: %v", err)
	}
	if got == nil {
		t.Fatal("expected repository, got nil")
	}
	if got.OwnerLogin != ownerLogin {
		t.Fatalf("expected owner_login %q, got %q", ownerLogin, got.OwnerLogin)
	}
	if got.Description != description {
		t.Fatalf("expected description %q, got %q", description, got.Description)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
