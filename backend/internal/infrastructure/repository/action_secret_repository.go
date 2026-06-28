package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/crypto"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxActionSecretRepository struct {
	*sqlx.DB
	enc *crypto.ActionSecretEncryptor
}

var _ domainrepo.IActionSecretRepository = (*sqlxActionSecretRepository)(nil)

func NewActionSecretRepository(db *sqlx.DB, enc *crypto.ActionSecretEncryptor) *sqlxActionSecretRepository {
	return &sqlxActionSecretRepository{DB: db, enc: enc}
}

const actionSecretListColumns = `
	id, organization_id, repository_id, name, key_id, visibility, created_at, updated_at
`

func (r *sqlxActionSecretRepository) Upsert(ctx context.Context, secret *entity.ActionSecret) (bool, error) {
	repoID := actionSecretRepoIDPtr(secret)
	_, err := r.GetByName(ctx, secret.OrganizationID, repoID, secret.Name)
	exists := err == nil
	if err != nil && !errors.Is(err, apperror.ErrNotFound) {
		return false, dbErrors.MapDBError(err)
	}

	encrypted, err := r.enc.Encrypt([]byte(secret.EncryptedValue))
	if err != nil {
		return false, err
	}

	now := time.Now().UTC()
	if secret.CreatedAt.IsZero() {
		secret.CreatedAt = now
	}
	secret.UpdatedAt = now

	if exists {
		query := `
			UPDATE action_secrets
			SET encrypted_value = ?, key_id = ?, visibility = ?, updated_at = ?
			WHERE organization_id = ?
				AND name = ?
				AND ((repository_id = ?) OR (repository_id IS NULL AND ? IS NULL))
		`
		query = r.DB.Rebind(query)
		result, err := r.DB.ExecContext(ctx, query,
			encrypted,
			actionSecretKeyID(secret),
			actionSecretVisibility(secret),
			secret.UpdatedAt,
			secret.OrganizationID,
			secret.Name,
			repoIDParam(repoID),
			repoIDParam(repoID),
		)
		if err != nil {
			return false, dbErrors.MapDBError(err)
		}
		if rows, err := result.RowsAffected(); err != nil {
			return false, dbErrors.MapDBError(err)
		} else if rows == 0 {
			return false, apperror.ErrNotFound
		}
		return false, nil
	}

	if secret.ID == uuid.Nil {
		secret.ID = uuid.New()
	}

	var query string
	if r.DriverName() == "postgres" {
		query = `
			INSERT INTO action_secrets (
				id, organization_id, repository_id, name, encrypted_value, key_id, visibility, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9
			)
			ON CONFLICT (organization_id, repository_id, name) DO UPDATE SET
				encrypted_value = EXCLUDED.encrypted_value,
				key_id = EXCLUDED.key_id,
				visibility = EXCLUDED.visibility,
				updated_at = EXCLUDED.updated_at
		`
	} else {
		query = `
			INSERT INTO action_secrets (
				id, organization_id, repository_id, name, encrypted_value, key_id, visibility, created_at, updated_at
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?
			)
		`
	}

	args := []any{
		secret.ID,
		secret.OrganizationID,
		repoIDParam(repoID),
		secret.Name,
		encrypted,
		actionSecretKeyID(secret),
		actionSecretVisibility(secret),
		secret.CreatedAt,
		secret.UpdatedAt,
	}

	if r.DriverName() == "postgres" {
		_, err = r.DB.ExecContext(ctx, query, args...)
	} else {
		query = r.DB.Rebind(query)
		_, err = r.DB.ExecContext(ctx, query, args...)
	}
	if err != nil {
		return false, dbErrors.MapDBError(err)
	}
	return true, nil
}

