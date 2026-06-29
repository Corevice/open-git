package repository_test

import (
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/infrastructure/database"
)

func openTestDB(t *testing.T) *sqlx.DB {
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
