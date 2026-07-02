package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

// sqlxJobLogRepository persists CI job log lines and per-job log metadata
// (tables job_log_lines / job_logs_meta). It backs both the CI executor's log
// writes and the actions log read/stream endpoints.
type sqlxJobLogRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IJobLogRepository = (*sqlxJobLogRepository)(nil)

func NewJobLogRepository(db *sqlx.DB) *sqlxJobLogRepository {
	return &sqlxJobLogRepository{db: db}
}

func (r *sqlxJobLogRepository) AppendLines(ctx context.Context, lines []*entity.JobLogLine) error {
	if len(lines) == 0 {
		return nil
	}

	// id is a plain INTEGER PK (not auto-increment on Postgres), so ids are
	// generated here. Random 63-bit ids keep appends contention-free; ordering
	// uses line_number, never id.
	const query = `
		INSERT INTO job_log_lines (
			id, organization_id, repository_id, run_id, job_id, step_index,
			line_number, stream, text, created_at
		) VALUES (
			:id, :organization_id, :repository_id, :run_id, :job_id, :step_index,
			:line_number, :stream, :text, :created_at
		)
		ON CONFLICT (job_id, line_number) DO NOTHING
	`

	for _, line := range lines {
		if line.ID == 0 {
			id, err := randomTokenID()
			if err != nil {
				return err
			}
			line.ID = id
		}
		if line.CreatedAt.IsZero() {
			line.CreatedAt = time.Now().UTC()
		}
		stream := line.Stream
		if stream == "" {
			stream = "stdout"
		}
		if _, err := r.db.NamedExecContext(ctx, query, map[string]any{
			"id":              line.ID,
			"organization_id": line.OrganizationID,
			"repository_id":   line.RepositoryID,
			"run_id":          line.RunID,
			"job_id":          line.JobID,
			"step_index":      line.StepIndex,
			"line_number":     line.LineNumber,
			"stream":          stream,
			"text":            line.Text,
			"created_at":      line.CreatedAt,
		}); err != nil {
			return dbErrors.MapDBError(err)
		}
	}
	return nil
}

func (r *sqlxJobLogRepository) ListLines(ctx context.Context, orgID, jobID string, fromLine int64, limit int) ([]*entity.JobLogLine, error) {
	if limit < 1 {
		limit = 1000
	}
	query := r.db.Rebind(`
		SELECT id, organization_id, repository_id, run_id, job_id, step_index,
			line_number, stream, text, created_at
		FROM job_log_lines
		WHERE organization_id = ? AND job_id = ? AND line_number >= ?
		ORDER BY line_number ASC
		LIMIT ?
	`)

	rows, err := r.db.QueryxContext(ctx, query, orgID, jobID, fromLine, limit)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	lines := make([]*entity.JobLogLine, 0)
	for rows.Next() {
		var line entity.JobLogLine
		if err := rows.Scan(
			&line.ID,
			&line.OrganizationID,
			&line.RepositoryID,
			&line.RunID,
			&line.JobID,
			&line.StepIndex,
			&line.LineNumber,
			&line.Stream,
			&line.Text,
			&line.CreatedAt,
		); err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		lines = append(lines, &line)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return lines, nil
}

func (r *sqlxJobLogRepository) CountLines(ctx context.Context, orgID, jobID string) (int64, error) {
	query := r.db.Rebind(`SELECT COUNT(*) FROM job_log_lines WHERE organization_id = ? AND job_id = ?`)
	var count int64
	if err := r.db.GetContext(ctx, &count, query, orgID, jobID); err != nil {
		return 0, dbErrors.MapDBError(err)
	}
	return count, nil
}

func (r *sqlxJobLogRepository) SetMeta(ctx context.Context, meta *domainrepo.JobLogMeta) error {
	const query = `
		INSERT INTO job_logs_meta (job_id, organization_id, total_lines, status)
		VALUES (:job_id, :organization_id, :total_lines, :status)
		ON CONFLICT (job_id) DO UPDATE SET
			total_lines = excluded.total_lines,
			status = excluded.status
	`
	status := meta.Status
	if status == "" {
		status = "running"
	}
	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"job_id":          meta.JobID,
		"organization_id": meta.OrganizationID,
		"total_lines":     meta.TotalLines,
		"status":          status,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxJobLogRepository) GetMeta(ctx context.Context, orgID, jobID string) (*domainrepo.JobLogMeta, error) {
	query := r.db.Rebind(`
		SELECT job_id, organization_id, total_lines, status
		FROM job_logs_meta
		WHERE organization_id = ? AND job_id = ?
	`)

	var meta domainrepo.JobLogMeta
	err := r.db.QueryRowxContext(ctx, query, orgID, jobID).Scan(
		&meta.JobID,
		&meta.OrganizationID,
		&meta.TotalLines,
		&meta.Status,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return &meta, nil
}
