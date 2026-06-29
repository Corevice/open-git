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

type CreateWebhookInput struct {
	ActorID     uuid.UUID
	URL         string
	ContentType string
	Secret      string
	Events      []string
	Active      bool
}

type AuditLogWriter interface {
	InsertAuditLog(
		ctx context.Context,
		orgID, actorID uuid.UUID,
		action, targetType string,
		targetID uuid.UUID,
		metadata json.RawMessage,
	) error
}

type CreateWebhookUsecase struct {
	webhookRepo  domainrepo.IWebhookRepository
	auditLogRepo AuditLogWriter
	encryptor    *crypto.SecretEncryptor
}

func NewCreateWebhookUsecase(
	webhookRepo domainrepo.IWebhookRepository,
	auditLogRepo AuditLogWriter,
	encryptor *crypto.SecretEncryptor,
) *CreateWebhookUsecase {
	return &CreateWebhookUsecase{
		webhookRepo:  webhookRepo,
		auditLogRepo: auditLogRepo,
		encryptor:    encryptor,
	}
}

func (uc *CreateWebhookUsecase) Execute(
	ctx context.Context,
	orgID, repoID uuid.UUID,
	input CreateWebhookInput,
) (*entity.Webhook, error) {
	if err := validateEvents(input.Events); err != nil {
		return nil, err
	}

	repoIDCopy := repoID
	webhook := &entity.Webhook{
		ID:             uuid.New(),
		OrganizationID: orgID,
		RepositoryID:   &repoIDCopy,
		URL:            input.URL,
		ContentType:    input.ContentType,
		Events:         input.Events,
		Active:         input.Active,
	}

	if input.Secret != "" {
		encrypted, err := uc.encryptor.Encrypt([]byte(input.Secret))
		if err != nil {
			return nil, err
		}
		webhook.SecretEncrypted = encrypted
	}

	if err := webhook.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", apperror.ErrValidation, err.Error())
	}

	if err := uc.webhookRepo.Create(ctx, webhook); err != nil {
		return nil, err
	}

	metadata, err := json.Marshal(map[string]any{
		"url":          webhook.URL,
		"content_type": webhook.ContentType,
		"events":       webhook.Events,
		"active":       webhook.Active,
		"secret":       maskSecretValue(input.Secret),
	})
	if err != nil {
		return nil, err
	}

	if err := uc.auditLogRepo.InsertAuditLog(
		ctx,
		orgID,
		input.ActorID,
		"webhook.create",
		"webhook",
		webhook.ID,
		metadata,
	); err != nil {
		return nil, err
	}

	webhook.SecretEncrypted = nil
	return webhook, nil
}

var allowedWebhookEvents = map[string]struct{}{
	"*":             {},
	"push":          {},
	"pull_request":  {},
	"issues":        {},
	"issue_comment": {},
	"release":       {},
	"create":        {},
	"delete":        {},
	"workflow_run":  {},
}

func validateEvents(events []string) error {
	if len(events) < 1 {
		return fmt.Errorf("%w: events must contain at least one entry", apperror.ErrValidation)
	}

	hasWildcard := false
	for _, event := range events {
		if event == "*" {
			hasWildcard = true
			continue
		}
		if _, ok := allowedWebhookEvents[event]; !ok {
			return fmt.Errorf("%w: unknown event name: %s", apperror.ErrValidation, event)
		}
	}

	if hasWildcard && len(events) > 1 {
		return fmt.Errorf("%w: events cannot mix wildcard with specific events", apperror.ErrValidation)
	}

	return nil
}

func maskSecretValue(secret string) string {
	if secret == "" {
		return ""
	}
	return "***"
}

func maskSecretFromBytes(secret []byte) string {
	if len(secret) == 0 {
		return ""
	}
	return "***"
}

func webhookSnapshot(webhook *entity.Webhook) map[string]any {
	return map[string]any{
		"url":          webhook.URL,
		"content_type": webhook.ContentType,
		"events":       webhook.Events,
		"active":       webhook.Active,
		"secret":       maskSecretFromBytes(webhook.SecretEncrypted),
	}
}
