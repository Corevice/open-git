package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxRunnerRegistrationTokenRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IRunnerRegistrationTokenRepository = (*sqlxRunnerRegistrationTokenRepository)(nil)

func NewRunnerRegistrationTokenRepository(db *sqlx.DB) domainrepo.IRunnerRegistrationTokenRepository {
	return &sqlxRunnerRegistrationTokenRepository{db: db}
}

func (r *sqlxRunnerRegistrationTokenRepository) Create(ctx context.Context, token *entity.RunnerRegistrationToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}

	var usedAt sql.NullTime
	if token.UsedAt != nil {
		usedAt = sql.NullTime{Time: *token.UsedAt, Valid: true}
	}

	const query = `
		INSERT INTO runner_registration_tokens (id, organization_id, token_hash, expires_at, used_at)
		VALUES (:id, :organization_id, :token_hash, :expires_at, :used_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":              token.ID,
		"organization_id": token.OrganizationID,
		"token_hash":      token.TokenHash,
		"expires_at":      token.ExpiresAt,
		"used_at":         usedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxRunnerRegistrationTokenRepository) GetByTokenHash(ctx context.Context, hash string) (*entity.RunnerRegistrationToken, error) {
	now := time.Now().UTC()
	const query = `
		SELECT id, organization_id, token_hash, expires_at, used_at
		FROM runner_registration_tokens
		WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?
	`
	q := r.db.Rebind(query)
	row := r.db.QueryRowxContext(ctx, q, hash, now)

	var (
		token  entity.RunnerRegistrationToken
		usedAt sql.NullTime
	)
	err := row.Scan(&token.ID, &token.OrganizationID, &token.TokenHash, &token.ExpiresAt, &usedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	if usedAt.Valid {
		token.UsedAt = &usedAt.Time
	}
	return &token, nil
}

func (r *sqlxRunnerRegistrationTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error {
	const query = `
		UPDATE runner_registration_tokens
		SET used_at = ?
		WHERE id = ? AND used_at IS NULL
	`
	q := r.db.Rebind(query)
	result, err := r.db.ExecContext(ctx, q, usedAt, id)
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
