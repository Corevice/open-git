package webhook

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type GetWebhookUsecase struct {
	webhookRepo domainrepo.IWebhookRepository
}

func NewGetWebhookUsecase(webhookRepo domainrepo.IWebhookRepository) *GetWebhookUsecase {
	return &GetWebhookUsecase{webhookRepo: webhookRepo}
}

func (uc *GetWebhookUsecase) Execute(ctx context.Context, orgID, webhookID uuid.UUID) (*entity.Webhook, error) {
	webhook, err := uc.webhookRepo.GetByID(ctx, webhookID, orgID)
	if err != nil {
		return nil, err
	}

	webhook.SecretEncrypted = nil
	return webhook, nil
}
