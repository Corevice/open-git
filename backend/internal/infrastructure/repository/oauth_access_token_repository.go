package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
	appmiddleware "github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
)

type sqlxOAuthAccessTokenRepository struct {
	*sqlx.DB
}

func NewOAuthAccessTokenRepository(db *sqlx.DB) *sqlxOAuthAccessTokenRepository {
	return &sqlxOAuthAccessTokenRepository{DB: db}
}

var _ repo.IOAuthAccessTokenRepository = (*sqlxOAuthAccessTokenRepository)(nil)

const oauthAccessTokenSelectColumns = `
	id, token_hash, oauth_app_id, user_id, scopes, revoked_at, last_used_at, created_at
`

func (r *sqlxOAuthAccessTokenRepository) Create(ctx context.Context, token *domain.OAuthAccessToken) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}

	scopes, err := marshalScopes(r.DriverName(), token.Scopes)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO oauth_access_tokens (
			id, token_hash, oauth_app_id, user_id, scopes, revoked_at, last_used_at, created_at
		)
		VALUES (
			:id, :token_hash, :oauth_app_id, :user_id, :scopes, :revoked_at, :last_used_at, :created_at
		)
	`

	_, err = r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":           token.ID,
		"token_hash":   token.TokenHash,
		"oauth_app_id": token.OAuthAppID,
		"user_id":      formatTokenID(token.UserID),
		"scopes":       scopes,
		"revoked_at":   token.RevokedAt,
		"last_used_at": token.LastUsedAt,
		"created_at":   token.CreatedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxOAuthAccessTokenRepository) FindByTokenHash(ctx context.Context, hash string) (*domain.OAuthAccessToken, error) {
	query := `
		SELECT ` + oauthAccessTokenSelectColumns + `
		FROM oauth_access_tokens
		WHERE token_hash = ? AND revoked_at IS NULL
	`
	query = r.DB.Rebind(query)

	row := r.DB.QueryRowxContext(ctx, query, hash)
	token, err := scanOAuthAccessToken(row, r.DriverName())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return token, nil
}

func (r *sqlxOAuthAccessTokenRepository) RevokeByUserAndApp(ctx context.Context, userID int64, appID string) error {
	now := time.Now().UTC()
	query := `
		UPDATE oauth_access_tokens
		SET revoked_at = ?
		WHERE user_id = ? AND oauth_app_id = ? AND revoked_at IS NULL
	`
	query = r.DB.Rebind(query)

	_, err := r.DB.ExecContext(ctx, query, now, formatTokenID(userID), appID)
	return dbErrors.MapDBError(err)
}

func (r *sqlxOAuthAccessTokenRepository) RevokeAllByAppID(ctx context.Context, appID string, ownerUserID int64) error {
	now := time.Now().UTC()
	ownerID := appmiddleware.Int64ToUUID(ownerUserID).String()
	query := `
		UPDATE oauth_access_tokens
		SET revoked_at = ?
		WHERE oauth_app_id = ?
		  AND revoked_at IS NULL
		  AND EXISTS (
			SELECT 1 FROM oauth_apps
			WHERE oauth_apps.id = oauth_access_tokens.oauth_app_id
			  AND oauth_apps.owner_id = ?
		  )
	`
	query = r.DB.Rebind(query)

	_, err := r.DB.ExecContext(ctx, query, now, appID, ownerID)
	return dbErrors.MapDBError(err)
}

type oauthAccessTokenRow interface {
	Scan(dest ...any) error
}

func scanOAuthAccessToken(row oauthAccessTokenRow, driver string) (*domain.OAuthAccessToken, error) {
	var (
		token      domain.OAuthAccessToken
		userIDRaw  string
		scopesRaw  any
		revokedAt  sql.NullTime
		lastUsedAt sql.NullTime
	)

	if err := row.Scan(
		&token.ID,
		&token.TokenHash,
		&token.OAuthAppID,
		&userIDRaw,
		&scopesRaw,
		&revokedAt,
		&lastUsedAt,
		&token.CreatedAt,
	); err != nil {
		return nil, err
	}

	userID, err := parseTokenID(userIDRaw)
	if err != nil {
		return nil, err
	}
	token.UserID = userID

	scopes, err := unmarshalScopes(driver, scopesRaw)
	if err != nil {
		return nil, err
	}
	token.Scopes = scopes

	if revokedAt.Valid {
		t := revokedAt.Time
		token.RevokedAt = &t
	}
	if lastUsedAt.Valid {
		t := lastUsedAt.Time
		token.LastUsedAt = &t
	}

	return &token, nil
}
