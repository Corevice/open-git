package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IWorkflowRunRepository interface {
	ListByHeadSHA(ctx context.Context, repoID uuid.UUID, headSHA string) ([]*entity.WorkflowRun, error)
}
