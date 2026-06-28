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

type sqlxOAuthAppRepository struct {
	*sqlx.DB
}

func NewOAuthAppRepository(db *sqlx.DB) *sqlxOAuthAppRepository {
	return &sqlxOAuthAppRepository{DB: db}
}

var _ repo.IOAuthAppRepository = (*sqlxOAuthAppRepository)(nil)

const oauthAppSelectColumns = `
	id, client_id, client_secret_hash, redirect_uris, name, homepage_url,
	owner_type, owner_user_id, organization_id, created_at, updated_at
`

func (r *sqlxOAuthAppRepository) Create(ctx context.Context, app *domain.OAuthApp) error {
	if app.ID == "" {
		app.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if app.UpdatedAt.IsZero() {
		app.UpdatedAt = now
	}

	redirectURIs, err := marshalScopes(r.DriverName(), app.RedirectURIs)
	if err != nil {
		return err
	}

	ownerID := formatTokenID(app.OwnerUserID)
	ownerUserID := sql.NullString{}
	if app.OwnerUserID != 0 {
		ownerUserID = sql.NullString{String: ownerID, Valid: true}
	}
	organizationID := sql.NullString{}
	if app.OrganizationID != 0 {
		organizationID = sql.NullString{String: formatTokenID(app.OrganizationID), Valid: true}
	}

	const query = `
		INSERT INTO oauth_apps (
			id, owner_id, client_id, client_secret_hash, redirect_uris,
			name, homepage_url, owner_type, owner_user_id, organization_id,
			created_at, updated_at
		)
		VALUES (
			:id, :owner_id, :client_id, :client_secret_hash, :redirect_uris,
			:name, :homepage_url, :owner_type, :owner_user_id, :organization_id,
			:created_at, :updated_at
		)
	`

	_, err = r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":                 app.ID,
		"owner_id":           ownerID,
		"client_id":          app.ClientID,
		"client_secret_hash": app.ClientSecretHash,
		"redirect_uris":      redirectURIs,
		"name":               app.Name,
		"homepage_url":       app.HomepageURL,
		"owner_type":         app.OwnerType,
		"owner_user_id":      ownerUserID,
		"organization_id":    organizationID,
		"created_at":         now,
		"updated_at":         app.UpdatedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxOAuthAppRepository) GetByID(ctx context.Context, id string) (*domain.OAuthApp, error) {
	query := `SELECT ` + oauthAppSelectColumns + ` FROM oauth_apps WHERE id = ?`
	return r.getOne(ctx, query, id)
}

func (r *sqlxOAuthAppRepository) GetByClientID(ctx context.Context, clientID string) (*domain.OAuthApp, error) {
	query := `SELECT ` + oauthAppSelectColumns + ` FROM oauth_apps WHERE client_id = ?`
	return r.getOne(ctx, query, clientID)
}

func (r *sqlxOAuthAppRepository) ListByOwnerUser(ctx context.Context, userID int64, page, perPage int) ([]*domain.OAuthApp, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	query := `
		SELECT ` + oauthAppSelectColumns + `
		FROM oauth_apps
		WHERE owner_user_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, formatTokenID(userID), perPage, offset)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	return scanOAuthApps(rows, r.DriverName())
}

func (r *sqlxOAuthAppRepository) ListByOrganization(ctx context.Context, orgID int64, page, perPage int) ([]*domain.OAuthApp, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	query := `
		SELECT ` + oauthAppSelectColumns + `
		FROM oauth_apps
		WHERE organization_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, formatTokenID(orgID), perPage, offset)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	return scanOAuthApps(rows, r.DriverName())
}

func (r *sqlxOAuthAppRepository) Update(ctx context.Context, app *domain.OAuthApp) error {
	app.UpdatedAt = time.Now().UTC()

	redirectURIs, err := marshalScopes(r.DriverName(), app.RedirectURIs)
	if err != nil {
		return err
	}

	ownerUserID := sql.NullString{}
	if app.OwnerUserID != 0 {
		ownerUserID = sql.NullString{String: formatTokenID(app.OwnerUserID), Valid: true}
	}
	organizationID := sql.NullString{}
	if app.OrganizationID != 0 {
		organizationID = sql.NullString{String: formatTokenID(app.OrganizationID), Valid: true}
	}

	const query = `
		UPDATE oauth_apps
		SET client_id = :client_id,
			client_secret_hash = :client_secret_hash,
			redirect_uris = :redirect_uris,
			name = :name,
			homepage_url = :homepage_url,
			owner_type = :owner_type,
			owner_user_id = :owner_user_id,
			organization_id = :organization_id,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":                 app.ID,
		"client_id":          app.ClientID,
		"client_secret_hash": app.ClientSecretHash,
		"redirect_uris":      redirectURIs,
		"name":               app.Name,
		"homepage_url":       app.HomepageURL,
		"owner_type":         app.OwnerType,
		"owner_user_id":      ownerUserID,
		"organization_id":    organizationID,
		"updated_at":         app.UpdatedAt,
	})
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

func (r *sqlxOAuthAppRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM oauth_apps WHERE id = ?`
	query = r.DB.Rebind(query)

	result, err := r.DB.ExecContext(ctx, query, id)
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

func (r *sqlxOAuthAppRepository) UpdateSecretHash(ctx context.Context, id, hash string) error {
	now := time.Now().UTC()
	query := `
		UPDATE oauth_apps
		SET client_secret_hash = ?, updated_at = ?
		WHERE id = ?
	`
	query = r.DB.Rebind(query)

	result, err := r.DB.ExecContext(ctx, query, hash, now, id)
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

func (r *sqlxOAuthAppRepository) getOne(ctx context.Context, query string, arg any) (*domain.OAuthApp, error) {
	query = r.DB.Rebind(query)
	row := r.DB.QueryRowxContext(ctx, query, arg)

	app, err := scanOAuthApp(row, r.DriverName())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return app, nil
}

type oauthAppRow interface {
	Scan(dest ...any) error
}

func scanOAuthApp(row oauthAppRow, driver string) (*domain.OAuthApp, error) {
	var (
		app              domain.OAuthApp
		redirectURIsRaw  any
		ownerUserIDRaw   sql.NullString
		organizationIDRaw sql.NullString
		createdAt        time.Time
		updatedAt        sql.NullTime
	)

	if err := row.Scan(
		&app.ID,
		&app.ClientID,
		&app.ClientSecretHash,
		&redirectURIsRaw,
		&app.Name,
		&app.HomepageURL,
		&app.OwnerType,
		&ownerUserIDRaw,
		&organizationIDRaw,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	redirectURIs, err := unmarshalScopes(driver, redirectURIsRaw)
	if err != nil {
		return nil, err
	}
	app.RedirectURIs = redirectURIs

	if ownerUserIDRaw.Valid {
		app.OwnerUserID, err = parseTokenID(ownerUserIDRaw.String)
		if err != nil {
			return nil, err
		}
	}
	if organizationIDRaw.Valid {
		app.OrganizationID, err = parseTokenID(organizationIDRaw.String)
		if err != nil {
			return nil, err
		}
	}
	if updatedAt.Valid {
		app.UpdatedAt = updatedAt.Time
	}

	return &app, nil
}

func scanOAuthApps(rows *sqlx.Rows, driver string) ([]*domain.OAuthApp, error) {
	apps := make([]*domain.OAuthApp, 0)
	for rows.Next() {
		app, err := scanOAuthApp(rows, driver)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return apps, nil
}
