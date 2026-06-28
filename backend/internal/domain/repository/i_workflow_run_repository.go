package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IWorkflowRunRepository interface {
	ListByHeadSHA(ctx context.Context, repoID uuid.UUID, sha string) ([]*entity.WorkflowRun, error)
	Create(ctx context.Context, run *entity.WorkflowRun) error
	GetByID(ctx context.Context, runID, orgID uuid.UUID) (*entity.WorkflowRun, error)
	Update(ctx context.Context, run *entity.WorkflowRun) error
	IncrementRunNumber(ctx context.Context, orgID, repoID uuid.UUID) (int, error)
	IncrementRunAttempt(ctx context.Context, runID, orgID uuid.UUID) (int, error)
}
