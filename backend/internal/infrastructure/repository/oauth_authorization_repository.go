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
	repo "github.com/open-git/backend/internal/repository"
)

type sqlxOAuthAuthorizationRepository struct {
	*sqlx.DB
}

func NewOAuthAuthorizationRepository(db *sqlx.DB) *sqlxOAuthAuthorizationRepository {
	return &sqlxOAuthAuthorizationRepository{DB: db}
}

var _ repo.IOAuthAuthorizationRepository = (*sqlxOAuthAuthorizationRepository)(nil)

const oauthAuthorizationSelectColumns = `
	id, oauth_app_id, user_id, granted_scopes, created_at, updated_at
`

func (r *sqlxOAuthAuthorizationRepository) Upsert(ctx context.Context, auth *domain.OAuthAuthorization) error {
	if auth.ID == "" {
		auth.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if auth.CreatedAt.IsZero() {
		auth.CreatedAt = now
	}
	auth.UpdatedAt = now

	scopes, err := marshalScopes(r.DriverName(), auth.GrantedScopes)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO oauth_authorizations (
			id, oauth_app_id, user_id, granted_scopes, created_at, updated_at
		)
		VALUES (
			:id, :oauth_app_id, :user_id, :granted_scopes, :created_at, :updated_at
		)
		ON CONFLICT(oauth_app_id, user_id) DO UPDATE SET
			granted_scopes = excluded.granted_scopes,
			updated_at = excluded.updated_at
	`

	_, err = r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":             auth.ID,
		"oauth_app_id":   auth.OAuthAppID,
		"user_id":        formatTokenID(auth.UserID),
		"granted_scopes": scopes,
		"created_at":     auth.CreatedAt,
		"updated_at":     auth.UpdatedAt,
	})
	if err != nil {
		return dbErrors.MapDBError(err)
	}

	stored, err := r.GetByUserAndApp(ctx, auth.UserID, auth.OAuthAppID)
	if err != nil {
		return err
	}
	if stored != nil {
		auth.ID = stored.ID
		auth.CreatedAt = stored.CreatedAt
		auth.UpdatedAt = stored.UpdatedAt
	}
	return nil
}

func (r *sqlxOAuthAuthorizationRepository) GetByUserAndApp(ctx context.Context, userID int64, appID string) (*domain.OAuthAuthorization, error) {
	query := `SELECT ` + oauthAuthorizationSelectColumns + ` FROM oauth_authorizations WHERE user_id = ? AND oauth_app_id = ?`
	query = r.DB.Rebind(query)

	row := r.DB.QueryRowxContext(ctx, query, formatTokenID(userID), appID)
	auth, err := scanOAuthAuthorization(row, r.DriverName())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return auth, nil
}

func (r *sqlxOAuthAuthorizationRepository) ListByUser(ctx context.Context, userID int64) ([]*domain.OAuthAuthorization, error) {
	query := `
		SELECT ` + oauthAuthorizationSelectColumns + `
		FROM oauth_authorizations
		WHERE user_id = ?
		ORDER BY created_at DESC
	`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, formatTokenID(userID))
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	return scanOAuthAuthorizations(rows, r.DriverName())
}

func (r *sqlxOAuthAuthorizationRepository) Delete(ctx context.Context, userID int64, appID string) error {
	query := `DELETE FROM oauth_authorizations WHERE user_id = ? AND oauth_app_id = ?`
	query = r.DB.Rebind(query)

	result, err := r.DB.ExecContext(ctx, query, formatTokenID(userID), appID)
	if err != nil {
		return dbErrors.MapDBError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type oauthAuthorizationRow interface {
	Scan(dest ...any) error
}

func scanOAuthAuthorization(row oauthAuthorizationRow, driver string) (*domain.OAuthAuthorization, error) {
	var (
		auth      domain.OAuthAuthorization
		userIDRaw string
		scopesRaw any
	)

	if err := row.Scan(
		&auth.ID,
		&auth.OAuthAppID,
		&userIDRaw,
		&scopesRaw,
		&auth.CreatedAt,
		&auth.UpdatedAt,
	); err != nil {
		return nil, err
	}

	userID, err := parseTokenID(userIDRaw)
	if err != nil {
		return nil, err
	}
	auth.UserID = userID

	scopes, err := unmarshalScopes(driver, scopesRaw)
	if err != nil {
		return nil, err
	}
	auth.GrantedScopes = scopes

	return &auth, nil
}

func scanOAuthAuthorizations(rows *sqlx.Rows, driver string) ([]*domain.OAuthAuthorization, error) {
	auths := make([]*domain.OAuthAuthorization, 0)
	for rows.Next() {
		auth, err := scanOAuthAuthorization(rows, driver)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		auths = append(auths, auth)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return auths, nil
}
