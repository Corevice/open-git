package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ISecurityAdvisoryRepository interface {
	ListByOrg(ctx context.Context, orgID uuid.UUID, state, severity string, page, perPage int) ([]*entity.SecurityAdvisory, int, error)
	GetByGHSAPID(ctx context.Context, orgID uuid.UUID, ghsaID string) (*entity.SecurityAdvisory, error)
	UpdateState(ctx context.Context, orgID uuid.UUID, ghsaID string, state entity.AdvisoryState, reason *entity.DismissedReason) (*entity.SecurityAdvisory, error)
}
