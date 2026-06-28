package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
)

const freePlanMonthlyRunLimit = 10

type RunVerificationInput struct {
	RepositoryFullName string
	Targets            []string
}

type RunVerificationUsecase struct {
	repo         domainrepo.IMCPVerificationRepository
	auditLogRepo domainrepo.IAuditLogRepository
	asynqClient  *asynq.Client
}

func NewRunVerificationUsecase(
	repo domainrepo.IMCPVerificationRepository,
	auditLogRepo domainrepo.IAuditLogRepository,
	asynqClient *asynq.Client,
) *RunVerificationUsecase {
	return &RunVerificationUsecase{
		repo:         repo,
		auditLogRepo: auditLogRepo,
		asynqClient:  asynqClient,
	}
}

func (uc *RunVerificationUsecase) Execute(
	ctx context.Context,
	orgID, actorID uuid.UUID,
	input RunVerificationInput,
) (*entity.MCPVerificationRun, error) {
	activeRun, err := uc.repo.GetActiveRun(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if activeRun != nil {
		return nil, ErrMCPRunConflict
	}

	count, err := uc.repo.CountRunsThisMonth(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if count >= freePlanMonthlyRunLimit {
		return nil, ErrMCPPlanLimitExceeded
	}

	if input.RepositoryFullName == "" {
		return nil, fmt.Errorf("repository full name is required: %w", errors.New("repository is required"))
	}

	targets := input.Targets
	if len(targets) == 0 {
		targets = []string{"graphql", "rest", "auth"}
	}

	targetsJSON, err := json.Marshal(targets)
	if err != nil {
		return nil, fmt.Errorf("marshal targets: %w", err)
	}

	now := time.Now().UTC()
	triggeredBy := actorID
	run := &entity.MCPVerificationRun{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		RepositoryFullName: input.RepositoryFullName,
		TriggeredBy:        &triggeredBy,
		Status:             entity.RunStatusQueued,
		Targets:            targetsJSON,
		CreatedAt:          now,
	}

	if err := run.Validate(); err != nil {
		return nil, fmt.Errorf("validate run: %w", err)
	}

	if err := uc.repo.CreateRun(ctx, run); err != nil {
		return nil, err
	}

	if _, err := queue.EnqueueMCPVerification(ctx, uc.asynqClient, queue.MCPVerificationPayload{
		RunID:              run.ID.String(),
		OrganizationID:     orgID.String(),
		RepositoryFullName: input.RepositoryFullName,
		Targets:            targets,
	}); err != nil {
		return nil, fmt.Errorf("enqueue mcp verification: %w", err)
	}

	if err := uc.auditLogRepo.Create(ctx, &entity.AuditLog{
		ID:             uuid.New(),
		OrganizationID: orgID,
		ActorID:        actorID,
		Action:         "mcp_verification.run",
		TargetType:     "mcp_verification_run",
		TargetID:       run.ID.String(),
		CreatedAt:      now,
	}); err != nil {
		return nil, err
	}

	return run, nil
}
