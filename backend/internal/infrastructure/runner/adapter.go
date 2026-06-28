package runner

import "context"

type ActJobPayload struct {
	JobID          string
	WorkflowYAML   []byte
	Env            []string
	TimeoutMinutes int
}

type RunnerAdapter interface {
	Execute(ctx context.Context, job ActJobPayload) error
	Cancel(ctx context.Context, jobID string) error
}
