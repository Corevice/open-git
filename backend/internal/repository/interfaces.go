package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
)

type IUserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByLogin(ctx context.Context, login string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

type IAccessTokenRepository interface {
	Create(ctx context.Context, token *domain.AccessToken) error
	ListByUserID(ctx context.Context, userID int64) ([]*domain.AccessToken, error)
	Revoke(ctx context.Context, tokenID, userID int64) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.AccessToken, error)
}

type IRepositoryRepository interface {
	Create(ctx context.Context, repo *entity.Repository) error
	GetByOwnerAndName(ctx context.Context, ownerID uuid.UUID, name string) (*entity.Repository, error)
	GetByOwnerLoginAndName(ctx context.Context, ownerLogin, name string) (*entity.Repository, error)
	ListByOrg(ctx context.Context, organizationID uuid.UUID, page, perPage int) ([]*entity.Repository, error)
	UpdateVisibility(ctx context.Context, id uuid.UUID, visibility string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type IMembershipRepository interface {
	HasReadAccess(ctx context.Context, userID, organizationID uuid.UUID) (bool, error)
	HasWriteAccess(ctx context.Context, userID, organizationID uuid.UUID) (bool, error)
}

type IOAuthAppRepository interface {
	GetByClientID(ctx context.Context, clientID string) (*domain.OAuthApp, error)
}

type IOrganizationRepository interface {
	// GetByLogin returns nil, nil when the organization is not found.
	GetByLogin(ctx context.Context, login string) (*domain.Organization, error)
	ListByUserID(ctx context.Context, userID int64) ([]*domain.Organization, error)
	GetMemberRole(ctx context.Context, orgID, userID int64) (string, error)
}

type ISSHKeyStore interface {
	FindByFingerprint(ctx context.Context, fingerprint string) (*entity.SSHKey, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.SSHKey, error)
	Create(ctx context.Context, key *entity.SSHKey) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}
