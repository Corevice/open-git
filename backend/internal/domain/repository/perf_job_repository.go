package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type PerfJobRepository interface {
	Create(ctx context.Context, job *entity.PerfJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.PerfJob, error)
	GetActiveJob(ctx context.Context) (*entity.PerfJob, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.JobStatus, benchmarkID *uuid.UUID) error
}
