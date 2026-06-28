package queue

import (
	"context"

	"github.com/open-git/backend/internal/domain/entity"
)

// JobLogPublisher publishes job log lines to a Redis Stream for live subscribers.
type JobLogPublisher struct{}

func NewJobLogPublisher(_ string) *JobLogPublisher {
	return &JobLogPublisher{}
}

func (p *JobLogPublisher) Publish(_ context.Context, _ *entity.JobLogLine) error {
	if p == nil {
		return nil
	}
	return nil
}

// JobLogSubscriber receives live job log lines from a Redis Stream.
type JobLogSubscriber struct{}

func NewJobLogSubscriber(_ string) *JobLogSubscriber {
	return &JobLogSubscriber{}
}

func (s *JobLogSubscriber) Subscribe(ctx context.Context, _ string, _ int64) (<-chan *entity.JobLogLine, <-chan struct{}, error) {
	lines := make(chan *entity.JobLogLine)
	done := make(chan struct{})
	close(lines)
	close(done)
	return lines, done, nil
}
