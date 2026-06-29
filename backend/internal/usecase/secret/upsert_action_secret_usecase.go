package secret

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

const maxSecretPlaintextSize = 65536

type SecretVisibility string

const (
	VisibilityAll      SecretVisibility = "all"
	VisibilityPrivate  SecretVisibility = "private"
	VisibilitySelected SecretVisibility = "selected"
)

type SecretEncryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
	KeyID() string
	PublicKeyBase64() string
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

type UpsertActionSecretInput struct {
	ActorID          uuid.UUID
	Name             string
	PlaintextValue   string
	Visibility       SecretVisibility
	SelectedRepoIDs  []uuid.UUID
}

type UpsertActionSecretUsecase struct {
	secretRepo   domainrepo.IActionSecretRepository
	auditLogRepo AuditLogWriter
	enc          SecretEncryptor
}

func NewUpsertActionSecretUsecase(
	secretRepo domainrepo.IActionSecretRepository,
	auditLogRepo AuditLogWriter,
	enc SecretEncryptor,
) *UpsertActionSecretUsecase {
	return &UpsertActionSecretUsecase{
		secretRepo:   secretRepo,
		auditLogRepo: auditLogRepo,
		enc:          enc,
	}
}

func (uc *UpsertActionSecretUsecase) Execute(
	ctx context.Context,
	orgID uuid.UUID,
	repoID *uuid.UUID,
	input UpsertActionSecretInput,
) (created bool, err error) {
	if err := (&entity.ActionSecret{Name: input.Name}).Validate(); err != nil {
		return false, fmt.Errorf("%w: %s", apperror.ErrValidation, err.Error())
	}
	if len(input.PlaintextValue) == 0 {
		return false, fmt.Errorf("%w: secret value is required", apperror.ErrValidation)
	}
	if len(input.PlaintextValue) > maxSecretPlaintextSize {
		return false, fmt.Errorf("%w: secret value exceeds maximum size", apperror.ErrValidation)
	}

	encrypted, err := uc.enc.Encrypt([]byte(input.PlaintextValue))
	if err != nil {
		return false, err
	}

	secret := &entity.ActionSecret{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           input.Name,
		EncryptedValue: string(encrypted),
		KeyID:          uc.enc.KeyID(),
		Visibility:     string(input.Visibility),
	}
	if repoID != nil {
		secret.RepositoryID = *repoID
	}

	created, err = uc.secretRepo.Upsert(ctx, secret)
	if err != nil {
		return false, err
	}

	if repoID == nil && input.Visibility == VisibilitySelected {
		if err := uc.secretRepo.SetSelectedRepositories(ctx, orgID, secret.ID, input.SelectedRepoIDs); err != nil {
			return false, err
		}
	}

	action := "secret.update"
	if created {
		action = "secret.create"
	}

	metadata, err := json.Marshal(map[string]any{
		"name":  input.Name,
		"scope": secretScope(repoID),
	})
	if err != nil {
		return false, err
	}

	if err := uc.auditLogRepo.InsertAuditLog(
		ctx,
		orgID,
		input.ActorID,
		action,
		"secret",
		secret.ID,
		metadata,
	); err != nil {
		return false, err
	}

	secret.EncryptedValue = ""
	return created, nil
}

func secretScope(repoID *uuid.UUID) string {
	if repoID != nil {
		return "repo"
	}
	return "org"
}
