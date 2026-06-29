package security

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type GetAdvisoryInput struct {
	OrganizationID uuid.UUID
	GHSAPID        string
}

type GetAdvisoryUsecase struct {
	advisoryRepo repository.ISecurityAdvisoryRepository
}

func NewGetAdvisoryUsecase(advisoryRepo repository.ISecurityAdvisoryRepository) *GetAdvisoryUsecase {
	return &GetAdvisoryUsecase{advisoryRepo: advisoryRepo}
}

func (uc *GetAdvisoryUsecase) Execute(ctx context.Context, input GetAdvisoryInput) (*entity.SecurityAdvisory, error) {
	advisory, err := uc.advisoryRepo.GetByGHSAPID(ctx, input.OrganizationID, input.GHSAPID)
	if err != nil {
		return nil, err
	}
	if advisory == nil {
		return nil, apperror.ErrNotFound
	}
	return advisory, nil
}
