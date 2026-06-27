package database

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type txContextKey struct{}

type TxManager struct {
	DB *sqlx.DB
}

func NewTxManager(db *sqlx.DB) *TxManager {
	return &TxManager{DB: db}
}

func (m *TxManager) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx, err := m.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := context.WithValue(ctx, txContextKey{}, tx)
	if err := fn(txCtx); err != nil {
		return err
	}
	return tx.Commit()
}

type sqlxExtContext interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row
	QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error)
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
}

func SQLxExecutor(ctx context.Context, db *sqlx.DB) sqlxExtContext {
	if tx, ok := ctx.Value(txContextKey{}).(*sqlx.Tx); ok && tx != nil {
		return tx
	}
	return db
}
