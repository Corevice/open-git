package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IWorkflowRepository interface {
	ListActiveByRepository(ctx context.Context, orgID, repoID uuid.UUID) ([]*entity.Workflow, error)
}
