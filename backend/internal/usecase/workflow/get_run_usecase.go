package workflow

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type GetRunInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	RunID          uuid.UUID
}

type getRunRepository interface {
	GetByID(ctx context.Context, orgID, repoID, runID uuid.UUID) (*entity.WorkflowRun, error)
}

type GetRunUsecase struct {
	runRepo getRunRepository
}

func NewGetRunUsecase(runRepo getRunRepository) *GetRunUsecase {
	return &GetRunUsecase{runRepo: runRepo}
}

func (uc *GetRunUsecase) Execute(ctx context.Context, input GetRunInput) (*entity.WorkflowRun, error) {
	return uc.runRepo.GetByID(ctx, input.OrganizationID, input.RepositoryID, input.RunID)
}
