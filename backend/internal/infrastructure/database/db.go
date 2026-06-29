package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/open-git/backend/internal/config"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func Connect(cfg config.Config) (*sql.DB, error) {
	var db *sql.DB
	var err error

	switch cfg.DBType {
	case "sqlite":
		dsn := cfg.DBDSN
		if dsn == "" {
			dsn = "./data/open-git.db"
		}
		if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
			return nil, fmt.Errorf("create data directory: %w", err)
		}
		db, err = sql.Open("sqlite3", dsn)
		if err != nil {
			return nil, fmt.Errorf("open sqlite: %w", err)
		}
		if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;"); err != nil {
			db.Close()
			return nil, fmt.Errorf("configure sqlite: %w", err)
		}
	case "postgres":
		if cfg.DBDSN == "" {
			return nil, fmt.Errorf("DB_DSN is required for postgres")
		}
		db, err = sql.Open("postgres", cfg.DBDSN)
		if err != nil {
			return nil, fmt.Errorf("open postgres: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown DB_TYPE: %s", cfg.DBType)
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)

	return db, nil
}

func Ping(ctx context.Context, db *sql.DB) error {
	return db.PingContext(ctx)
}

func MaskDSN(dsn string) string {
	if dsn == "" {
		return ""
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	if u.User != nil {
		if _, ok := u.User.Password(); ok {
			return u.Redacted()
		}
	}
	return dsn
}
