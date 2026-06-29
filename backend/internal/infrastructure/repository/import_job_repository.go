package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlImportJobRepository struct {
	*sqlx.DB
}

var (
	_ domainrepo.IImportJobRepository              = (*sqlImportJobRepository)(nil)
	_ domainrepo.IImportUserMappingRepository      = (*sqlImportJobRepository)(nil)
	_ domainrepo.IImportPhaseCheckpointRepository  = (*sqlImportJobRepository)(nil)
)

func NewImportJobRepository(db *sqlx.DB) *sqlImportJobRepository {
	return &sqlImportJobRepository{DB: db}
}

const importJobSelectColumns = `
	id, organization_id, created_by, source_url, target_repository_id, target_name,
	include, status, phase, progress, token_secret_ref, error, created_at, updated_at
`

func (r *sqlImportJobRepository) Create(ctx context.Context, job *entity.ImportJob) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	now := time.Now().UTC()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = now
	}
	if job.Status == "" {
		job.Status = entity.ImportJobStatusQueued
	}
	if job.Phase == "" {
		job.Phase = entity.ImportJobPhaseClone
	}
	if job.Progress == nil {
		job.Progress = entity.ImportProgress{}
	}

	include, err := marshalImportInclude(job.Include)
	if err != nil {
		return err
	}
	progress, err := json.Marshal(job.Progress)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO import_jobs (
			id, organization_id, created_by, source_url, target_repository_id, target_name,
			include, status, phase, progress, token_secret_ref, error, created_at, updated_at
		) VALUES (
			:id, :organization_id, :created_by, :source_url, :target_repository_id, :target_name,
			:include, :status, :phase, :progress, :token_secret_ref, :error, :created_at, :updated_at
		)
	`

	_, err = r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":                   job.ID,
		"organization_id":      job.OrganizationID,
		"created_by":           job.CreatedBy,
		"source_url":           job.SourceURL,
		"target_repository_id": job.TargetRepositoryID,
		"target_name":          job.TargetName,
		"include":              include,
		"status":               job.Status,
		"phase":                job.Phase,
		"progress":             string(progress),
		"token_secret_ref":     job.TokenSecretRef,
		"error":                job.Error,
		"created_at":           job.CreatedAt,
		"updated_at":           job.UpdatedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlImportJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.ImportJob, error) {
	query := `SELECT ` + importJobSelectColumns + ` FROM import_jobs WHERE id = ?`
	query = r.DB.Rebind(query)

	job, err := r.scanImportJob(r.DB.QueryRowxContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return job, nil
}

func (r *sqlImportJobRepository) GetByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (*entity.ImportJob, error) {
	query := `SELECT ` + importJobSelectColumns + ` FROM import_jobs WHERE id = ? AND organization_id = ?`
	query = r.DB.Rebind(query)

	job, err := r.scanImportJob(r.DB.QueryRowxContext(ctx, query, id, orgID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("import job not found: %w", sql.ErrNoRows)
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return job, nil
}

func (r *sqlImportJobRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.ImportJob, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	countQuery := `SELECT COUNT(*) FROM import_jobs WHERE organization_id = ?`
	countQuery = r.DB.Rebind(countQuery)

	var total int
	if err := r.DB.QueryRowxContext(ctx, countQuery, orgID).Scan(&total); err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}

	listQuery := `
		SELECT ` + importJobSelectColumns + `
		FROM import_jobs
		WHERE organization_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	listQuery = r.DB.Rebind(listQuery)

	rows, err := r.DB.QueryxContext(ctx, listQuery, orgID, perPage, offset)
	if err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	jobs, err := r.scanImportJobRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return jobs, total, nil
}

func (r *sqlImportJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.ImportJobStatus) error {
	query := `UPDATE import_jobs SET status = ?, updated_at = ? WHERE id = ?`
	query = r.DB.Rebind(query)
	_, err := r.DB.ExecContext(ctx, query, status, time.Now().UTC(), id)
	return dbErrors.MapDBError(err)
}

func (r *sqlImportJobRepository) UpdatePhase(ctx context.Context, id uuid.UUID, phase entity.ImportJobPhase) error {
	query := `UPDATE import_jobs SET phase = ?, updated_at = ? WHERE id = ?`
	query = r.DB.Rebind(query)
	_, err := r.DB.ExecContext(ctx, query, phase, time.Now().UTC(), id)
	return dbErrors.MapDBError(err)
}

