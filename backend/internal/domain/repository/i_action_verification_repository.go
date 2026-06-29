package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IActionVerificationRepository interface {
	Create(ctx context.Context, v *entity.ActionVerification) error
	Update(ctx context.Context, v *entity.ActionVerification) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.ActionVerification, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entity.ActionVerification, error)
}
