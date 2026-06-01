package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IIssueRepository interface {
	Create(ctx context.Context, issue *entity.Issue) error
	GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.Issue, error)
	ListByRepo(ctx context.Context, repoID uuid.UUID, state, labels string, page, perPage int) ([]*entity.Issue, error)
	NextNumber(ctx context.Context, repoID uuid.UUID) (int, error)
}
