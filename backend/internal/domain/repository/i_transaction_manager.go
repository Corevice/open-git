package repository

import "context"

type Tx interface{}

type ITransactionManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
