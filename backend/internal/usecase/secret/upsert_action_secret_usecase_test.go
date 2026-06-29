package secret_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/crypto"
	secretusecase "github.com/open-git/backend/internal/usecase/secret"
)

type mockActionSecretRepo struct {
	upserted       *entity.ActionSecret
	upsertCreated  bool
	selectedRepoIDs []uuid.UUID
}

func (m *mockActionSecretRepo) Upsert(_ context.Context, secret *entity.ActionSecret) (bool, error) {
	copySecret := *secret
	m.upserted = &copySecret
	return m.upsertCreated, nil
}

func (m *mockActionSecretRepo) GetByName(context.Context, uuid.UUID, *uuid.UUID, string) (*entity.ActionSecret, error) {
	return nil, apperror.ErrNotFound
}

func (m *mockActionSecretRepo) ListByRepo(context.Context, uuid.UUID, uuid.UUID) ([]*entity.ActionSecret, error) {
	return nil, nil
}

func (m *mockActionSecretRepo) ListByOrg(context.Context, uuid.UUID) ([]*entity.ActionSecret, error) {
	return nil, nil
}

func (m *mockActionSecretRepo) Delete(context.Context, uuid.UUID, *uuid.UUID, string) error {
	return nil
}

func (m *mockActionSecretRepo) ListForWorkflow(context.Context, uuid.UUID, uuid.UUID) ([]*entity.ActionSecret, error) {
	return nil, nil
}

func (m *mockActionSecretRepo) SetSelectedRepositories(_ context.Context, _, _ uuid.UUID, repoIDs []uuid.UUID) error {
	m.selectedRepoIDs = append([]uuid.UUID(nil), repoIDs...)
	return nil
}

func (m *mockActionSecretRepo) GetSelectedRepositories(context.Context, uuid.UUID, uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

var _ domainrepo.IActionSecretRepository = (*mockActionSecretRepo)(nil)

type mockUpsertAuditLogRepo struct {
	action   string
	metadata json.RawMessage
}

func (m *mockUpsertAuditLogRepo) InsertAuditLog(
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

type mockSecretEncryptor struct {
	key []byte
}

func (m *mockSecretEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	enc := crypto.NewSecretEncryptor(m.key)
	return enc.Encrypt(plaintext)
}

func (m *mockSecretEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	enc := crypto.NewSecretEncryptor(m.key)
	return enc.Decrypt(ciphertext)
}

func (m *mockSecretEncryptor) KeyID() string {
	return "test-key-id"
}

func (m *mockSecretEncryptor) PublicKeyBase64() string {
	return "dGVzdC1wdWJsaWNLZXk="
}

var _ secretusecase.SecretEncryptor = (*mockSecretEncryptor)(nil)

func TestUpsertActionSecretUsecaseSuccess(t *testing.T) {
	repo := &mockActionSecretRepo{upsertCreated: true}
	auditRepo := &mockUpsertAuditLogRepo{}
	enc := &mockSecretEncryptor{key: bytes.Repeat([]byte{0x11}, 32)}
	uc := secretusecase.NewUpsertActionSecretUsecase(repo, auditRepo, enc)

	orgID := uuid.New()
	repoID := uuid.New()
	actorID := uuid.New()
	plaintext := "super-secret-value"

	created, err := uc.Execute(context.Background(), orgID, &repoID, secretusecase.UpsertActionSecretInput{
		ActorID:        actorID,
		Name:           "MY_SECRET",
		PlaintextValue: plaintext,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !created {
		t.Fatalf("expected created=true")
	}
	if repo.upserted == nil {
		t.Fatalf("expected secret to be upserted")
	}
	if repo.upserted.EncryptedValue == plaintext {
		t.Fatalf("expected encrypted value to be stored")
	}
	if repo.upserted.EncryptedValue == "" {
		t.Fatalf("expected encrypted value to be stored")
	}
	if repo.upserted.KeyID != "test-key-id" {
		t.Fatalf("key id = %q, want test-key-id", repo.upserted.KeyID)
	}

	decrypted, err := enc.Decrypt([]byte(repo.upserted.EncryptedValue))
	if err != nil {
		t.Fatalf("decrypt stored value: %v", err)
	}
	if string(decrypted) != plaintext {
		t.Fatalf("decrypted value = %q, want %q", string(decrypted), plaintext)
	}

	if auditRepo.action != "secret.create" {
		t.Fatalf("audit action = %q, want secret.create", auditRepo.action)
	}
	assertAuditMetadataHasNoValue(t, auditRepo.metadata, plaintext)
}

func TestUpsertActionSecretUsecaseUpdate(t *testing.T) {
	repo := &mockActionSecretRepo{upsertCreated: false}
	auditRepo := &mockUpsertAuditLogRepo{}
	enc := &mockSecretEncryptor{key: bytes.Repeat([]byte{0x22}, 32)}
	uc := secretusecase.NewUpsertActionSecretUsecase(repo, auditRepo, enc)

	_, err := uc.Execute(context.Background(), uuid.New(), nil, secretusecase.UpsertActionSecretInput{
		ActorID:        uuid.New(),
		Name:           "ORG_SECRET",
		PlaintextValue: "org-value",
		Visibility:     secretusecase.VisibilityAll,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if auditRepo.action != "secret.update" {
		t.Fatalf("audit action = %q, want secret.update", auditRepo.action)
	}
	assertAuditMetadataHasNoValue(t, auditRepo.metadata, "org-value")
}

func TestUpsertActionSecretUsecaseValueTooLarge(t *testing.T) {
	repo := &mockActionSecretRepo{}
	auditRepo := &mockUpsertAuditLogRepo{}
	enc := &mockSecretEncryptor{key: bytes.Repeat([]byte{0x33}, 32)}
	uc := secretusecase.NewUpsertActionSecretUsecase(repo, auditRepo, enc)

	_, err := uc.Execute(context.Background(), uuid.New(), nil, secretusecase.UpsertActionSecretInput{
		ActorID:        uuid.New(),
		Name:           "LARGE_SECRET",
		PlaintextValue: strings.Repeat("a", 65537),
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if repo.upserted != nil {
		t.Fatalf("expected secret not to be upserted")
	}
}

func TestUpsertActionSecretUsecaseReservedPrefix(t *testing.T) {
	repo := &mockActionSecretRepo{}
	auditRepo := &mockUpsertAuditLogRepo{}
	enc := &mockSecretEncryptor{key: bytes.Repeat([]byte{0x44}, 32)}
	uc := secretusecase.NewUpsertActionSecretUsecase(repo, auditRepo, enc)

	_, err := uc.Execute(context.Background(), uuid.New(), nil, secretusecase.UpsertActionSecretInput{
		ActorID:        uuid.New(),
		Name:           "GITHUB_FOO",
		PlaintextValue: "value",
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if repo.upserted != nil {
		t.Fatalf("expected secret not to be upserted")
	}
}

func assertAuditMetadataHasNoValue(t *testing.T, metadata json.RawMessage, plaintext string) {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal(metadata, &payload); err != nil {
		t.Fatalf("unmarshal audit metadata: %v", err)
	}
	if strings.Contains(string(metadata), plaintext) {
		t.Fatalf("audit metadata must not contain secret value")
	}
	if payload["name"] == "" {
		t.Fatalf("expected audit metadata to include secret name")
	}
	if payload["scope"] == "" {
		t.Fatalf("expected audit metadata to include scope")
	}
}
