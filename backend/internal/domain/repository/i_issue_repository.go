package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IIssueRepository interface {
	Create(ctx context.Context, issue *entity.Issue) error
	GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.Issue, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Issue, error)
	ListByRepo(ctx context.Context, filter ListIssuesFilter) ([]*entity.Issue, int, error)
	Update(ctx context.Context, issue *entity.Issue) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context, filter ListIssuesFilter) (int, error)
	NextNumber(ctx context.Context, repoID uuid.UUID) (int, error)
}
