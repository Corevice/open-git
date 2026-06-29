package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func setupTxTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(`CREATE TABLE t(id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	return sqlx.NewDb(db, "sqlite3")
}

func TestWithinTx_CommitsOnSuccess(t *testing.T) {
	db := setupTxTestDB(t)
	mgr := NewTxManager(db)

	err := mgr.WithinTx(context.Background(), func(ctx context.Context) error {
		tx := TxFromContext(ctx)
		if tx == nil {
			t.Fatal("TxFromContext returned nil inside WithinTx")
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO t(id) VALUES (1)`)
		return err
	})
	if err != nil {
		t.Fatalf("WithinTx: %v", err)
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM t`); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("row count = %d, want 1", count)
	}
}

func TestWithinTx_RollsBackOnError(t *testing.T) {
	db := setupTxTestDB(t)
	mgr := NewTxManager(db)

	err := mgr.WithinTx(context.Background(), func(ctx context.Context) error {
		tx := TxFromContext(ctx)
		if tx == nil {
			t.Fatal("TxFromContext returned nil inside WithinTx")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO t(id) VALUES (1)`); err != nil {
			return err
		}
		return errors.New("rollback me")
	})
	if err == nil {
		t.Fatal("WithinTx: expected error, got nil")
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM t`); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("row count = %d, want 0", count)
	}
}
