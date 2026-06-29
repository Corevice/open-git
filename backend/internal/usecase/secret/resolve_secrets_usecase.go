package secret

import (
	"context"

	"github.com/google/uuid"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type ResolveSecretsUsecase struct {
	secretRepo domainrepo.IActionSecretRepository
	enc        SecretEncryptor
}

func NewResolveSecretsUsecase(
	secretRepo domainrepo.IActionSecretRepository,
	enc SecretEncryptor,
) *ResolveSecretsUsecase {
	return &ResolveSecretsUsecase{
		secretRepo: secretRepo,
		enc:        enc,
	}
}

func (uc *ResolveSecretsUsecase) Execute(
	ctx context.Context,
	orgID, repoID uuid.UUID,
) (map[string]string, error) {
	secrets, err := uc.secretRepo.ListForWorkflow(ctx, orgID, repoID)
	if err != nil {
		return nil, err
	}

	orgSecrets := make(map[string]string)
	repoSecrets := make(map[string]string)
	for _, secret := range secrets {
		plaintext, err := uc.enc.Decrypt([]byte(secret.EncryptedValue))
		if err != nil {
			return nil, err
		}
		if secret.RepositoryID == uuid.Nil {
			orgSecrets[secret.Name] = string(plaintext)
			continue
		}
		repoSecrets[secret.Name] = string(plaintext)
	}

	resolved := make(map[string]string, len(orgSecrets)+len(repoSecrets))
	for name, value := range orgSecrets {
		resolved[name] = value
	}
	for name, value := range repoSecrets {
		resolved[name] = value
	}

	return resolved, nil
}
