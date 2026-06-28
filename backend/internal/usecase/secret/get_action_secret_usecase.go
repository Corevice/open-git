package secret

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type GetActionSecretUsecase struct {
	secretRepo domainrepo.IActionSecretRepository
}

func NewGetActionSecretUsecase(secretRepo domainrepo.IActionSecretRepository) *GetActionSecretUsecase {
	return &GetActionSecretUsecase{secretRepo: secretRepo}
}

func (uc *GetActionSecretUsecase) Execute(
	ctx context.Context,
	orgID uuid.UUID,
	repoID *uuid.UUID,
	name string,
) (*entity.ActionSecret, error) {
	secret, err := uc.secretRepo.GetByName(ctx, orgID, repoID, name)
	if err != nil {
		return nil, err
	}

	secret.EncryptedValue = ""
	return secret, nil
}
