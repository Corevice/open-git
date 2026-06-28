package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/redis/go-redis/v9"
)

type JobLogEvent struct {
	Step   int    `json:"step"`
	Line   int    `json:"line"`
	TS     string `json:"ts"`
	Stream string `json:"stream"`
	Text   string `json:"text"`
}

type JobLogPublisher struct {
	client *redis.Client
}

func NewJobLogPublisher(client *redis.Client) *JobLogPublisher {
	return &JobLogPublisher{client: client}
}

func (p *JobLogPublisher) streamKey(jobID string) string {
	return "joblog:" + jobID
}

func (p *JobLogPublisher) Publish(ctx context.Context, jobID string, line *entity.JobLogLine) error {
	event := JobLogEvent{
		Step:   line.StepIndex,
		Line:   int(line.LineNumber),
		TS:     line.CreatedAt.UTC().Format(time.RFC3339Nano),
		Stream: line.Stream,
		Text:   line.Text,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal job log event: %w", err)
	}
	return p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: p.streamKey(jobID),
		MaxLen: 50000,
		Approx: true,
		Values: map[string]interface{}{
			"data": string(payload),
		},
	}).Err()
}

type JobLogSubscriber struct {
	client *redis.Client
}

func NewJobLogSubscriber(client *redis.Client) *JobLogSubscriber {
	return &JobLogSubscriber{client: client}
}

func (s *JobLogSubscriber) streamKey(jobID string) string {
	return "joblog:" + jobID
}

func (s *JobLogSubscriber) Subscribe(ctx context.Context, jobID, lastID string, sink func(*entity.JobLogLine) error) error {
	if lastID == "" {
		lastID = "0-0"
	}
	stream := s.streamKey(jobID)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		streams, err := s.client.XRead(ctx, &redis.XReadArgs{
			Streams: []string{stream, lastID},
			Block:   5 * time.Second,
		}).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}

		for _, streamResult := range streams {
			for _, message := range streamResult.Messages {
				lastID = message.ID

				raw, ok := message.Values["data"].(string)
				if !ok {
					continue
				}

				var event JobLogEvent
				if err := json.Unmarshal([]byte(raw), &event); err != nil {
					continue
				}

				ts, err := time.Parse(time.RFC3339Nano, event.TS)
				if err != nil {
					ts = time.Now().UTC()
				}

				line := &entity.JobLogLine{
					JobID:      jobID,
					StepIndex:  event.Step,
					LineNumber: int64(event.Line),
					Stream:     event.Stream,
					Text:       event.Text,
					CreatedAt:  ts,
				}
				if err := sink(line); err != nil {
					return context.Canceled
				}
			}
		}
	}
}