func (r *sqlImportJobRepository) UpdateProgress(ctx context.Context, id uuid.UUID, progress entity.ImportProgress) error {
	data, err := json.Marshal(progress)
	if err != nil {
		return err
	}

	query := `UPDATE import_jobs SET progress = ?, updated_at = ? WHERE id = ?`
	query = r.DB.Rebind(query)
	_, err = r.DB.ExecContext(ctx, query, string(data), time.Now().UTC(), id)
	return dbErrors.MapDBError(err)
}

func (r *sqlImportJobRepository) SetError(ctx context.Context, id uuid.UUID, errMsg string) error {
	query := `UPDATE import_jobs SET error = ?, updated_at = ? WHERE id = ?`
	query = r.DB.Rebind(query)
	_, err := r.DB.ExecContext(ctx, query, errMsg, time.Now().UTC(), id)
	return dbErrors.MapDBError(err)
}

func (r *sqlImportJobRepository) SetTargetRepository(ctx context.Context, id, repoID uuid.UUID) error {
	query := `UPDATE import_jobs SET target_repository_id = ?, updated_at = ? WHERE id = ?`
	query = r.DB.Rebind(query)
	_, err := r.DB.ExecContext(ctx, query, repoID, time.Now().UTC(), id)
	return dbErrors.MapDBError(err)
}

func (r *sqlImportJobRepository) UpsertMapping(ctx context.Context, m *entity.ImportUserMapping) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}

	const query = `
		INSERT INTO import_user_mappings (id, import_job_id, github_login, github_display_name, local_user_id)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(import_job_id, github_login) DO UPDATE SET
			github_display_name = excluded.github_display_name,
			local_user_id = excluded.local_user_id
	`

	q := r.DB.Rebind(query)
	_, err := r.DB.ExecContext(ctx, q, m.ID, m.ImportJobID, m.GitHubLogin, m.GitHubDisplayName, m.LocalUserID)
	return dbErrors.MapDBError(err)
}

func (r *sqlImportJobRepository) GetMappingByLogin(ctx context.Context, jobID uuid.UUID, githubLogin string) (*entity.ImportUserMapping, error) {
	const query = `
		SELECT id, import_job_id, github_login, github_display_name, local_user_id
		FROM import_user_mappings
		WHERE import_job_id = ? AND github_login = ?
	`
	q := r.DB.Rebind(query)

	mapping, err := r.scanImportUserMapping(r.DB.QueryRowxContext(ctx, q, jobID, githubLogin))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return mapping, nil
}

