package mcp

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type DeleteVerificationUsecase struct {
	repo         domainrepo.IMCPVerificationRepository
	auditLogRepo domainrepo.IAuditLogRepository
}

func NewDeleteVerificationUsecase(
	repo domainrepo.IMCPVerificationRepository,
	auditLogRepo domainrepo.IAuditLogRepository,
) *DeleteVerificationUsecase {
	return &DeleteVerificationUsecase{
		repo:         repo,
		auditLogRepo: auditLogRepo,
	}
}

func (uc *DeleteVerificationUsecase) Execute(
	ctx context.Context,
	orgID, actorID, runID uuid.UUID,
) error {
	if err := uc.repo.DeleteRun(ctx, runID, orgID); err != nil {
		return err
	}

	return uc.auditLogRepo.Create(ctx, &entity.AuditLog{
		ID:             uuid.New(),
		OrganizationID: orgID,
		ActorID:        actorID,
		Action:         "mcp_verification.delete",
		TargetType:     "mcp_verification_run",
		TargetID:       runID.String(),
		CreatedAt:      time.Now().UTC(),
	})
}
