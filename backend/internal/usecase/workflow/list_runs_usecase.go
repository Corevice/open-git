package workflow

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ListWorkflowRunsInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Status         string
	Conclusion     string
	Branch         string
	Event          string
	Page           int
	PerPage        int
}

type ListWorkflowRunsOutput struct {
	Runs    []*entity.WorkflowRun
	Total   int
	Page    int
	PerPage int
}

type ListWorkflowRunsUsecase struct {
	runRepo WorkflowRunRepository
}

func NewListWorkflowRunsUsecase(runRepo WorkflowRunRepository) *ListWorkflowRunsUsecase {
	return &ListWorkflowRunsUsecase{runRepo: runRepo}
}

func (uc *ListWorkflowRunsUsecase) Execute(ctx context.Context, input ListWorkflowRunsInput) (*ListWorkflowRunsOutput, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}
	perPage := input.PerPage
	if perPage < 1 {
		perPage = 1
	}
	if perPage > 100 {
		perPage = 100
	}

	filter := ListWorkflowRunsFilter{
		OrganizationID: input.OrganizationID,
		RepositoryID:   input.RepositoryID,
		Status:         input.Status,
		Conclusion:     input.Conclusion,
		Branch:         input.Branch,
		Event:          input.Event,
		Page:           page,
		PerPage:        perPage,
	}

	runs, total, err := uc.runRepo.ListByRepo(ctx, filter)
	if err != nil {
		return nil, err
	}
	if runs == nil {
		runs = []*entity.WorkflowRun{}
	}

	filtered := make([]*entity.WorkflowRun, 0, len(runs))
	for _, run := range runs {
		if run.OrganizationID == input.OrganizationID {
			filtered = append(filtered, run)
		}
	}
	runs = filtered

	return &ListWorkflowRunsOutput{
		Runs:    runs,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}
