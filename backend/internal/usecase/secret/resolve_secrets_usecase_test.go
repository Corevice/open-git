package secret_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/crypto"
	secretusecase "github.com/open-git/backend/internal/usecase/secret"
)

type mockResolveSecretRepo struct {
	secrets []*entity.ActionSecret
}

func (m *mockResolveSecretRepo) Upsert(context.Context, *entity.ActionSecret) (bool, error) {
	return false, nil
}

func (m *mockResolveSecretRepo) GetByName(context.Context, uuid.UUID, *uuid.UUID, string) (*entity.ActionSecret, error) {
	return nil, apperror.ErrNotFound
}

func (m *mockResolveSecretRepo) ListByRepo(context.Context, uuid.UUID, uuid.UUID) ([]*entity.ActionSecret, error) {
	return nil, nil
}

func (m *mockResolveSecretRepo) ListByOrg(context.Context, uuid.UUID) ([]*entity.ActionSecret, error) {
	return nil, nil
}

func (m *mockResolveSecretRepo) Delete(context.Context, uuid.UUID, *uuid.UUID, string) error {
	return nil
}

func (m *mockResolveSecretRepo) ListForWorkflow(context.Context, uuid.UUID, uuid.UUID) ([]*entity.ActionSecret, error) {
	return m.secrets, nil
}

func (m *mockResolveSecretRepo) SetSelectedRepositories(context.Context, uuid.UUID, uuid.UUID, []uuid.UUID) error {
	return nil
}

func (m *mockResolveSecretRepo) GetSelectedRepositories(context.Context, uuid.UUID, uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

var _ domainrepo.IActionSecretRepository = (*mockResolveSecretRepo)(nil)

func TestResolveSecretsUsecaseRepoWinsOnCollision(t *testing.T) {
	key := bytes.Repeat([]byte{0x55}, 32)
	enc := crypto.NewSecretEncryptor(key)

	repoValue := "repo-value"
	orgValue := "org-value"
	otherOrgValue := "other-org-value"

	repoEncrypted, err := enc.Encrypt([]byte(repoValue))
	if err != nil {
		t.Fatalf("encrypt repo value: %v", err)
	}
	orgEncrypted, err := enc.Encrypt([]byte(orgValue))
	if err != nil {
		t.Fatalf("encrypt org value: %v", err)
	}
	otherEncrypted, err := enc.Encrypt([]byte(otherOrgValue))
	if err != nil {
		t.Fatalf("encrypt other org value: %v", err)
	}

	repoID := uuid.New()
	mockRepo := &mockResolveSecretRepo{
		secrets: []*entity.ActionSecret{
			{Name: "SHARED", EncryptedValue: string(repoEncrypted), RepositoryID: repoID},
			{Name: "REPO_ONLY", EncryptedValue: string(repoEncrypted), RepositoryID: repoID},
			{Name: "SHARED", EncryptedValue: string(orgEncrypted), RepositoryID: uuid.Nil},
			{Name: "ORG_ONLY", EncryptedValue: string(otherEncrypted), RepositoryID: uuid.Nil},
		},
	}

	uc := secretusecase.NewResolveSecretsUsecase(mockRepo, &mockSecretEncryptor{key: key})

	resolved, err := uc.Execute(context.Background(), uuid.New(), repoID)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if resolved["SHARED"] != repoValue {
		t.Fatalf("SHARED = %q, want repo value %q", resolved["SHARED"], repoValue)
	}
	if resolved["REPO_ONLY"] != repoValue {
		t.Fatalf("REPO_ONLY = %q, want %q", resolved["REPO_ONLY"], repoValue)
	}
	if resolved["ORG_ONLY"] != otherOrgValue {
		t.Fatalf("ORG_ONLY = %q, want %q", resolved["ORG_ONLY"], otherOrgValue)
	}
}
