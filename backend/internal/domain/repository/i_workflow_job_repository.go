package repository

import (
	"context"
	"time"

	"github.com/open-git/backend/internal/domain/entity"
)

type IWorkflowJobRepository interface {
	Create(ctx context.Context, job *entity.WorkflowJob) error
	GetByID(ctx context.Context, orgID, jobID string) (*entity.WorkflowJob, error)
	UpdateStatus(ctx context.Context, jobID, status, conclusion string, completedAt *time.Time) error
	ListByRunID(ctx context.Context, orgID, runID string) ([]*entity.WorkflowJob, error)
}
