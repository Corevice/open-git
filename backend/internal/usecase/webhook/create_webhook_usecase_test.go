package webhook_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/crypto"
	webhookusecase "github.com/open-git/backend/internal/usecase/webhook"
)

type mockWebhookRepo struct {
	created *entity.Webhook
}

func (m *mockWebhookRepo) Create(_ context.Context, webhook *entity.Webhook) error {
	copyHook := *webhook
	m.created = &copyHook
	return nil
}

func (m *mockWebhookRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*entity.Webhook, error) {
	return nil, apperror.ErrNotFound
}

func (m *mockWebhookRepo) ListByRepo(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*entity.Webhook, int64, error) {
	return nil, 0, nil
}

func (m *mockWebhookRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Webhook, int64, error) {
	return nil, 0, nil
}

func (m *mockWebhookRepo) Update(context.Context, *entity.Webhook) error {
	return nil
}

func (m *mockWebhookRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (m *mockWebhookRepo) ListActiveByRepoAndEvent(context.Context, uuid.UUID, uuid.UUID, string) ([]*entity.Webhook, error) {
	return nil, nil
}

var _ domainrepo.IWebhookRepository = (*mockWebhookRepo)(nil)

type mockCreateAuditLogRepo struct {
	metadata json.RawMessage
	action   string
}

func (m *mockCreateAuditLogRepo) InsertAuditLog(
	_ context.Context,
	_, _ uuid.UUID,
	action, _ string,
	_ uuid.UUID,
	metadata json.RawMessage,
) error {
	m.action = action
	m.metadata = metadata
	return nil
}

func TestCreateWebhookUsecaseSuccess(t *testing.T) {
	repo := &mockWebhookRepo{}
	auditRepo := &mockCreateAuditLogRepo{}
	encryptor := crypto.NewSecretEncryptor(bytes.Repeat([]byte{0x33}, 32))
	uc := webhookusecase.NewCreateWebhookUsecase(repo, auditRepo, encryptor)

	orgID := uuid.New()
	repoID := uuid.New()
	actorID := uuid.New()

	created, err := uc.Execute(context.Background(), orgID, repoID, webhookusecase.CreateWebhookInput{
		ActorID:     actorID,
		URL:         "https://example.com/hook",
		ContentType: entity.ContentTypeJSON,
		Secret:      "super-secret",
		Events:      []string{"push"},
		Active:      true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if created.SecretEncrypted != nil {
		t.Fatalf("expected response without secret")
	}
	if repo.created == nil {
		t.Fatalf("expected webhook to be stored")
	}
	if len(repo.created.SecretEncrypted) == 0 {
		t.Fatalf("expected encrypted secret to be stored")
	}
	if bytes.Equal(repo.created.SecretEncrypted, []byte("super-secret")) {
		t.Fatalf("expected stored secret to be encrypted")
	}

	if auditRepo.action != "webhook.create" {
		t.Fatalf("audit action = %q, want webhook.create", auditRepo.action)
	}

	var metadata map[string]any
	if err := json.Unmarshal(auditRepo.metadata, &metadata); err != nil {
		t.Fatalf("unmarshal audit metadata: %v", err)
	}
	if metadata["secret"] != "***" {
		t.Fatalf("audit secret = %v, want masked", metadata["secret"])
	}
}

func TestCreateWebhookUsecaseInvalidURL(t *testing.T) {
	repo := &mockWebhookRepo{}
	auditRepo := &mockCreateAuditLogRepo{}
	encryptor := crypto.NewSecretEncryptor(bytes.Repeat([]byte{0x33}, 32))
	uc := webhookusecase.NewCreateWebhookUsecase(repo, auditRepo, encryptor)

	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), webhookusecase.CreateWebhookInput{
		ActorID:     uuid.New(),
		URL:         "ftp://example.com/hook",
		ContentType: entity.ContentTypeJSON,
		Secret:      "",
		Events:      []string{"push"},
		Active:      true,
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if repo.created != nil {
		t.Fatalf("expected webhook not to be stored")
	}
}
