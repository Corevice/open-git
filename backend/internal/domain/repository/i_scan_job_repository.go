package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IScanJobRepository interface {
	Create(ctx context.Context, job *entity.ScanJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.ScanJob, error)
	ListByRepo(ctx context.Context, orgID, repoID uuid.UUID, page, perPage int) ([]*entity.ScanJob, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	Upsert(ctx context.Context, job *entity.ScanJob) error
}
