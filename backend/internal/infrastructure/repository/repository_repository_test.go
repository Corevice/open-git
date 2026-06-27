package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

const repositoryTestSchema = `
CREATE TABLE organizations (
    id TEXT PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    plan_tier TEXT NOT NULL DEFAULT 'free',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE repositories (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    owner_id TEXT NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    visibility TEXT NOT NULL DEFAULT 'private',
    default_branch TEXT NOT NULL DEFAULT 'main',
    description TEXT NOT NULL DEFAULT '',
    disk_path TEXT NOT NULL DEFAULT '',
    is_empty INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(owner_id, name)
);
`

func newRepositoryTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(repositoryTestSchema); err != nil {
		_ = db.Close()
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return sqlx.NewDb(db, "sqlite3")
}

func seedRepositoryFixtures(t *testing.T, db *sqlx.DB, orgID, ownerID, repoID uuid.UUID, ownerLogin, repoName string) {
	t.Helper()

	now := time.Now().UTC()
	if _, err := db.Exec(
		`INSERT INTO organizations (id, login, name, plan_tier, created_at) VALUES (?, ?, ?, ?, ?)`,
		orgID.String(), "acme", "Acme", "free", now,
	); err != nil {
		t.Fatalf("insert organization: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO users (id, login, email, password_hash, created_at) VALUES (?, ?, ?, ?, ?)`,
		ownerID.String(), ownerLogin, ownerLogin+"@example.com", "hash", now,
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO repositories (id, organization_id, owner_id, name, visibility, default_branch, description, disk_path, is_empty, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		repoID.String(), orgID.String(), ownerID.String(), repoName, entity.VisibilityPrivate, "main", "demo repo", "", 1, now,
	); err != nil {
		t.Fatalf("insert repository: %v", err)
	}
}

func TestListByOrgIncludesOrganizationFilter(t *testing.T) {
	// Ensures ListByOrg scopes results to the requested organization_id.
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

	mock.ExpectQuery(`WHERE\s+organization_id\s*=\s*\$1`).
		WithArgs(orgID, 30, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "owner_id", "name", "visibility", "default_branch", "description", "disk_path", "is_empty", "created_at",
		}).AddRow(repoID, orgID, ownerID, "demo", "private", "main", "", "", 1, now))

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

	mock.ExpectQuery(regexp.QuoteMeta(`LIMIT $2 OFFSET $3`)).
		WithArgs(orgID, 10, 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "owner_id", "name", "visibility", "default_branch", "description", "disk_path", "is_empty", "created_at",
		}))

	if _, err := repo.ListByOrg(context.Background(), orgID, 2, 10); err != nil {
		t.Fatalf("ListByOrg: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestGetByOwnerLoginAndName_NotFound(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := repository.NewRepositoryRepository(db)

	found, err := repo.GetByOwnerLoginAndName(context.Background(), "missing-user", "missing-repo")
	if err != nil {
		t.Fatalf("GetByOwnerLoginAndName: %v", err)
	}
	if found != nil {
		t.Fatalf("expected nil repository, got %+v", found)
	}
}

func TestGetByOwnerLoginAndName_Found(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := repository.NewRepositoryRepository(db)

	orgID := uuid.New()
	ownerID := uuid.New()
	repoID := uuid.New()
	seedRepositoryFixtures(t, db, orgID, ownerID, repoID, "alice", "demo")

	found, err := repo.GetByOwnerLoginAndName(context.Background(), "alice", "demo")
	if err != nil {
		t.Fatalf("GetByOwnerLoginAndName: %v", err)
	}
	if found == nil {
		t.Fatal("expected repository, got nil")
	}
	if found.ID != repoID {
		t.Fatalf("expected repo id %s, got %s", repoID, found.ID)
	}
	if found.Description != "demo repo" {
		t.Fatalf("expected description %q, got %q", "demo repo", found.Description)
	}
	if !found.IsEmpty {
		t.Fatal("expected repository to be empty")
	}
	if found.DiskPath != "" {
		t.Fatalf("expected disk_path to be omitted, got %q", found.DiskPath)
	}
}

func TestUpdateDiskPath(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := repository.NewRepositoryRepository(db)

	orgID := uuid.New()
	ownerID := uuid.New()
	repoID := uuid.New()
	seedRepositoryFixtures(t, db, orgID, ownerID, repoID, "alice", "demo")

	const diskPath = "/data/alice/demo.git"
	if err := repo.UpdateDiskPath(context.Background(), repoID, diskPath); err != nil {
		t.Fatalf("UpdateDiskPath: %v", err)
	}

	var storedPath string
	if err := db.Get(&storedPath, `SELECT disk_path FROM repositories WHERE id = ?`, repoID.String()); err != nil {
		t.Fatalf("select disk_path: %v", err)
	}
	if storedPath != diskPath {
		t.Fatalf("expected disk_path %q, got %q", diskPath, storedPath)
	}
}

func TestUpdateDiskPath_NotFound(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := repository.NewRepositoryRepository(db)

	err := repo.UpdateDiskPath(context.Background(), uuid.New(), "/data/missing.git")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSetIsEmpty(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := repository.NewRepositoryRepository(db)

	orgID := uuid.New()
	ownerID := uuid.New()
	repoID := uuid.New()
	seedRepositoryFixtures(t, db, orgID, ownerID, repoID, "alice", "demo")

	if err := repo.SetIsEmpty(context.Background(), repoID, false); err != nil {
		t.Fatalf("SetIsEmpty: %v", err)
	}

	var isEmpty int
	if err := db.Get(&isEmpty, `SELECT is_empty FROM repositories WHERE id = ?`, repoID.String()); err != nil {
		t.Fatalf("select is_empty: %v", err)
	}
	if isEmpty != 0 {
		t.Fatalf("expected is_empty 0, got %d", isEmpty)
	}
}

func TestSetIsEmpty_NotFound(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := repository.NewRepositoryRepository(db)

	err := repo.SetIsEmpty(context.Background(), uuid.New(), false)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
