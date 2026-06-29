package actions

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type DeleteRunnerUsecase struct {
	runnerRepo   domainrepo.IRunnerRepository
	auditLogRepo domainrepo.IAuditLogRepository
}

func NewDeleteRunnerUsecase(
	runnerRepo domainrepo.IRunnerRepository,
	auditLogRepo domainrepo.IAuditLogRepository,
) *DeleteRunnerUsecase {
	return &DeleteRunnerUsecase{
		runnerRepo:   runnerRepo,
		auditLogRepo: auditLogRepo,
	}
}

func (uc *DeleteRunnerUsecase) Execute(
	ctx context.Context,
	orgID uuid.UUID,
	runnerID uuid.UUID,
	actorRole string,
) error {
	if actorRole != entity.RoleAdmin {
		return domain.ErrForbidden
	}

	runner, err := uc.runnerRepo.GetByID(ctx, runnerID)
	if err != nil {
		return err
	}
	if runner.OrganizationID != orgID {
		return domain.ErrNotFound
	}

	if err := uc.runnerRepo.Delete(ctx, runnerID); err != nil {
		return err
	}

	now := time.Now().UTC()
	return uc.auditLogRepo.Create(ctx, &entity.AuditLog{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Action:         "runner_deleted",
		TargetType:     "runner",
		TargetID:       runnerID.String(),
		CreatedAt:      now,
	})
}
