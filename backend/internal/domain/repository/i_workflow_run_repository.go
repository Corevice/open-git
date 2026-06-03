package repository

import (
	"context"

	"github.com/Corevice/open-git/backend/internal/domain/entity"
	"github.com/google/uuid"
)

type IWorkflowRunRepository interface {
	ListByHeadSHA(ctx context.Context, repositoryID uuid.UUID, headSHA string) ([]*entity.WorkflowRun, error)
}
