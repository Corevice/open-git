package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type txContextKey struct{}

var txKey txContextKey

type sqlxTransactionManager struct {
	db *sqlx.DB
}

var _ domainrepo.ITransactionManager = (*sqlxTransactionManager)(nil)

func NewTransactionManager(db *sqlx.DB) domainrepo.ITransactionManager {
	return &sqlxTransactionManager{db: db}
}

func (m *sqlxTransactionManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
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
