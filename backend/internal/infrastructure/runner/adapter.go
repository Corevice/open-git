package runner

import "context"

type RunnerJobPayload struct {
	JobID          string
	WorkflowYAML   []byte
	Env            []string
	TimeoutMinutes int
}

// ActJobPayload is a backward-compatible alias for RunnerJobPayload.
type ActJobPayload = RunnerJobPayload

type RunnerAdapter interface {
	Execute(ctx context.Context, job RunnerJobPayload) error
	Cancel(ctx context.Context, jobID string) error
}
