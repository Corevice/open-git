package label

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/repository"
)

type DeleteLabelInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	ActorID        uuid.UUID
	Name           string
}

type DeleteLabelUsecase struct {
	labelRepo    repository.ILabelRepository
	auditLogRepo repository.IAuditLogRepository
}

func NewDeleteLabelUsecase(
	labelRepo repository.ILabelRepository,
	auditLogRepo repository.IAuditLogRepository,
) *DeleteLabelUsecase {
	return &DeleteLabelUsecase{
		labelRepo:    labelRepo,
		auditLogRepo: auditLogRepo,
	}
}

func (uc *DeleteLabelUsecase) Execute(ctx context.Context, input DeleteLabelInput) error {
	label, err := uc.labelRepo.GetByName(ctx, input.RepositoryID, input.Name)
	if err != nil {
		return err
	}
	if label == nil {
		return apperror.ErrNotFound
	}

	if err := uc.labelRepo.Delete(ctx, label.ID); err != nil {
		return err
	}

	return uc.auditLogRepo.InsertAuditLog(
		ctx,
		input.OrganizationID,
		input.ActorID,
		"label.delete",
		"label",
		label.ID,
		json.RawMessage(`{}`),
	)
}
