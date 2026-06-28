package repository

import (
	"context"
	"database/sql"

	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type JobLogRepository struct {
	db *sql.DB
}

var _ domainrepo.IJobLogRepository = (*JobLogRepository)(nil)

func NewJobLogRepository(db *sql.DB) *JobLogRepository {
	return &JobLogRepository{db: db}
}

func (r *JobLogRepository) AppendLines(ctx context.Context, lines []*entity.JobLogLine) error {
	const query = `
		INSERT INTO job_log_lines(organization_id, repository_id, run_id, job_id, step_index, line_number, stream, text, created_at)
		VALUES(?,?,?,?,?,?,?,?,?)
		ON CONFLICT(job_id, line_number) DO NOTHING
	`
	for _, line := range lines {
		_, err := r.db.ExecContext(ctx, query,
			line.OrganizationID,
			line.RepositoryID,
			line.RunID,
			line.JobID,
			line.StepIndex,
			line.LineNumber,
			line.Stream,
			line.Text,
			line.CreatedAt,
		)
		if err != nil {
			return dbErrors.MapDBError(err)
		}
	}
	return nil
}

func (r *JobLogRepository) ListLines(ctx context.Context, orgID, jobID string, fromLine int64, limit int) ([]*entity.JobLogLine, error) {
	const query = `
		SELECT id, organization_id, repository_id, run_id, job_id, step_index, line_number, stream, text, created_at
		FROM job_log_lines
		WHERE organization_id=? AND job_id=? AND line_number>=?
		ORDER BY line_number
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, orgID, jobID, fromLine, limit)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	lines := make([]*entity.JobLogLine, 0)
	for rows.Next() {
		line, err := scanJobLogLine(rows)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return lines, nil
}

func (r *JobLogRepository) CountLines(ctx context.Context, orgID, jobID string) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM job_log_lines
		WHERE organization_id=? AND job_id=?
	`
	var count int64
	if err := r.db.QueryRowContext(ctx, query, orgID, jobID).Scan(&count); err != nil {
		return 0, dbErrors.MapDBError(err)
	}
	return count, nil
}

func (r *JobLogRepository) SetMeta(ctx context.Context, meta *domainrepo.JobLogMeta) error {
	const query = `
		INSERT INTO job_logs_meta(job_id, organization_id, total_lines, status)
		VALUES(?,?,?,?)
		ON CONFLICT(job_id) DO UPDATE SET status=excluded.status, total_lines=excluded.total_lines
	`
	_, err := r.db.ExecContext(ctx, query, meta.JobID, meta.OrganizationID, meta.TotalLines, meta.Status)
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	return nil
}

func (r *JobLogRepository) GetMeta(ctx context.Context, orgID, jobID string) (*domainrepo.JobLogMeta, error) {
	const query = `
		SELECT job_id, organization_id, total_lines, status
		FROM job_logs_meta
		WHERE organization_id=? AND job_id=?
	`
	var meta domainrepo.JobLogMeta
	err := r.db.QueryRowContext(ctx, query, orgID, jobID).Scan(
		&meta.JobID,
		&meta.OrganizationID,
		&meta.TotalLines,
		&meta.Status,
	)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return &meta, nil
}

type jobLogLineScanner interface {
	Scan(dest ...any) error
}

func scanJobLogLine(scanner jobLogLineScanner) (*entity.JobLogLine, error) {
	var line entity.JobLogLine
	if err := scanner.Scan(
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
		return nil, err
	}
	return &line, nil
}
