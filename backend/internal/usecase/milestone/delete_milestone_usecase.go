package milestone

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/repository"
)

type DeleteMilestoneInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	ActorID        uuid.UUID
	Number         int
}

type DeleteMilestoneUsecase struct {
	milestoneRepo repository.IMilestoneRepository
	auditLogRepo  repository.IAuditLogRepository
}

func NewDeleteMilestoneUsecase(
	milestoneRepo repository.IMilestoneRepository,
	auditLogRepo repository.IAuditLogRepository,
) *DeleteMilestoneUsecase {
	return &DeleteMilestoneUsecase{
		milestoneRepo: milestoneRepo,
		auditLogRepo:  auditLogRepo,
	}
}

func (uc *DeleteMilestoneUsecase) Execute(ctx context.Context, input DeleteMilestoneInput) error {
	milestone, err := uc.milestoneRepo.GetByNumber(ctx, input.RepositoryID, input.Number)
	if err != nil {
		return err
	}
	if milestone == nil {
		return apperror.ErrNotFound
	}

	if err := uc.milestoneRepo.Delete(ctx, milestone.ID); err != nil {
		return err
	}

	return uc.auditLogRepo.InsertAuditLog(
		ctx,
		input.OrganizationID,
		input.ActorID,
		"milestone.delete",
		"milestone",
		milestone.ID,
		json.RawMessage(`{}`),
	)
}
