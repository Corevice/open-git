package webhook

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type ListDeliveriesUsecase struct {
	webhookRepo  domainrepo.IWebhookRepository
	deliveryRepo domainrepo.IWebhookDeliveryRepository
}

func NewListDeliveriesUsecase(
	webhookRepo domainrepo.IWebhookRepository,
	deliveryRepo domainrepo.IWebhookDeliveryRepository,
) *ListDeliveriesUsecase {
	return &ListDeliveriesUsecase{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
	}
}

func (uc *ListDeliveriesUsecase) Execute(
	ctx context.Context,
	webhookID, orgID uuid.UUID,
	page, perPage int,
) ([]*entity.WebhookDelivery, int64, error) {
	if _, err := uc.webhookRepo.GetByID(ctx, webhookID, orgID); err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}

	return uc.deliveryRepo.ListByWebhook(ctx, webhookID, orgID, page, perPage)
}
