package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const (
	TypeWorkflowSchedule = "workflow:schedule"
	TypeWorkflowJobExec  = "workflow:job_exec"
)

type WorkflowSchedulePayload struct {
	RunID string `json:"run_id"`
	OrgID string `json:"org_id"`
}

type WorkflowJobExecPayload struct {
	JobID string `json:"job_id"`
	RunID string `json:"run_id"`
	OrgID string `json:"org_id"`
}

func EnqueueWorkflowSchedule(ctx context.Context, client *asynq.Client, payload WorkflowSchedulePayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal workflow schedule payload: %w", err)
	}
	task := asynq.NewTask(TypeWorkflowSchedule, data)
	return client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
}

func EnqueueWorkflowJobExec(ctx context.Context, client *asynq.Client, payload WorkflowJobExecPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal workflow job exec payload: %w", err)
	}
	task := asynq.NewTask(TypeWorkflowJobExec, data)
	return client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
}
