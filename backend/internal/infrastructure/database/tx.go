package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type txContextKey struct{}

var txKey txContextKey

type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type sqlxTxManager struct {
	db *sqlx.DB
}

func NewTxManager(db *sqlx.DB) TxManager {
	return &sqlxTxManager{db: db}
}

func (m *sqlxTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txCtx := context.WithValue(ctx, txKey, tx)
	if err := fn(txCtx); err != nil {
		return err
	}

	return tx.Commit()
}

func TxFromContext(ctx context.Context) *sqlx.Tx {
	tx, _ := ctx.Value(txKey).(*sqlx.Tx)
	return tx
}
