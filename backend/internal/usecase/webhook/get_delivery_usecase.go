package webhook

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type GetDeliveryUsecase struct {
	webhookRepo  domainrepo.IWebhookRepository
	deliveryRepo domainrepo.IWebhookDeliveryRepository
}

func NewGetDeliveryUsecase(
	webhookRepo domainrepo.IWebhookRepository,
	deliveryRepo domainrepo.IWebhookDeliveryRepository,
) *GetDeliveryUsecase {
	return &GetDeliveryUsecase{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
	}
}

func (uc *GetDeliveryUsecase) Execute(
	ctx context.Context,
	deliveryID, webhookID, orgID uuid.UUID,
) (*entity.WebhookDelivery, error) {
	if _, err := uc.webhookRepo.GetByID(ctx, webhookID, orgID); err != nil {
		return nil, err
	}

	return uc.deliveryRepo.GetByID(ctx, deliveryID, webhookID, orgID)
}
