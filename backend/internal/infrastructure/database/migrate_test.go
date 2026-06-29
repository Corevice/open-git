package database

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

const sqliteMigrationsPath = "../../../migrations"

func TestRunMigrations_SQLiteIdempotency(t *testing.T) {
	// Shared in-memory DSN so a second connection sees the same database.
	dsn := "file:migrate_test?mode=memory&cache=shared"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := RunMigrations(db, "sqlite", sqliteMigrationsPath); err != nil {
		t.Fatalf("first RunMigrations: %v", err)
	}

	if err := RunMigrations(db, "sqlite", sqliteMigrationsPath); err != nil {
		t.Fatalf("second RunMigrations (expected ErrNoChange swallowed): %v", err)
	}

	db2, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open second sqlite connection: %v", err)
	}
	t.Cleanup(func() { _ = db2.Close() })

	driver, err := sqlite3.WithInstance(db2, &sqlite3.Config{})
	if err != nil {
		t.Fatalf("create sqlite3 migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+sqliteMigrationsPath,
		"sqlite3",
		driver,
	)
	if err != nil {
		t.Fatalf("create migrator: %v", err)
	}
	defer m.Close()

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("down migrations: %v", err)
	}

	if err := RunMigrations(db2, "sqlite", sqliteMigrationsPath); err != nil {
		t.Fatalf("RunMigrations after down: %v", err)
	}
}
