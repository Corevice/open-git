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

type sqlxArtifactRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IArtifactRepository = (*sqlxArtifactRepository)(nil)

func NewArtifactRepository(db *sqlx.DB) domainrepo.IArtifactRepository {
	return &sqlxArtifactRepository{db: db}
}

func (r *sqlxArtifactRepository) Create(ctx context.Context, artifact *entity.Artifact) error {
	if artifact.ID == uuid.Nil {
		artifact.ID = uuid.New()
	}
	now := time.Now().UTC()
	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = now
	}
	if artifact.ExpiresAt.IsZero() {
		artifact.ExpiresAt = artifact.CreatedAt.Add(90 * 24 * time.Hour)
	}

	status := entity.ArtifactStatusPending

	retentionDays := int(artifact.ExpiresAt.Sub(artifact.CreatedAt).Hours() / 24)
	if retentionDays < 1 {
		retentionDays = 90
	}

	var repositoryID uuid.UUID
	const lookupQuery = `SELECT repository_id FROM workflow_runs WHERE id = ?`
	q := r.db.Rebind(lookupQuery)
	if err := r.db.GetContext(ctx, &repositoryID, q, artifact.RunID); err != nil {
		return dbErrors.MapDBError(err)
	}

	const query = `
		INSERT INTO artifacts (
			id, organization_id, repository_id, workflow_run_id, name, storage_key,
			size_in_bytes, status, retention_days, created_at, expires_at
		) VALUES (
			:id, :organization_id, :repository_id, :workflow_run_id, :name, :storage_key,
			:size_in_bytes, :status, :retention_days, :created_at, :expires_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":              artifact.ID,
		"organization_id": artifact.OrganizationID,
		"repository_id":   repositoryID,
		"workflow_run_id": artifact.RunID,
		"name":            artifact.Name,
		"storage_key":     artifact.StorageKey,
		"size_in_bytes":   artifact.SizeBytes,
		"status":          status,
		"retention_days":  retentionDays,
		"created_at":      artifact.CreatedAt,
		"expires_at":      artifact.ExpiresAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxArtifactRepository) GetByID(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*entity.Artifact, error) {
	const query = `
		SELECT id, organization_id, repository_id, workflow_run_id, name, storage_key,
			size_in_bytes, status, retention_days, created_at, expires_at, deleted_at
		FROM artifacts
		WHERE id = ? AND organization_id = ? AND deleted_at IS NULL
	`
	q := r.db.Rebind(query)
	row := r.db.QueryRowxContext(ctx, q, id, orgID)

	artifact, err := scanArtifactRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return artifact, nil
}

func (r *sqlxArtifactRepository) ListByRun(ctx context.Context, runID uuid.UUID, orgID uuid.UUID) ([]*entity.Artifact, error) {
	const query = `
		SELECT id, organization_id, repository_id, workflow_run_id, name, storage_key,
			size_in_bytes, status, retention_days, created_at, expires_at, deleted_at
		FROM artifacts
		WHERE workflow_run_id = ? AND organization_id = ? AND deleted_at IS NULL
		ORDER BY created_at
	`
	q := r.db.Rebind(query)
	rows, err := r.db.QueryxContext(ctx, q, runID, orgID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	artifacts := make([]*entity.Artifact, 0)
	for rows.Next() {
		artifact, err := scanArtifactRow(rows)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		artifacts = append(artifacts, artifact)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return artifacts, nil
}

func (r *sqlxArtifactRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.ArtifactStatus, sizeInBytes int64) error {
	const query = `
		UPDATE artifacts
		SET status = ?, size_in_bytes = ?
		WHERE id = ?
	`
	q := r.db.Rebind(query)
	_, err := r.db.ExecContext(ctx, q, status, sizeInBytes, id)
	return dbErrors.MapDBError(err)
}

func (r *sqlxArtifactRepository) SoftDelete(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error {
	const query = `
		UPDATE artifacts
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = ? AND organization_id = ?
	`
	q := r.db.Rebind(query)
	_, err := r.db.ExecContext(ctx, q, id, orgID)
	return dbErrors.MapDBError(err)
}

func (r *sqlxArtifactRepository) ListExpired(ctx context.Context, limit int) ([]*entity.Artifact, error) {
	if limit <= 0 {
		limit = 100
	}

	const query = `
		SELECT id, organization_id, repository_id, workflow_run_id, name, storage_key,
			size_in_bytes, status, retention_days, created_at, expires_at, deleted_at
		FROM artifacts
		WHERE expires_at < CURRENT_TIMESTAMP
			AND status = 'completed'
			AND deleted_at IS NULL
		LIMIT ?
	`
	q := r.db.Rebind(query)
	rows, err := r.db.QueryxContext(ctx, q, limit)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	artifacts := make([]*entity.Artifact, 0)
	for rows.Next() {
		artifact, err := scanArtifactRow(rows)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		artifacts = append(artifacts, artifact)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return artifacts, nil
}

func (r *sqlxArtifactRepository) DeleteByRunID(ctx context.Context, runID uuid.UUID) error {
	const query = `
		UPDATE artifacts
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE workflow_run_id = ? AND deleted_at IS NULL
	`
	q := r.db.Rebind(query)
	_, err := r.db.ExecContext(ctx, q, runID)
	return dbErrors.MapDBError(err)
}

type artifactScanner interface {
	Scan(dest ...any) error
}

func scanArtifactRow(scanner artifactScanner) (*entity.Artifact, error) {
	var (
		artifact  entity.Artifact
		repoID    uuid.UUID
		status    string
		retention int
		deletedAt sql.NullTime
	)

	if err := scanner.Scan(
		&artifact.ID,
		&artifact.OrganizationID,
		&repoID,
		&artifact.RunID,
		&artifact.Name,
		&artifact.StorageKey,
		&artifact.SizeBytes,
		&status,
		&retention,
		&artifact.CreatedAt,
		&artifact.ExpiresAt,
		&deletedAt,
	); err != nil {
		return nil, err
	}

	_ = repoID
	_ = status
	_ = retention
	return &artifact, nil
}
