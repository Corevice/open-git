package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ISecurityAdvisoryRepository interface {
	Create(ctx context.Context, advisory *entity.SecurityAdvisory) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.SecurityAdvisory, error)
	GetByGHSAID(ctx context.Context, orgID uuid.UUID, ghsaID string) (*entity.SecurityAdvisory, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, state, severity string, page, perPage int) ([]*entity.SecurityAdvisory, int, error)
	UpdateState(ctx context.Context, id uuid.UUID, state, dismissedReason string) error
	Upsert(ctx context.Context, advisory *entity.SecurityAdvisory) error
}
