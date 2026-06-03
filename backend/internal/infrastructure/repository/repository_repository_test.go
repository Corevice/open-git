package repository_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/Corevice/open-git/backend/internal/infrastructure/repository"
)

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
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "owner_id", "name", "visibility", "default_branch", "created_at",
		}).AddRow(repoID, orgID, ownerID, "demo", "private", "main", now))

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
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "owner_id", "name", "visibility", "default_branch", "created_at",
		}))

	if _, err := repo.ListByOrg(context.Background(), orgID, 2, 10); err != nil {
		t.Fatalf("ListByOrg: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
