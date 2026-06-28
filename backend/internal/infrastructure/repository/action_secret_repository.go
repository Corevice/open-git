package repository

import (
	"context"
	"database/sql"
	"errors"
	"sort"
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

	encrypted, err := r.enc.Encrypt([]byte(secret.EncryptedValue))
	if err != nil {
		return false, err
	}

	now := time.Now().UTC()
	if secret.CreatedAt.IsZero() {
		secret.CreatedAt = now
	}
	secret.UpdatedAt = now
	if secret.ID == uuid.Nil {
		secret.ID = uuid.New()
	}

	keyID := secret.KeyID
	visibility := actionSecretVisibility(secret)

	if r.DriverName() == "postgres" && repoID == nil {
		return r.upsertPostgresOrgSecret(ctx, secret, encrypted, keyID, visibility)
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
			RETURNING (created_at = updated_at)
		`
	} else {
		query = `
			INSERT INTO action_secrets (
				id, organization_id, repository_id, name, encrypted_value, key_id, visibility, created_at, updated_at
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?
			)
			ON CONFLICT(organization_id, IFNULL(repository_id, ''), name) DO UPDATE SET
				encrypted_value = excluded.encrypted_value,
				key_id = excluded.key_id,
				visibility = excluded.visibility,
				updated_at = excluded.updated_at
			RETURNING (created_at = updated_at)
		`
	}

	args := []any{
		secret.ID,
		secret.OrganizationID,
		repoIDParam(repoID),
		secret.Name,
		encrypted,
		keyID,
		visibility,
		secret.CreatedAt,
		secret.UpdatedAt,
	}

	if r.DriverName() != "postgres" {
		query = r.DB.Rebind(query)
	}
	var created bool
	if err := r.DB.QueryRowxContext(ctx, query, args...).Scan(&created); err != nil {
		return false, dbErrors.MapDBError(err)
	}
	return created, nil
}

func (r *sqlxActionSecretRepository) upsertPostgresOrgSecret(
	ctx context.Context,
	secret *entity.ActionSecret,
	encrypted []byte,
	keyID, visibility string,
) (bool, error) {
	updateQuery := `
		UPDATE action_secrets
		SET encrypted_value = $1, key_id = $2, visibility = $3, updated_at = $4
		WHERE organization_id = $5 AND repository_id IS NULL AND name = $6
	`
	result, err := r.DB.ExecContext(ctx, updateQuery, encrypted, keyID, visibility, secret.UpdatedAt, secret.OrganizationID, secret.Name)
	if err != nil {
		return false, dbErrors.MapDBError(err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, dbErrors.MapDBError(err)
	}
	if rows > 0 {
		return false, nil
	}

	insertQuery := `
		INSERT INTO action_secrets (
			id, organization_id, repository_id, name, encrypted_value, key_id, visibility, created_at, updated_at
		) VALUES (
			$1, $2, NULL, $3, $4, $5, $6, $7, $8
		)
	`
	if _, err := r.DB.ExecContext(ctx, insertQuery, secret.ID, secret.OrganizationID, secret.Name, encrypted, keyID, visibility, secret.CreatedAt, secret.UpdatedAt); err != nil {
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

	secret, err := r.scanActionSecret(r.DB.QueryRowxContext(ctx, query, orgID, repoIDParam(repoID), repoIDParam(repoID), name), true)
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
	if err := r.validateRepositoryOrganizationIDs(ctx, orgID, []uuid.UUID{repoID}); err != nil {
		return nil, err
	}

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
						INNER JOIN repositories rep ON rep.id = sr.repository_id
						WHERE sr.secret_id = s.id
							AND sr.repository_id = ?
							AND rep.organization_id = s.organization_id
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

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	merged := make([]*entity.ActionSecret, 0, len(byName))
	for _, name := range names {
		merged = append(merged, byName[name])
	}
	return merged, nil
}

func (r *sqlxActionSecretRepository) SetSelectedRepositories(ctx context.Context, orgID, secretID uuid.UUID, repoIDs []uuid.UUID) error {
	secretOrgID, visibility, err := r.getActionSecretMeta(ctx, secretID)
	if err != nil {
		return err
	}
	if secretOrgID != orgID {
		return apperror.ErrNotFound
	}
	if visibility != "selected" {
		return apperror.ErrNotFound
	}

	uniqueRepoIDs, err := uniqueUUIDs(repoIDs)
	if err != nil {
		return err
	}

	if err := r.validateRepositoryOrganizationIDs(ctx, orgID, uniqueRepoIDs); err != nil {
		return err
	}

	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	defer func() { _ = tx.Rollback() }()

	deleteQuery := `DELETE FROM action_secret_repositories WHERE secret_id = ?`
	deleteQuery = tx.Rebind(deleteQuery)
	if _, err := tx.ExecContext(ctx, deleteQuery, secretID); err != nil {
		return dbErrors.MapDBError(err)
	}

	if len(uniqueRepoIDs) > 0 {
		insertQuery := tx.Rebind(`INSERT INTO action_secret_repositories (secret_id, repository_id) VALUES (?, ?)`)
		for _, repoID := range uniqueRepoIDs {
			if _, err := tx.ExecContext(ctx, insertQuery, secretID, repoID); err != nil {
				return dbErrors.MapDBError(err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return dbErrors.MapDBError(err)
	}
	return nil
}

func (r *sqlxActionSecretRepository) GetSelectedRepositories(ctx context.Context, orgID, secretID uuid.UUID) ([]uuid.UUID, error) {
	secretOrgID, _, err := r.getActionSecretMeta(ctx, secretID)
	if err != nil {
		return nil, err
	}
	if secretOrgID != orgID {
		return nil, apperror.ErrNotFound
	}

	query := `SELECT repository_id FROM action_secret_repositories WHERE secret_id = ? ORDER BY repository_id ASC`
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

func (r *sqlxActionSecretRepository) getActionSecretMeta(ctx context.Context, secretID uuid.UUID) (uuid.UUID, string, error) {
	query := `SELECT organization_id, visibility FROM action_secrets WHERE id = ?`
	query = r.DB.Rebind(query)

	var orgID uuid.UUID
	var visibility sql.NullString
	err := r.DB.QueryRowxContext(ctx, query, secretID).Scan(&orgID, &visibility)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, "", apperror.ErrNotFound
	}
	if err != nil {
		return uuid.Nil, "", dbErrors.MapDBError(err)
	}
	if visibility.Valid {
		return orgID, visibility.String, nil
	}
	return orgID, "", nil
}

func uniqueUUIDs(ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return ids, nil
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	unique := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			return nil, apperror.ErrNotFound
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique, nil
}

func (r *sqlxActionSecretRepository) validateRepositoryOrganizationIDs(ctx context.Context, orgID uuid.UUID, repoIDs []uuid.UUID) error {
	if len(repoIDs) == 0 {
		return nil
	}

	query := `SELECT COUNT(DISTINCT id) FROM repositories WHERE organization_id = ? AND id IN (?)`
	query, args, err := sqlx.In(query, orgID, repoIDs)
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	query = r.DB.Rebind(query)

	var count int
	if err := r.DB.QueryRowxContext(ctx, query, args...).Scan(&count); err != nil {
		return dbErrors.MapDBError(err)
	}
	if count != len(repoIDs) {
		return apperror.ErrNotFound
	}
	return nil
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
	if keyID.Valid {
		secret.KeyID = keyID.String
	}
	if visibility.Valid {
		secret.Visibility = visibility.String
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
	if keyID.Valid {
		secret.KeyID = keyID.String
	}
	if visibility.Valid {
		secret.Visibility = visibility.String
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

func actionSecretVisibility(secret *entity.ActionSecret) string {
	if secret.RepositoryID == uuid.Nil {
		if secret.Visibility != "" {
			return secret.Visibility
		}
		return "all"
	}
	if secret.Visibility != "" {
		return secret.Visibility
	}
	return ""
}
