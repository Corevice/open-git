package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/database"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func newUserTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.RunMigrations(db, "sqlite", "../../migrations"); err != nil {
		_ = db.Close()
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func TestUserRepository_CreateGetByID(t *testing.T) {
	db := newUserTestDB(t)
	repo := repository.NewUserRepository(db)

	user := &entity.User{
		Login:        "alice",
		Email:        "alice@example.com",
		PasswordHash: "hashed",
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.ID != user.ID || got.Login != user.Login || got.Email != user.Email {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestUserRepository_GetByLogin(t *testing.T) {
	db := newUserTestDB(t)
	repo := repository.NewUserRepository(db)

	user := &entity.User{
		Login:        "bob",
		Email:        "bob@example.com",
		PasswordHash: "hashed",
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByLogin(context.Background(), "bob")
	if err != nil {
		t.Fatalf("GetByLogin: %v", err)
	}
	if got == nil || got.Login != "bob" {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := newUserTestDB(t)
	repo := repository.NewUserRepository(db)

	user := &entity.User{
		Login:        "carol",
		Email:        "carol@example.com",
		PasswordHash: "hashed",
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByEmail(context.Background(), "carol@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got == nil || got.Email != "carol@example.com" {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestUserRepository_DuplicateLoginConflict(t *testing.T) {
	db := newUserTestDB(t)
	repo := repository.NewUserRepository(db)

	first := &entity.User{
		Login:        "dupe",
		Email:        "first@example.com",
		PasswordHash: "hashed",
	}
	if err := repo.Create(context.Background(), first); err != nil {
		t.Fatalf("Create first: %v", err)
	}

	second := &entity.User{
		Login:        "dupe",
		Email:        "second@example.com",
		PasswordHash: "hashed",
	}
	err := repo.Create(context.Background(), second)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := newUserTestDB(t)
	repo := repository.NewUserRepository(db)

	user := &entity.User{
		Login:        "profile-user",
		Email:        "profile@example.com",
		PasswordHash: "hashed",
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	user.Name = "Profile Name"
	user.Bio = "A short bio"
	user.AvatarURL = "https://example.com/avatar.png"
	if err := repo.Update(context.Background(), user); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Profile Name" {
		t.Fatalf("Name: got %q, want %q", got.Name, "Profile Name")
	}
	if got.Bio != "A short bio" {
		t.Fatalf("Bio: got %q, want %q", got.Bio, "A short bio")
	}
	if got.AvatarURL != "https://example.com/avatar.png" {
		t.Fatalf("AvatarURL: got %q, want %q", got.AvatarURL, "https://example.com/avatar.png")
	}
	if got.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}
}

func TestUserRepository_GetByIDNotFound(t *testing.T) {
	db := newUserTestDB(t)
	repo := repository.NewUserRepository(db)

	got, err := repo.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got err=%v user=%+v", err, got)
	}
	if got != nil {
		t.Fatalf("expected nil user, got %+v", got)
	}
}
