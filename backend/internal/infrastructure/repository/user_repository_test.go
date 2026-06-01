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

func newUserMock(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	return sqlxDB, mock, func() { mockDB.Close() }
}

func TestCreate(t *testing.T) {
	db, mock, closeFn := newUserMock(t)
	defer closeFn()
	repo := repository.NewUserRepository(db)

	user := &entity.User{
		ID:           uuid.New(),
		Login:        "alice",
		Email:        "alice@example.com",
		PasswordHash: "hashed",
		CreatedAt:    time.Now().UTC(),
	}

	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO users (id, login, email, password_hash, created_at)`,
	)).WithArgs(
		user.ID, user.Login, user.Email, user.PasswordHash, user.CreatedAt,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestGetByLoginNotFound(t *testing.T) {
	db, mock, closeFn := newUserMock(t)
	defer closeFn()
	repo := repository.NewUserRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, login, email, password_hash, created_at FROM users WHERE login =`,
	)).WithArgs("missing").WillReturnRows(sqlmock.NewRows([]string{
		"id", "login", "email", "password_hash", "created_at",
	}))

	got, err := repo.GetByLogin(context.Background(), "missing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil user, got %+v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestGetByLoginFound(t *testing.T) {
	db, mock, closeFn := newUserMock(t)
	defer closeFn()
	repo := repository.NewUserRepository(db)

	id := uuid.New()
	now := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, login, email, password_hash, created_at FROM users WHERE login =`,
	)).WithArgs("alice").WillReturnRows(sqlmock.NewRows([]string{
		"id", "login", "email", "password_hash", "created_at",
	}).AddRow(id, "alice", "alice@example.com", "hashed", now))

	got, err := repo.GetByLogin(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetByLogin: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.ID != id || got.Login != "alice" {
		t.Fatalf("unexpected user: %+v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
