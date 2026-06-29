package webhook

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type ListWebhooksInput struct {
	OrganizationID uuid.UUID
	RepositoryID   *uuid.UUID
	Page           int
	PerPage        int
}

type ListWebhooksOutput struct {
	Webhooks []*entity.Webhook
	Page     int
	PerPage  int
	Total    int64
}

type ListWebhooksUsecase struct {
	webhookRepo domainrepo.IWebhookRepository
}

func NewListWebhooksUsecase(webhookRepo domainrepo.IWebhookRepository) *ListWebhooksUsecase {
	return &ListWebhooksUsecase{webhookRepo: webhookRepo}
}

func (uc *ListWebhooksUsecase) Execute(ctx context.Context, input ListWebhooksInput) (*ListWebhooksOutput, error) {
	page := input.Page
	perPage := input.PerPage
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}

	var (
		webhooks []*entity.Webhook
		total    int64
		err      error
	)

	if input.RepositoryID != nil {
		webhooks, total, err = uc.webhookRepo.ListByRepo(ctx, input.OrganizationID, *input.RepositoryID, page, perPage)
	} else {
		webhooks, total, err = uc.webhookRepo.ListByOrg(ctx, input.OrganizationID, page, perPage)
	}
	if err != nil {
		return nil, err
	}

	for _, hook := range webhooks {
		hook.SecretEncrypted = nil
	}

	return &ListWebhooksOutput{
		Webhooks: webhooks,
		Page:     page,
		PerPage:  perPage,
		Total:    total,
	}, nil
}
