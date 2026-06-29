package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IScanJobRepository interface {
	Create(ctx context.Context, job *entity.ScanJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.ScanJob, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.ScanJobStatus, errMsg string) error
}
