package workflow

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
)

type RerunRunInput struct {
	OrganizationID uuid.UUID
	RunID          uuid.UUID
}

type RerunRunUsecase struct {
	runRepo WorkflowRunRepository
}

func NewRerunRunUsecase(runRepo WorkflowRunRepository) *RerunRunUsecase {
	return &RerunRunUsecase{runRepo: runRepo}
}

func (uc *RerunRunUsecase) Execute(ctx context.Context, input RerunRunInput) (*entity.WorkflowRun, error) {
	run, err := uc.runRepo.GetByID(ctx, input.OrganizationID, input.RunID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, domain.ErrNotFound
	}

	if run.Status != "completed" {
		return nil, domain.ErrConflict
	}

	newRun := &entity.WorkflowRun{
		OrganizationID: run.OrganizationID,
		RepositoryID:   run.RepositoryID,
		WorkflowID:     run.WorkflowID,
		Workflow:       run.Workflow,
		Status:         "queued",
		RunNumber:      run.RunNumber + 1,
		HeadSHA:        run.HeadSHA,
		HeadBranch:     run.HeadBranch,
		Event:          "workflow_dispatch",
	}

	if err := uc.runRepo.Create(ctx, newRun); err != nil {
		return nil, err
	}

	return newRun, nil
}
