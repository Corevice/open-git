package repository

import (
	"context"

	"github.com/google/uuid"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	workflowusecase "github.com/open-git/backend/internal/usecase/workflow"
)

// WorkflowJobListingAdapter adapts the domain workflow-job repository (which
// returns entity.WorkflowJob keyed by org+run) to the listing shape the
// workflow usecases expect (workflowusecase.WorkflowJob keyed by
// org+repo+run). The extra repo id is accepted for interface parity and used
// as a defensive filter.
type WorkflowJobListingAdapter struct {
	repo domainrepo.IWorkflowJobRepository
}

func NewWorkflowJobListingAdapter(repo domainrepo.IWorkflowJobRepository) *WorkflowJobListingAdapter {
	return &WorkflowJobListingAdapter{repo: repo}
}

func (a *WorkflowJobListingAdapter) ListByRunID(ctx context.Context, orgID, repoID, runID uuid.UUID) ([]*workflowusecase.WorkflowJob, error) {
	jobs, err := a.repo.ListByRunID(ctx, orgID, runID)
	if err != nil {
		return nil, err
	}

	out := make([]*workflowusecase.WorkflowJob, 0, len(jobs))
	for _, job := range jobs {
		if job.RepositoryID != uuid.Nil && repoID != uuid.Nil && job.RepositoryID != repoID {
			continue
		}
		runIDValue := runID
		if job.WorkflowRunID != nil {
			runIDValue = *job.WorkflowRunID
		}
		out = append(out, &workflowusecase.WorkflowJob{
			ID:          job.ID,
			RunID:       runIDValue,
			Name:        job.Name,
			Status:      job.Status,
			Conclusion:  job.Conclusion,
			StartedAt:   job.StartedAt,
			CompletedAt: job.FinishedAt,
		})
	}
	return out, nil
}
