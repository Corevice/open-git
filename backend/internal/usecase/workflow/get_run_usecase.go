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

type RerunRunInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	RunID          uuid.UUID
	ActorID        uuid.UUID
}

type rerunRunRepository interface {
	GetByID(ctx context.Context, orgID, repoID, runID uuid.UUID) (*entity.WorkflowRun, error)
	Rerun(ctx context.Context, orgID, repoID, runID, actorID uuid.UUID) (*entity.WorkflowRun, error)
}

type RerunRunUsecase struct {
	runRepo rerunRunRepository
}

func NewRerunRunUsecase(runRepo rerunRunRepository) *RerunRunUsecase {
	return &RerunRunUsecase{runRepo: runRepo}
}

func (uc *RerunRunUsecase) Execute(ctx context.Context, input RerunRunInput) (*entity.WorkflowRun, error) {
	run, err := uc.runRepo.GetByID(ctx, input.OrganizationID, input.RepositoryID, input.RunID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, apperror.ErrNotFound
	}
	return uc.runRepo.Rerun(ctx, input.OrganizationID, input.RepositoryID, input.RunID, input.ActorID)
}

func isTerminalRun(run *entity.WorkflowRun) bool {
	return run.Status == entity.WorkflowStatusCompleted
}
