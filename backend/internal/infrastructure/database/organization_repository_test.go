package database_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/open-git/backend/internal/infrastructure/database"
)

func newOrgMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, func()) {
	t.Helper()

	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	return mockDB, mock, func() { mockDB.Close() }
}

func TestGetByLoginNotFound(t *testing.T) {
	db, mock, closeFn := newOrgMock(t)
	defer closeFn()
	repo := database.NewOrganizationRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, login, name, created_at FROM organizations WHERE login = $1`,
	)).WithArgs("unknown").WillReturnError(sql.ErrNoRows)

	org, err := repo.GetByLogin(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if org != nil {
		t.Fatalf("expected nil org, got %+v", org)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestGetByLoginFound(t *testing.T) {
	db, mock, closeFn := newOrgMock(t)
	defer closeFn()
	repo := database.NewOrganizationRepository(db)

	now := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, login, name, created_at FROM organizations WHERE login = $1`,
	)).WithArgs("acme").WillReturnRows(sqlmock.NewRows([]string{
		"id", "login", "name", "created_at",
	}).AddRow(int64(1), "acme", "Acme Corp", now))

	org, err := repo.GetByLogin(context.Background(), "acme")
	if err != nil {
		t.Fatalf("GetByLogin: %v", err)
	}
	if org == nil {
		t.Fatal("expected org, got nil")
	}
	if org.ID != 1 || org.Login != "acme" || org.Name != "Acme Corp" {
		t.Fatalf("unexpected org: %+v", org)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestListByUserIDEmpty(t *testing.T) {
	db, mock, closeFn := newOrgMock(t)
	defer closeFn()
	repo := database.NewOrganizationRepository(db)

	mock.ExpectQuery(`SELECT o\.id, o\.login, o\.name, o\.created_at`).
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "login", "name", "created_at"}))

	orgs, err := repo.ListByUserID(context.Background(), 99)
	if err != nil {
		t.Fatalf("ListByUserID: %v", err)
	}
	if len(orgs) != 0 {
		t.Fatalf("expected empty slice, got %d orgs", len(orgs))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestGetMemberRoleNonMember(t *testing.T) {
	db, mock, closeFn := newOrgMock(t)
	defer closeFn()
	repo := database.NewOrganizationRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT role FROM memberships WHERE organization_id = $1 AND user_id = $2`,
	)).WithArgs(int64(1), int64(2)).WillReturnError(sql.ErrNoRows)

	role, err := repo.GetMemberRole(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("GetMemberRole: %v", err)
	}
	if role != "" {
		t.Fatalf("expected empty role, got %q", role)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestGetMemberRoleAdmin(t *testing.T) {
	db, mock, closeFn := newOrgMock(t)
	defer closeFn()
	repo := database.NewOrganizationRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT role FROM memberships WHERE organization_id = $1 AND user_id = $2`,
	)).WithArgs(int64(1), int64(2)).WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("admin"))

	role, err := repo.GetMemberRole(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("GetMemberRole: %v", err)
	}
	if role != "admin" {
		t.Fatalf("expected admin role, got %q", role)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