func (r *sqlImportJobRepository) ListMappings(ctx context.Context, jobID uuid.UUID) ([]*entity.ImportUserMapping, error) {
	const query = `
		SELECT id, import_job_id, github_login, github_display_name, local_user_id
		FROM import_user_mappings
		WHERE import_job_id = ?
		ORDER BY github_login ASC
	`
	q := r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, q, jobID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	var mappings []*entity.ImportUserMapping
	for rows.Next() {
		mapping, err := r.scanImportUserMapping(rows)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		mappings = append(mappings, mapping)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return mappings, nil
}

func (r *sqlImportJobRepository) SaveCheckpoint(ctx context.Context, cp *entity.ImportPhaseCheckpoint) error {
	const query = `
		INSERT INTO import_phase_checkpoints (import_job_id, phase, last_cursor, completed)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(import_job_id, phase) DO UPDATE SET
			last_cursor = excluded.last_cursor,
			completed = excluded.completed
	`

	completed := 0
	if cp.Completed {
		completed = 1
	}

	q := r.DB.Rebind(query)
	_, err := r.DB.ExecContext(ctx, q, cp.ImportJobID, cp.Phase, cp.LastCursor, completed)
	return dbErrors.MapDBError(err)
}

func (r *sqlImportJobRepository) GetCheckpoint(ctx context.Context, jobID uuid.UUID, phase entity.ImportJobPhase) (*entity.ImportPhaseCheckpoint, error) {
	const query = `
		SELECT import_job_id, phase, last_cursor, completed
		FROM import_phase_checkpoints
		WHERE import_job_id = ? AND phase = ?
	`
	q := r.DB.Rebind(query)

	cp, err := r.scanImportPhaseCheckpoint(r.DB.QueryRowxContext(ctx, q, jobID, phase))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return cp, nil
}

func (r *sqlImportJobRepository) MarkPhaseComplete(ctx context.Context, jobID uuid.UUID, phase entity.ImportJobPhase) error {
	const query = `
		INSERT INTO import_phase_checkpoints (import_job_id, phase, last_cursor, completed)
		VALUES (?, ?, '', 1)
		ON CONFLICT(import_job_id, phase) DO UPDATE SET completed = 1
	`
	q := r.DB.Rebind(query)
	_, err := r.DB.ExecContext(ctx, q, jobID, phase)
	return dbErrors.MapDBError(err)
}

type importRowScanner interface {
	Scan(dest ...any) error
}

func (r *sqlImportJobRepository) scanImportJob(row importRowScanner) (*entity.ImportJob, error) {
	var (
		job                entity.ImportJob
		targetRepositoryID sql.NullString
		includeRaw         string
		progressRaw        string
		tokenSecretRef     sql.NullString
		errorMsg           sql.NullString
	)

	err := row.Scan(
		&job.ID,
		&job.OrganizationID,
		&job.CreatedBy,
		&job.SourceURL,
		&targetRepositoryID,
		&job.TargetName,
		&includeRaw,
		&job.Status,
		&job.Phase,
		&progressRaw,
		&tokenSecretRef,
		&errorMsg,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if targetRepositoryID.Valid {
		parsed, err := uuid.Parse(targetRepositoryID.String)
		if err != nil {
			return nil, err
		}
		job.TargetRepositoryID = &parsed
	}
	if tokenSecretRef.Valid {
		ref := tokenSecretRef.String
		job.TokenSecretRef = &ref
	}
	if errorMsg.Valid {
		msg := errorMsg.String
		job.Error = &msg
	}

	job.Include, err = unmarshalImportInclude(includeRaw)
	if err != nil {
		return nil, err
	}
	if progressRaw != "" {
		if err := json.Unmarshal([]byte(progressRaw), &job.Progress); err != nil {
			return nil, err
		}
	}
	if job.Progress == nil {
		job.Progress = entity.ImportProgress{}
	}

	return &job, nil
}

func (r *sqlImportJobRepository) scanImportJobRows(rows *sqlx.Rows) ([]*entity.ImportJob, error) {
	var jobs []*entity.ImportJob
	for rows.Next() {
		job, err := r.scanImportJob(rows)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return jobs, nil
}

func (r *sqlImportJobRepository) scanImportUserMapping(row importRowScanner) (*entity.ImportUserMapping, error) {
	var (
		mapping     entity.ImportUserMapping
		localUserID sql.NullString
	)

	err := row.Scan(
		&mapping.ID,
		&mapping.ImportJobID,
		&mapping.GitHubLogin,
		&mapping.GitHubDisplayName,
		&localUserID,
	)
	if err != nil {
		return nil, err
	}

	if localUserID.Valid {
		parsed, err := uuid.Parse(localUserID.String)
		if err != nil {
			return nil, err
		}
		mapping.LocalUserID = &parsed
	}

	return &mapping, nil
}

func (r *sqlImportJobRepository) scanImportPhaseCheckpoint(row importRowScanner) (*entity.ImportPhaseCheckpoint, error) {
	var (
		cp        entity.ImportPhaseCheckpoint
		completed int
	)

	err := row.Scan(&cp.ImportJobID, &cp.Phase, &cp.LastCursor, &completed)
	if err != nil {
		return nil, err
	}
	cp.Completed = completed != 0
	return &cp, nil
}

func marshalImportInclude(include []string) (string, error) {
	if include == nil {
		include = []string{}
	}
	data, err := json.Marshal(include)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalImportInclude(raw string) ([]string, error) {
	if raw == "" {
		return []string{}, nil
	}
	var include []string
	if err := json.Unmarshal([]byte(raw), &include); err != nil {
		return nil, err
	}
	if include == nil {
		return []string{}, nil
	}
	return include, nil
}
