package repository

import (
	"context"

	"github.com/Corevice/open-git/backend/internal/domain"
)

type IAccessTokenRepository interface {
	Create(ctx context.Context, token *domain.AccessToken) error
	ListByUserID(ctx context.Context, userID int64) ([]*domain.AccessToken, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.AccessToken, error)
	Revoke(ctx context.Context, tokenID, userID int64) error
}

type IUserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByLogin(ctx context.Context, login string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

type IRepositoryRepository interface {
	Create(ctx context.Context, repo *domain.Repository) error
	GetByOwnerAndName(ctx context.Context, ownerID int64, name string) (*domain.Repository, error)
	GetByOwnerLoginAndName(ctx context.Context, ownerLogin, name string) (*domain.Repository, error)
	ListByOrg(ctx context.Context, orgID int64) ([]*domain.Repository, error)
	UpdateVisibility(ctx context.Context, id int64, visibility domain.Visibility) error
	Delete(ctx context.Context, id int64) error
	NextNumber(ctx context.Context, ownerID int64) (int64, error)
}

type IMembershipRepository interface {
	HasReadAccess(ctx context.Context, userID, orgID int64) (bool, error)
}

type IOAuthAppRepository interface {
	GetByClientID(ctx context.Context, clientID string) (*domain.OAuthApp, error)
}
