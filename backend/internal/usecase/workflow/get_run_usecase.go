package workflow

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
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

type CancelRunInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	RunID          uuid.UUID
	ActorID        uuid.UUID
}

type cancelRunRepository interface {
	GetByID(ctx context.Context, orgID, repoID, runID uuid.UUID) (*entity.WorkflowRun, error)
	Cancel(ctx context.Context, orgID, repoID, runID, actorID uuid.UUID) error
}

type CancelRunUsecase struct {
	runRepo cancelRunRepository
}

func NewCancelRunUsecase(runRepo cancelRunRepository) *CancelRunUsecase {
	return &CancelRunUsecase{runRepo: runRepo}
}

func (uc *CancelRunUsecase) Execute(ctx context.Context, input CancelRunInput) error {
	run, err := uc.runRepo.GetByID(ctx, input.OrganizationID, input.RepositoryID, input.RunID)
	if err != nil {
		return err
	}
	if run == nil {
		return apperror.ErrNotFound
	}
	if isTerminalRun(run) {
		return domain.ErrConflict
	}
	return uc.runRepo.Cancel(ctx, input.OrganizationID, input.RepositoryID, input.RunID, input.ActorID)
}

type RerunRunInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	RunID          uuid.UUID
	ActorID        uuid.UUID
}

type rerunRunRepository interface {
	Rerun(ctx context.Context, orgID, repoID, runID, actorID uuid.UUID) (*entity.WorkflowRun, error)
}

type RerunRunUsecase struct {
	runRepo rerunRunRepository
}

func NewRerunRunUsecase(runRepo rerunRunRepository) *RerunRunUsecase {
	return &RerunRunUsecase{runRepo: runRepo}
}

func (uc *RerunRunUsecase) Execute(ctx context.Context, input RerunRunInput) (*entity.WorkflowRun, error) {
	return uc.runRepo.Rerun(ctx, input.OrganizationID, input.RepositoryID, input.RunID, input.ActorID)
}

func isTerminalRun(run *entity.WorkflowRun) bool {
	if run.Status == entity.WorkflowStatusCompleted {
		return true
	}
	switch run.Conclusion {
	case entity.WorkflowConclusionSuccess, "failure", "cancelled", "skipped":
		return true
	default:
		return false
	}
}
