package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const TypeAuditLogExport = "audit_log:export"

var ErrRedisNotConfigured = errors.New("redis not configured")

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
		return ErrRedisNotConfigured
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal audit log export payload: %w", err)
	}
	task := asynq.NewTask(TypeAuditLogExport, data, asynq.MaxRetry(3))
	_, err = e.client.EnqueueContext(ctx, task)
	return err
}

type ExportAuditLogsUsecase struct {
	enqueuer AuditLogExportEnqueuer
}

func NewExportAuditLogsUsecase(client *asynq.Client) *ExportAuditLogsUsecase {
	if client == nil {
		return &ExportAuditLogsUsecase{}
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
	if uc.enqueuer == nil {
		return nil, ErrRedisNotConfigured
	}

	after, before := normalizeAuditLogDateRange(input.After, input.Before)

	jobID := uuid.New()
	if err := uc.enqueuer.EnqueueAuditLogExport(ctx, AuditLogExportPayload{
		JobID:          jobID.String(),
		OrganizationID: input.OrganizationID.String(),
		ActorID:        input.ActorID.String(),
		Format:         input.Format,
		Phrase:         input.Phrase,
		Action:         input.Action,
		After:          after,
		Before:         before,
	}); err != nil {
		if errors.Is(err, ErrRedisNotConfigured) {
			return nil, err
		}
		return nil, fmt.Errorf("enqueue audit log export: %w", err)
	}

	return &ExportAuditLogsOutput{JobID: jobID}, nil
}
