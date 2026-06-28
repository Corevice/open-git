package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ILabelRepository interface {
	Create(ctx context.Context, label *entity.Label) error
	GetByName(ctx context.Context, repoID uuid.UUID, name string) (*entity.Label, error)
	ListByRepo(ctx context.Context, repoID uuid.UUID, page, perPage int) ([]*entity.Label, int, error)
	Update(ctx context.Context, label *entity.Label) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddToIssue(ctx context.Context, repoID uuid.UUID, issueNumber int, labelID uuid.UUID) error
	RemoveFromIssue(ctx context.Context, repoID uuid.UUID, issueNumber int, labelID uuid.UUID) error
}
