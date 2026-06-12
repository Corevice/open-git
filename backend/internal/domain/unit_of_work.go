package domain

import "context"

type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type NoopUnitOfWork struct{}

func (NoopUnitOfWork) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
