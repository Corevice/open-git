package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IMembershipRepository interface {
	Add(ctx context.Context, m *entity.Membership) error
	GetRole(ctx context.Context, orgID, userID uuid.UUID) (string, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.Membership, error)
	UpdateRole(ctx context.Context, orgID, userID uuid.UUID, role string) error
	Remove(ctx context.Context, orgID, userID uuid.UUID) error
}
