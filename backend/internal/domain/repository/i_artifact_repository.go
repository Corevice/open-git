package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IArtifactRepository interface {
	Create(ctx context.Context, artifact *entity.Artifact) error
	GetByID(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*entity.Artifact, error)
	ListByRun(ctx context.Context, runID uuid.UUID, orgID uuid.UUID) ([]*entity.Artifact, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.ArtifactStatus, sizeInBytes int64) error
	SoftDelete(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error
	ListExpired(ctx context.Context, limit int) ([]*entity.Artifact, error)
	DeleteByRunID(ctx context.Context, runID uuid.UUID) error
}
