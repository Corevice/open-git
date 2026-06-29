package secret

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type ListRepoSecretsUsecase struct {
	secretRepo domainrepo.IActionSecretRepository
}

func NewListRepoSecretsUsecase(secretRepo domainrepo.IActionSecretRepository) *ListRepoSecretsUsecase {
	return &ListRepoSecretsUsecase{secretRepo: secretRepo}
}

func (uc *ListRepoSecretsUsecase) Execute(
	ctx context.Context,
	orgID, repoID uuid.UUID,
) ([]*entity.ActionSecret, error) {
	secrets, err := uc.secretRepo.ListByRepo(ctx, orgID, repoID)
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		secret.EncryptedValue = ""
	}

	return secrets, nil
}

type ListOrgSecretsUsecase struct {
	secretRepo domainrepo.IActionSecretRepository
}

func NewListOrgSecretsUsecase(secretRepo domainrepo.IActionSecretRepository) *ListOrgSecretsUsecase {
	return &ListOrgSecretsUsecase{secretRepo: secretRepo}
}

func (uc *ListOrgSecretsUsecase) Execute(
	ctx context.Context,
	orgID uuid.UUID,
) ([]*entity.ActionSecret, error) {
	secrets, err := uc.secretRepo.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		secret.EncryptedValue = ""
	}

	return secrets, nil
}
