package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(db *sql.DB, dbType, path string) error {
	var (
		driver database.Driver
		name   string
		err    error
	)

	switch dbType {
	case "sqlite":
		name = "sqlite3"
		driver, err = sqlite3.WithInstance(db, &sqlite3.Config{})
	case "postgres":
		name = "postgres"
		driver, err = postgres.WithInstance(db, &postgres.Config{})
	default:
		return fmt.Errorf("unknown DB_TYPE: %s", dbType)
	}
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+path,
		name,
		driver,
	)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
