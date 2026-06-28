package workflow

import (
	"context"

	"github.com/google/uuid"
)

type ListArtifactsInput struct {
	OrganizationID uuid.UUID
	RunID          uuid.UUID
}

type ListArtifactsUsecase struct {
	artifactRepo ArtifactRepository
}

func NewListArtifactsUsecase(artifactRepo ArtifactRepository) *ListArtifactsUsecase {
	return &ListArtifactsUsecase{artifactRepo: artifactRepo}
}

func (uc *ListArtifactsUsecase) Execute(ctx context.Context, input ListArtifactsInput) ([]*Artifact, error) {
	return uc.artifactRepo.ListByRunID(ctx, input.RunID, input.OrganizationID)
}
