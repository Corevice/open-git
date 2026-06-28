package actions

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type HeartbeatRunnerUsecase struct {
	runnerRepo domainrepo.IRunnerRepository
}

func NewHeartbeatRunnerUsecase(runnerRepo domainrepo.IRunnerRepository) *HeartbeatRunnerUsecase {
	return &HeartbeatRunnerUsecase{runnerRepo: runnerRepo}
}

func (uc *HeartbeatRunnerUsecase) Execute(
	ctx context.Context,
	orgID uuid.UUID,
	runnerID uuid.UUID,
	status string,
	runningJobID *string,
) error {
	_ = runningJobID

	runner, err := uc.runnerRepo.GetByID(ctx, runnerID)
	if err != nil {
		return err
	}
	if runner.OrganizationID != orgID {
		return domain.ErrNotFound
	}

	now := time.Now().UTC()
	return uc.runnerRepo.UpdateStatus(ctx, runnerID, status, now)
}
