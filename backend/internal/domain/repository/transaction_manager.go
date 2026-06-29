package repository

import "context"

// TransactionManager runs the given function within a database transaction,
// passing a context that carries the active transaction.
type TransactionManager interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
