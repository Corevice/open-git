package workflow

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
)

type CancelRunInput struct {
	OrganizationID uuid.UUID
	RunID          uuid.UUID
}

type CancelRunUsecase struct {
	runRepo WorkflowRunRepository
}

func NewCancelRunUsecase(runRepo WorkflowRunRepository) *CancelRunUsecase {
	return &CancelRunUsecase{runRepo: runRepo}
}

func (uc *CancelRunUsecase) Execute(ctx context.Context, input CancelRunInput) error {
	run, err := uc.runRepo.GetByID(ctx, input.OrganizationID, input.RunID)
	if err != nil {
		return err
	}
	if run == nil {
		return domain.ErrNotFound
	}

	if run.Status != "queued" && run.Status != "in_progress" {
		return domain.ErrConflict
	}

	now := time.Now().UTC()
	return uc.runRepo.UpdateStatus(ctx, input.RunID, "completed", "cancelled", &now)
}