func (r *sqlxActionSecretRepository) GetByName(ctx context.Context, orgID uuid.UUID, repoID *uuid.UUID, name string) (*entity.ActionSecret, error) {
	query := `
		SELECT id, organization_id, repository_id, name, encrypted_value, key_id, visibility, created_at, updated_at
		FROM action_secrets
		WHERE organization_id = ?
			AND ((repository_id = ?) OR (repository_id IS NULL AND ? IS NULL))
			AND name = ?
	`
	query = r.DB.Rebind(query)

	secret, err := r.scanActionSecret(r.DB.QueryRowxContext(ctx, query, orgID, repoIDParam(repoID), repoIDParam(repoID), name), false)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return secret, nil
}

func (r *sqlxActionSecretRepository) ListByRepo(ctx context.Context, orgID, repoID uuid.UUID) ([]*entity.ActionSecret, error) {
	query := `
		SELECT ` + actionSecretListColumns + `
		FROM action_secrets
		WHERE organization_id = ? AND repository_id = ?
		ORDER BY name ASC
	`
	query = r.DB.Rebind(query)

	return r.queryActionSecretList(ctx, query, orgID, repoID)
}

func (r *sqlxActionSecretRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entity.ActionSecret, error) {
	query := `
		SELECT ` + actionSecretListColumns + `
		FROM action_secrets
		WHERE organization_id = ? AND repository_id IS NULL
		ORDER BY name ASC
	`
	query = r.DB.Rebind(query)

	return r.queryActionSecretList(ctx, query, orgID)
}

func (r *sqlxActionSecretRepository) Delete(ctx context.Context, orgID uuid.UUID, repoID *uuid.UUID, name string) error {
	query := `
		DELETE FROM action_secrets
		WHERE organization_id = ?
			AND name = ?
			AND ((repository_id = ?) OR (repository_id IS NULL AND ? IS NULL))
	`
	query = r.DB.Rebind(query)

	result, err := r.DB.ExecContext(ctx, query, orgID, name, repoIDParam(repoID), repoIDParam(repoID))
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	if rows == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *sqlxActionSecretRepository) ListForWorkflow(ctx context.Context, orgID, repoID uuid.UUID) ([]*entity.ActionSecret, error) {
	query := `
		SELECT id, organization_id, repository_id, name, encrypted_value, key_id, visibility, created_at, updated_at
		FROM action_secrets
		WHERE organization_id = ? AND repository_id = ?
		UNION ALL
		SELECT s.id, s.organization_id, s.repository_id, s.name, s.encrypted_value, s.key_id, s.visibility, s.created_at, s.updated_at
		FROM action_secrets s
		WHERE s.organization_id = ?
			AND s.repository_id IS NULL
			AND (
				s.visibility = 'all'
				OR (
					s.visibility = 'private'
					AND EXISTS (
						SELECT 1
						FROM repositories rep
						WHERE rep.id = ?
							AND rep.organization_id = s.organization_id
							AND rep.visibility = 'private'
					)
				)
				OR (
					s.visibility = 'selected'
					AND EXISTS (
						SELECT 1
						FROM action_secret_repositories sr
						WHERE sr.secret_id = s.id AND sr.repository_id = ?
					)
				)
			)
	`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, orgID, repoID, orgID, repoID, repoID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	repoSecrets := make([]*entity.ActionSecret, 0)
	orgSecrets := make([]*entity.ActionSecret, 0)
	for rows.Next() {
		secret, err := r.scanActionSecret(rows, true)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		if secret.RepositoryID == uuid.Nil {
			orgSecrets = append(orgSecrets, secret)
			continue
		}
		repoSecrets = append(repoSecrets, secret)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}

	byName := make(map[string]*entity.ActionSecret, len(repoSecrets)+len(orgSecrets))
	for _, secret := range repoSecrets {
		byName[secret.Name] = secret
	}
	for _, secret := range orgSecrets {
		if _, exists := byName[secret.Name]; !exists {
			byName[secret.Name] = secret
		}
	}

	merged := make([]*entity.ActionSecret, 0, len(byName))
	for _, secret := range byName {
		merged = append(merged, secret)
	}
	return merged, nil
}

