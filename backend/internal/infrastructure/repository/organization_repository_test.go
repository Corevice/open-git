package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/database"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func newOrganizationTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.RunMigrations(db, "sqlite", "../../../migrations"); err != nil {
		_ = db.Close()
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func TestOrganizationRepository_CreateGetByLogin(t *testing.T) {
	db := newOrganizationTestDB(t)
	repo := repository.NewOrganizationRepository(db)

	org := &entity.Organization{
		Login: "acme-corp",
		Name:  "Acme Corp",
	}
	if err := repo.Create(context.Background(), org); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByLogin(context.Background(), "acme-corp")
	if err != nil {
		t.Fatalf("GetByLogin: %v", err)
	}
	if got == nil {
		t.Fatal("expected organization, got nil")
	}
	if got.ID != org.ID || got.Login != org.Login || got.Name != org.Name {
		t.Fatalf("unexpected organization: %+v", got)
	}
}

func TestOrganizationRepository_ListPagination(t *testing.T) {
	db := newOrganizationTestDB(t)
	repo := repository.NewOrganizationRepository(db)

	for i, login := range []string{"org-alpha", "org-beta", "org-gamma"} {
		org := &entity.Organization{
			Login: login,
			Name:  login,
		}
		if err := repo.Create(context.Background(), org); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}

	page1, err := repo.List(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page 1: expected 2 orgs, got %d", len(page1))
	}

	page2, err := repo.List(context.Background(), 2, 2)
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("page 2: expected 1 org, got %d", len(page2))
	}
}

func TestOrganizationRepository_Delete(t *testing.T) {
	db := newOrganizationTestDB(t)
	repo := repository.NewOrganizationRepository(db)

	org := &entity.Organization{
		Login: "to-delete",
		Name:  "To Delete",
	}
	if err := repo.Create(context.Background(), org); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(context.Background(), org.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := repo.GetByID(context.Background(), org.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got err=%v org=%+v", err, got)
	}
}

func TestOrganizationRepository_DuplicateLoginConflict(t *testing.T) {
	db := newOrganizationTestDB(t)
	repo := repository.NewOrganizationRepository(db)

	first := &entity.Organization{
		Login: "same-login",
		Name:  "First Org",
	}
	if err := repo.Create(context.Background(), first); err != nil {
		t.Fatalf("Create first: %v", err)
	}

	second := &entity.Organization{
		Login: "same-login",
		Name:  "Second Org",
	}
	err := repo.Create(context.Background(), second)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}
