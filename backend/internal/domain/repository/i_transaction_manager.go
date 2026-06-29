package repository

import "context"

type ITransactionManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
