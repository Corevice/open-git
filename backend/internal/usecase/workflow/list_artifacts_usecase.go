package workflow

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type ListArtifactsInput struct {
	OrganizationID uuid.UUID
	RunID          uuid.UUID
}

type ListArtifactsUsecase struct {
	artifactRepo repository.IArtifactRepository
}

func NewListArtifactsUsecase(artifactRepo repository.IArtifactRepository) *ListArtifactsUsecase {
	return &ListArtifactsUsecase{artifactRepo: artifactRepo}
}

func (uc *ListArtifactsUsecase) Execute(ctx context.Context, input ListArtifactsInput) ([]*entity.Artifact, error) {
	return uc.artifactRepo.ListByRunID(ctx, input.RunID, input.OrganizationID)
}
