package repository

import "context"

type TransactionManager interface {
	RunInTransaction(ctx context.Context, fn func(txCtx context.Context) error) error
}
