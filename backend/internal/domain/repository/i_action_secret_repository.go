package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IActionSecretRepository interface {
	Upsert(ctx context.Context, secret *entity.ActionSecret) (created bool, err error)
	GetByName(ctx context.Context, orgID uuid.UUID, repoID *uuid.UUID, name string) (*entity.ActionSecret, error)
	ListByRepo(ctx context.Context, orgID, repoID uuid.UUID) ([]*entity.ActionSecret, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entity.ActionSecret, error)
	Delete(ctx context.Context, orgID uuid.UUID, repoID *uuid.UUID, name string) error
	ListForWorkflow(ctx context.Context, orgID, repoID uuid.UUID) ([]*entity.ActionSecret, error)
	SetSelectedRepositories(ctx context.Context, orgID, secretID uuid.UUID, repoIDs []uuid.UUID) error
	GetSelectedRepositories(ctx context.Context, orgID, secretID uuid.UUID) ([]uuid.UUID, error)
}
