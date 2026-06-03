package repository

import (
	"context"

	"github.com/Corevice/open-git/backend/internal/domain/entity"
	"github.com/google/uuid"
)

type ListIssuesFilter struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	State          string
	Labels         []string
	Page           int
	PerPage        int
}

type IIssueRepository interface {
	Create(ctx context.Context, issue *entity.Issue) error
	GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.Issue, error)
	ListByRepo(ctx context.Context, filter ListIssuesFilter) ([]*entity.Issue, int, error)
	NextNumber(ctx context.Context, repoID uuid.UUID) (int, error)
}
