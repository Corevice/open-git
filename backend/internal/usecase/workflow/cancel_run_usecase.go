package workflow

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type CancelRunInput struct {
	RunID   uuid.UUID
	OrgID   uuid.UUID
	ActorID uuid.UUID
}

type CancelRunUsecase struct {
	runRepo        domainrepo.IWorkflowRunRepository
	jobRepo        domainrepo.IWorkflowJobRepository
	membershipRepo domainrepo.IMembershipRepository
	auditLogRepo   domainrepo.IAuditLogRepository
}

func NewCancelRunUsecase(
	runRepo domainrepo.IWorkflowRunRepository,
	jobRepo domainrepo.IWorkflowJobRepository,
	membershipRepo domainrepo.IMembershipRepository,
	auditLogRepo domainrepo.IAuditLogRepository,
) *CancelRunUsecase {
	return &CancelRunUsecase{
		runRepo:        runRepo,
		jobRepo:        jobRepo,
		membershipRepo: membershipRepo,
		auditLogRepo:   auditLogRepo,
	}
}

func (uc *CancelRunUsecase) Execute(ctx context.Context, input CancelRunInput) error {
	if err := uc.checkActorWriteAccess(ctx, input.OrgID, input.ActorID); err != nil {
		return err
	}

	run, err := uc.runRepo.GetByID(ctx, input.RunID, input.OrgID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	run.Status = runStatusCompleted
	run.Conclusion = "cancelled"
	run.UpdatedAt = now
	run.CompletedAt = &now

	if err := uc.runRepo.Update(ctx, run); err != nil {
		return err
	}

	if err := uc.jobRepo.CancelInProgressByRunID(ctx, input.OrgID, input.RunID); err != nil {
		return err
	}

	return uc.auditLogRepo.Create(ctx, &entity.AuditLog{
		ID:             uuid.New(),
		OrganizationID: input.OrgID,
		ActorID:        input.ActorID,
		Action:         "run.cancel",
		TargetType:     "workflow_run",
		TargetID:       input.RunID.String(),
		CreatedAt:      now,
	})
}

func (uc *CancelRunUsecase) checkActorWriteAccess(ctx context.Context, organizationID, actorID uuid.UUID) error {
	role, err := uc.membershipRepo.GetRole(ctx, organizationID, actorID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.ErrForbidden
	}
	if err != nil {
		return err
	}
	if role != entity.RoleOwner && role != entity.RoleAdmin {
		return domain.ErrForbidden
	}
	return nil
}
