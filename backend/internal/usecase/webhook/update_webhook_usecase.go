package webhook

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/crypto"
)

type UpdateWebhookInput struct {
	ActorID     uuid.UUID
	URL         *string
	ContentType *string
	Secret      *string
	Events      []string
	Active      *bool
}

type UpdateWebhookUsecase struct {
	webhookRepo  domainrepo.IWebhookRepository
	auditLogRepo AuditLogWriter
	encryptor    *crypto.SecretEncryptor
}

func NewUpdateWebhookUsecase(
	webhookRepo domainrepo.IWebhookRepository,
	auditLogRepo AuditLogWriter,
	encryptor *crypto.SecretEncryptor,
) *UpdateWebhookUsecase {
	return &UpdateWebhookUsecase{
		webhookRepo:  webhookRepo,
		auditLogRepo: auditLogRepo,
		encryptor:    encryptor,
	}
}

func (uc *UpdateWebhookUsecase) Execute(
	ctx context.Context,
	orgID, webhookID uuid.UUID,
	input UpdateWebhookInput,
) (*entity.Webhook, error) {
	webhook, err := uc.webhookRepo.GetByID(ctx, webhookID, orgID)
	if err != nil {
		return nil, err
	}

	before := webhookSnapshot(webhook)

	if input.URL != nil {
		webhook.URL = *input.URL
	}
	if input.ContentType != nil {
		webhook.ContentType = *input.ContentType
	}
	if input.Events != nil {
		if err := validateEvents(input.Events); err != nil {
			return nil, err
		}
		webhook.Events = input.Events
	}
	if input.Active != nil {
		webhook.Active = *input.Active
	}

	secretChanged := false
	if input.Secret != nil && *input.Secret != "" {
		encrypted, err := uc.encryptor.Encrypt([]byte(*input.Secret))
		if err != nil {
			return nil, err
		}
		webhook.SecretEncrypted = encrypted
		secretChanged = true
	}

	if err := webhook.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", apperror.ErrValidation, err.Error())
	}

	if err := uc.webhookRepo.Update(ctx, webhook); err != nil {
		return nil, err
	}

	after := webhookSnapshot(webhook)

	if secretChanged {
		secretMetadata, err := json.Marshal(map[string]any{
			"secret": maskSecretValue(*input.Secret),
		})
		if err != nil {
			return nil, err
		}
		if err := uc.auditLogRepo.InsertAuditLog(
			ctx,
			orgID,
			input.ActorID,
			"webhook.secret_change",
			"webhook",
			webhook.ID,
			secretMetadata,
		); err != nil {
			return nil, err
		}
	}

	updateMetadata, err := json.Marshal(map[string]any{
		"before": before,
		"after":  after,
	})
	if err != nil {
		return nil, err
	}

	if err := uc.auditLogRepo.InsertAuditLog(
		ctx,
		orgID,
		input.ActorID,
		"webhook.update",
		"webhook",
		webhook.ID,
		updateMetadata,
	); err != nil {
		return nil, err
	}

	webhook.SecretEncrypted = nil
	return webhook, nil
}
