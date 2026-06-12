package domain

import "context"

type JobEnqueuer interface {
	Enqueue(ctx context.Context, task string, payload any) error
}
