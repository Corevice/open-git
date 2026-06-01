package queue

import "github.com/hibiken/asynq"

const (
	TypeWebhookDeliver = "webhook:deliver"
)

func NewAsynqClient(addr string) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{Addr: addr})
}

func NewAsynqServer(addr string, concurrency int) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: addr},
		asynq.Config{Concurrency: concurrency},
	)
}
