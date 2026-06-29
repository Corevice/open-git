package webhook

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type DeleteWebhookInput struct {
	ActorID uuid.UUID
}

type DeleteWebhookUsecase struct {
	webhookRepo  domainrepo.IWebhookRepository
	auditLogRepo AuditLogWriter
}

func NewDeleteWebhookUsecase(
	webhookRepo domainrepo.IWebhookRepository,
	auditLogRepo AuditLogWriter,
) *DeleteWebhookUsecase {
	return &DeleteWebhookUsecase{
		webhookRepo:  webhookRepo,
		auditLogRepo: auditLogRepo,
	}
}

func (uc *DeleteWebhookUsecase) Execute(
	ctx context.Context,
	orgID, webhookID uuid.UUID,
	input DeleteWebhookInput,
) error {
	if err := uc.webhookRepo.Delete(ctx, webhookID, orgID); err != nil {
		return err
	}

	return uc.auditLogRepo.InsertAuditLog(
		ctx,
		orgID,
		input.ActorID,
		"webhook.delete",
		"webhook",
		webhookID,
		json.RawMessage(`{}`),
	)
}