func (r *sqlxActionSecretRepository) SetSelectedRepositories(ctx context.Context, secretID uuid.UUID, repoIDs []uuid.UUID) error {
	deleteQuery := `DELETE FROM action_secret_repositories WHERE secret_id = ?`
	deleteQuery = r.DB.Rebind(deleteQuery)
	if _, err := r.DB.ExecContext(ctx, deleteQuery, secretID); err != nil {
		return dbErrors.MapDBError(err)
	}

	if len(repoIDs) == 0 {
		return nil
	}

	insertQuery := `INSERT INTO action_secret_repositories (secret_id, repository_id) VALUES (?, ?)`
	insertQuery = r.DB.Rebind(insertQuery)
	for _, repoID := range repoIDs {
		if _, err := r.DB.ExecContext(ctx, insertQuery, secretID, repoID); err != nil {
			return dbErrors.MapDBError(err)
		}
	}
	return nil
}

func (r *sqlxActionSecretRepository) GetSelectedRepositories(ctx context.Context, secretID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT repository_id FROM action_secret_repositories WHERE secret_id = ?`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, secretID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	repoIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var repoID uuid.UUID
		if err := rows.Scan(&repoID); err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		repoIDs = append(repoIDs, repoID)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return repoIDs, nil
}

func (r *sqlxActionSecretRepository) queryActionSecretList(ctx context.Context, query string, args ...any) ([]*entity.ActionSecret, error) {
	rows, err := r.DB.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	secrets := make([]*entity.ActionSecret, 0)
	for rows.Next() {
		secret, err := r.scanActionSecretListRow(rows)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		secrets = append(secrets, secret)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return secrets, nil
}

type actionSecretScanner interface {
	Scan(dest ...any) error
}

func (r *sqlxActionSecretRepository) scanActionSecretListRow(scanner actionSecretScanner) (*entity.ActionSecret, error) {
	var (
		secret       entity.ActionSecret
		repositoryID sql.NullString
		keyID        sql.NullString
		visibility   sql.NullString
	)

	if err := scanner.Scan(
		&secret.ID,
		&secret.OrganizationID,
		&repositoryID,
		&secret.Name,
		&keyID,
		&visibility,
		&secret.CreatedAt,
		&secret.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if repositoryID.Valid {
		parsed, err := uuid.Parse(repositoryID.String)
		if err != nil {
			return nil, err
		}
		secret.RepositoryID = parsed
	}

	return &secret, nil
}

func (r *sqlxActionSecretRepository) scanActionSecret(scanner actionSecretScanner, decrypt bool) (*entity.ActionSecret, error) {
	var (
		secret         entity.ActionSecret
		repositoryID   sql.NullString
		encryptedValue []byte
		keyID          sql.NullString
		visibility     sql.NullString
	)

	if err := scanner.Scan(
		&secret.ID,
		&secret.OrganizationID,
		&repositoryID,
		&secret.Name,
		&encryptedValue,
		&keyID,
		&visibility,
		&secret.CreatedAt,
		&secret.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if repositoryID.Valid {
		parsed, err := uuid.Parse(repositoryID.String)
		if err != nil {
			return nil, err
		}
		secret.RepositoryID = parsed
	}

	if len(encryptedValue) > 0 {
		if decrypt {
			decrypted, err := r.enc.Decrypt(encryptedValue)
			if err != nil {
				return nil, err
			}
			secret.EncryptedValue = string(decrypted)
		} else {
			secret.EncryptedValue = string(encryptedValue)
		}
	}

	return &secret, nil
}

func actionSecretRepoIDPtr(secret *entity.ActionSecret) *uuid.UUID {
	if secret.RepositoryID == uuid.Nil {
		return nil
	}
	repoID := secret.RepositoryID
	return &repoID
}

func repoIDParam(repoID *uuid.UUID) any {
	if repoID == nil {
		return nil
	}
	return *repoID
}

func actionSecretKeyID(secret *entity.ActionSecret) string {
	return ""
}

func actionSecretVisibility(secret *entity.ActionSecret) string {
	if secret.RepositoryID == uuid.Nil {
		return "all"
	}
	return ""
}
