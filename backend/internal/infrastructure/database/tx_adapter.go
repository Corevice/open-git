package database

import (
	"context"

	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/jmoiron/sqlx"
)

type domainTxManagerAdapter struct {
	db *sqlx.DB
}

func NewDomainTxManager(db *sqlx.DB) domainrepo.TransactionManager {
	return &domainTxManagerAdapter{db: db}
}

func (a *domainTxManagerAdapter) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := a.db.BeginTxx(ctx, nil)
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
