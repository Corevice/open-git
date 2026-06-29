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

type sqlxOAuthAuthorizationCodeRepository struct {
	*sqlx.DB
}

func NewOAuthAuthorizationCodeRepository(db *sqlx.DB) *sqlxOAuthAuthorizationCodeRepository {
	return &sqlxOAuthAuthorizationCodeRepository{DB: db}
}

var _ repo.IOAuthAuthorizationCodeRepository = (*sqlxOAuthAuthorizationCodeRepository)(nil)

const oauthAuthorizationCodeSelectColumns = `
	id, code_hash, oauth_app_id, user_id, redirect_uri, scopes,
	expires_at, consumed_at, created_at
`

func (r *sqlxOAuthAuthorizationCodeRepository) Create(ctx context.Context, code *domain.OAuthAuthorizationCode) error {
	if code.ID == "" {
		code.ID = uuid.New().String()
	}
	if code.CreatedAt.IsZero() {
		code.CreatedAt = time.Now().UTC()
	}

	scopes, err := marshalScopes(r.DriverName(), code.Scopes)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO oauth_authorization_codes (
			id, code_hash, oauth_app_id, user_id, redirect_uri, scopes,
			expires_at, consumed_at, created_at
		)
		VALUES (
			:id, :code_hash, :oauth_app_id, :user_id, :redirect_uri, :scopes,
			:expires_at, :consumed_at, :created_at
		)
	`

	_, err = r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":           code.ID,
		"code_hash":    code.CodeHash,
		"oauth_app_id": code.OAuthAppID,
		"user_id":      formatTokenID(code.UserID),
		"redirect_uri": code.RedirectURI,
		"scopes":       scopes,
		"expires_at":   code.ExpiresAt,
		"consumed_at":  code.ConsumedAt,
		"created_at":   code.CreatedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxOAuthAuthorizationCodeRepository) ConsumeByCodeHash(ctx context.Context, codeHash string) (*domain.OAuthAuthorizationCode, error) {
	now := time.Now().UTC()

	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer tx.Rollback()

	updateQuery := `
		UPDATE oauth_authorization_codes
		SET consumed_at = ?
		WHERE code_hash = ? AND consumed_at IS NULL AND expires_at > ?
	`
	updateQuery = tx.Rebind(updateQuery)

	result, err := tx.ExecContext(ctx, updateQuery, now, codeHash, now)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	if rows == 0 {
		return nil, nil
	}

	selectQuery := `SELECT ` + oauthAuthorizationCodeSelectColumns + ` FROM oauth_authorization_codes WHERE code_hash = ?`
	selectQuery = tx.Rebind(selectQuery)

	row := tx.QueryRowxContext(ctx, selectQuery, codeHash)
	code, err := scanOAuthAuthorizationCode(row, r.DriverName())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return code, nil
}

type oauthAuthorizationCodeRow interface {
	Scan(dest ...any) error
}

func scanOAuthAuthorizationCode(row oauthAuthorizationCodeRow, driver string) (*domain.OAuthAuthorizationCode, error) {
	var (
		code        domain.OAuthAuthorizationCode
		userIDRaw   string
		scopesRaw   any
		consumedAt  sql.NullTime
	)

	if err := row.Scan(
		&code.ID,
		&code.CodeHash,
		&code.OAuthAppID,
		&userIDRaw,
		&code.RedirectURI,
		&scopesRaw,
		&code.ExpiresAt,
		&consumedAt,
		&code.CreatedAt,
	); err != nil {
		return nil, err
	}

	userID, err := parseTokenID(userIDRaw)
	if err != nil {
		return nil, err
	}
	code.UserID = userID

	scopes, err := unmarshalScopes(driver, scopesRaw)
	if err != nil {
		return nil, err
	}
	code.Scopes = scopes

	if consumedAt.Valid {
		t := consumedAt.Time
		code.ConsumedAt = &t
	}

	return &code, nil
}
