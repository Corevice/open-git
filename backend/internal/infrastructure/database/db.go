package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Corevice/open-git/backend/internal/config"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func Connect(cfg config.Config) (*sql.DB, error) {
	switch cfg.DBType {
	case "sqlite":
		dsn := cfg.DBDSN
		if dsn == "" {
			dsn = "./data/open-git.db"
		}
		if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
			return nil, fmt.Errorf("create data directory: %w", err)
		}
		db, err := sql.Open("sqlite3", dsn)
		if err != nil {
			return nil, fmt.Errorf("open sqlite: %w", err)
		}
		return db, nil
	case "postgres":
		if cfg.DBDSN == "" {
			return nil, fmt.Errorf("DB_DSN is required for postgres")
		}
		db, err := sql.Open("postgres", cfg.DBDSN)
		if err != nil {
			return nil, fmt.Errorf("open postgres: %w", err)
		}
		return db, nil
	default:
		return nil, fmt.Errorf("unknown DB_TYPE: %s", cfg.DBType)
	}
}
