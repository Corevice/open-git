package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IWorkflowJobRepository interface {
	Create(ctx context.Context, job *entity.WorkflowJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.WorkflowJob, error)
	AcquireForRunner(ctx context.Context, jobID uuid.UUID, runnerID uuid.UUID, lockVersion int) (bool, error)
	UpdateStatus(ctx context.Context, jobID uuid.UUID, status, conclusion string) error
	Complete(ctx context.Context, jobID uuid.UUID, conclusion string, finishedAt time.Time) error
	Cancel(ctx context.Context, jobID uuid.UUID) error
	ListQueued(ctx context.Context, orgID uuid.UUID) ([]*entity.WorkflowJob, error)
	ListByRunID(ctx context.Context, orgID uuid.UUID, runID uuid.UUID) ([]*entity.WorkflowJob, error)
}
