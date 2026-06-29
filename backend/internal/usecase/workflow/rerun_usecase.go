package workflow

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
)

type RerunInput struct {
	RunID   uuid.UUID
	OrgID   uuid.UUID
	ActorID uuid.UUID
}

type RerunUsecase struct {
	runRepo        domainrepo.IWorkflowRunRepository
	jobRepo        domainrepo.IWorkflowJobRepository
	stepRepo       domainrepo.IWorkflowStepRepository
	membershipRepo domainrepo.IMembershipRepository
	auditLogRepo   domainrepo.IAuditLogRepository
	enqueuer       WorkflowScheduleEnqueuer
}

func NewRerunUsecase(
	runRepo domainrepo.IWorkflowRunRepository,
	jobRepo domainrepo.IWorkflowJobRepository,
	stepRepo domainrepo.IWorkflowStepRepository,
	membershipRepo domainrepo.IMembershipRepository,
	auditLogRepo domainrepo.IAuditLogRepository,
	client *asynq.Client,
) *RerunUsecase {
	return NewRerunUsecaseWithEnqueuer(
		runRepo,
		jobRepo,
		stepRepo,
		membershipRepo,
		auditLogRepo,
		newAsynqWorkflowScheduleEnqueuer(client),
	)
}

func NewRerunUsecaseWithEnqueuer(
	runRepo domainrepo.IWorkflowRunRepository,
	jobRepo domainrepo.IWorkflowJobRepository,
	stepRepo domainrepo.IWorkflowStepRepository,
	membershipRepo domainrepo.IMembershipRepository,
	auditLogRepo domainrepo.IAuditLogRepository,
	enqueuer WorkflowScheduleEnqueuer,
) *RerunUsecase {
	return &RerunUsecase{
		runRepo:        runRepo,
		jobRepo:        jobRepo,
		stepRepo:       stepRepo,
		membershipRepo: membershipRepo,
		auditLogRepo:   auditLogRepo,
		enqueuer:       enqueuer,
	}
}

func (uc *RerunUsecase) Execute(ctx context.Context, input RerunInput) error {
	if err := uc.checkActorWriteAccess(ctx, input.OrgID, input.ActorID); err != nil {
		return err
	}

	run, err := uc.runRepo.GetByID(ctx, input.RunID, input.OrgID)
	if err != nil {
		return err
	}
	if run.OrganizationID != input.OrgID {
		return domain.ErrForbidden
	}

	newAttempt, err := uc.runRepo.IncrementRunAttempt(ctx, input.RunID, input.OrgID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	run.RunAttempt = newAttempt
	run.Status = runStatusQueued
	run.Conclusion = ""
	run.UpdatedAt = now
	run.StartedAt = nil
	run.CompletedAt = nil

	if err := uc.runRepo.Update(ctx, run); err != nil {
		return err
	}

	if err := uc.jobRepo.ResetQueuedByRunID(ctx, input.RunID); err != nil {
		return err
	}
	if err := uc.stepRepo.ResetQueuedByRunID(ctx, input.RunID); err != nil {
		return err
	}

	if err := uc.auditLogRepo.Create(ctx, &entity.AuditLog{
		ID:             uuid.New(),
		OrganizationID: input.OrgID,
		ActorID:        input.ActorID,
		Action:         "run.rerun",
		TargetType:     "workflow_run",
		TargetID:       input.RunID.String(),
		CreatedAt:      now,
	}); err != nil {
		return err
	}

	return uc.enqueuer.EnqueueSchedule(ctx, queue.WorkflowSchedulePayload{
		RunID: input.RunID.String(),
		OrgID: input.OrgID.String(),
	})
}

func (uc *RerunUsecase) checkActorWriteAccess(ctx context.Context, organizationID, actorID uuid.UUID) error {
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
