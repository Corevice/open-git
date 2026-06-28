package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IWorkflowStepRepository interface {
	CreateBatch(ctx context.Context, steps []*entity.WorkflowStep) error
	ResetQueuedByRunID(ctx context.Context, runID uuid.UUID) error
}
