package workflow

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
)

type GetWorkflowRunInput struct {
	OrganizationID uuid.UUID
	RunID          uuid.UUID
}

type GetWorkflowRunOutput struct {
	Run  *entity.WorkflowRun
	Jobs []*entity.WorkflowJob
}

type GetWorkflowRunUsecase struct {
	runRepo WorkflowRunRepository
	jobRepo WorkflowJobRepository
}

func NewGetWorkflowRunUsecase(
	runRepo WorkflowRunRepository,
	jobRepo WorkflowJobRepository,
) *GetWorkflowRunUsecase {
	return &GetWorkflowRunUsecase{
		runRepo: runRepo,
		jobRepo: jobRepo,
	}
}

func (uc *GetWorkflowRunUsecase) Execute(ctx context.Context, input GetWorkflowRunInput) (*GetWorkflowRunOutput, error) {
	run, err := uc.runRepo.GetByID(ctx, input.OrganizationID, input.RunID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, domain.ErrNotFound
	}

	jobs, err := uc.jobRepo.ListByRunID(ctx, input.OrganizationID, input.RunID)
	if err != nil {
		return nil, err
	}

	return &GetWorkflowRunOutput{
		Run:  run,
		Jobs: jobs,
	}, nil
}
