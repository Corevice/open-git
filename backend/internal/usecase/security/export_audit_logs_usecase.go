package security

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const TypeAuditLogExport = "audit_log:export"

type ExportAuditLogsInput struct {
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	Format         string
	Phrase         string
	Action         string
	After          *time.Time
	Before         *time.Time
}

type ExportAuditLogsOutput struct {
	JobID uuid.UUID
}

type AuditLogExportPayload struct {
	JobID          string     `json:"job_id"`
	OrganizationID string     `json:"organization_id"`
	ActorID        string     `json:"actor_id"`
	Format         string     `json:"format"`
	Phrase         string     `json:"phrase"`
	Action         string     `json:"action"`
	After          *time.Time `json:"after,omitempty"`
	Before         *time.Time `json:"before,omitempty"`
}

type AuditLogExportEnqueuer interface {
	EnqueueAuditLogExport(ctx context.Context, payload AuditLogExportPayload) error
}

type asynqAuditLogExportEnqueuer struct {
	client *asynq.Client
}

func newAsynqAuditLogExportEnqueuer(client *asynq.Client) AuditLogExportEnqueuer {
	return &asynqAuditLogExportEnqueuer{client: client}
}

func (e *asynqAuditLogExportEnqueuer) EnqueueAuditLogExport(ctx context.Context, payload AuditLogExportPayload) error {
	if e.client == nil {
		return fmt.Errorf("redis not configured")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal audit log export payload: %w", err)
	}
	task := asynq.NewTask(TypeAuditLogExport, data, asynq.MaxRetry(3))
	_, err = e.client.EnqueueContext(ctx, task)
	return err
}

type noopAuditLogExportEnqueuer struct{}

func (noopAuditLogExportEnqueuer) EnqueueAuditLogExport(context.Context, AuditLogExportPayload) error {
	return fmt.Errorf("redis not configured")
}

type ExportAuditLogsUsecase struct {
	enqueuer AuditLogExportEnqueuer
}

func NewExportAuditLogsUsecase(client *asynq.Client) *ExportAuditLogsUsecase {
	if client == nil {
		return NewExportAuditLogsUsecaseWithDeps(noopAuditLogExportEnqueuer{})
	}
	return NewExportAuditLogsUsecaseWithDeps(newAsynqAuditLogExportEnqueuer(client))
}

func NewExportAuditLogsUsecaseWithDeps(enqueuer AuditLogExportEnqueuer) *ExportAuditLogsUsecase {
	return &ExportAuditLogsUsecase{enqueuer: enqueuer}
}

func (uc *ExportAuditLogsUsecase) Execute(ctx context.Context, input ExportAuditLogsInput) (*ExportAuditLogsOutput, error) {
	if err := validateAuditLogDateRange(input.After, input.Before); err != nil {
		return nil, err
	}

	jobID := uuid.New()
	if err := uc.enqueuer.EnqueueAuditLogExport(ctx, AuditLogExportPayload{
		JobID:          jobID.String(),
		OrganizationID: input.OrganizationID.String(),
		ActorID:        input.ActorID.String(),
		Format:         input.Format,
		Phrase:         input.Phrase,
		Action:         input.Action,
		After:          input.After,
		Before:         input.Before,
	}); err != nil {
		return nil, fmt.Errorf("enqueue audit log export: %w", err)
	}

	return &ExportAuditLogsOutput{JobID: jobID}, nil
}
