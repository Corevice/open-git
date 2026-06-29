package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxRunnerRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IRunnerRepository = (*sqlxRunnerRepository)(nil)

func NewRunnerRepository(db *sqlx.DB) domainrepo.IRunnerRepository {
	return &sqlxRunnerRepository{db: db}
}

func (r *sqlxRunnerRepository) Create(ctx context.Context, runner *entity.Runner) error {
	if runner.ID == uuid.Nil {
		runner.ID = uuid.New()
	}
	now := time.Now().UTC()
	if runner.CreatedAt.IsZero() {
		runner.CreatedAt = now
	}
	if runner.UpdatedAt.IsZero() {
		runner.UpdatedAt = now
	}

	labelsJSON, err := json.Marshal(runner.Labels)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO runners (
			id, organization_id, name, labels, os, arch, runner_type, status,
			last_seen_at, ephemeral, created_at, updated_at
		) VALUES (
			:id, :organization_id, :name, :labels, :os, :arch, :runner_type, :status,
			:last_seen_at, :ephemeral, :created_at, :updated_at
		)
	`

	_, err = r.db.NamedExecContext(ctx, query, map[string]any{
		"id":              runner.ID,
		"organization_id": runner.OrganizationID,
		"name":            runner.Name,
		"labels":          string(labelsJSON),
		"os":              runner.OS,
		"arch":            runner.Arch,
		"runner_type":     runner.RunnerType,
		"status":          runner.Status,
		"last_seen_at":    runner.LastSeenAt,
		"ephemeral":       runner.Ephemeral,
		"created_at":      runner.CreatedAt,
		"updated_at":      runner.UpdatedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxRunnerRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Runner, error) {
	const query = `
		SELECT id, organization_id, name, labels, os, arch, runner_type, status,
			last_seen_at, ephemeral, created_at, updated_at
		FROM runners
		WHERE id = ?
	`
	q := r.db.Rebind(query)
	row := r.db.QueryRowxContext(ctx, q, id)

	runner, err := scanRunnerRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return runner, nil
}

func (r *sqlxRunnerRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entity.Runner, error) {
	const query = `
		SELECT id, organization_id, name, labels, os, arch, runner_type, status,
			last_seen_at, ephemeral, created_at, updated_at
		FROM runners
		WHERE organization_id = ?
		ORDER BY created_at ASC
	`
	q := r.db.Rebind(query)
	rows, err := r.db.QueryxContext(ctx, q, orgID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	return scanRunnerRows(rows)
}

func (r *sqlxRunnerRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, lastSeenAt time.Time) error {
	const query = `
		UPDATE runners
		SET status = ?, last_seen_at = ?, updated_at = ?
		WHERE id = ?
	`
	q := r.db.Rebind(query)
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, q, status, lastSeenAt, now, id)
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

func (r *sqlxRunnerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM runners WHERE id = ?`
	q := r.db.Rebind(query)
	result, err := r.db.ExecContext(ctx, q, id)
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

func (r *sqlxRunnerRepository) FindAvailable(ctx context.Context, orgID uuid.UUID, labels []string) (*entity.Runner, error) {
	const query = `
		SELECT id, organization_id, name, labels, os, arch, runner_type, status,
			last_seen_at, ephemeral, created_at, updated_at
		FROM runners
		WHERE organization_id = ? AND status = 'online'
		ORDER BY created_at ASC
	`
	q := r.db.Rebind(query)
	rows, err := r.db.QueryxContext(ctx, q, orgID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	runners, err := scanRunnerRows(rows)
	if err != nil {
		return nil, err
	}

	for _, runner := range runners {
		if runnerLabelsMatch(runner.Labels, labels) {
			return runner, nil
		}
	}
	return nil, domain.ErrNotFound
}

func runnerLabelsMatch(runnerLabels, required []string) bool {
	if len(required) == 0 {
		return true
	}
	labelSet := make(map[string]bool, len(runnerLabels))
	for _, label := range runnerLabels {
		labelSet[label] = true
	}
	for _, label := range required {
		if !labelSet[label] {
			return false
		}
	}
	return true
}

type runnerScanner interface {
	Scan(dest ...any) error
}

func scanRunnerRow(scanner runnerScanner) (*entity.Runner, error) {
	var (
		runner      entity.Runner
		labelsJSON  string
		lastSeenAt  sql.NullTime
		ephemeral   bool
	)
	if err := scanner.Scan(
		&runner.ID,
		&runner.OrganizationID,
		&runner.Name,
		&labelsJSON,
		&runner.OS,
		&runner.Arch,
		&runner.RunnerType,
		&runner.Status,
		&lastSeenAt,
		&ephemeral,
		&runner.CreatedAt,
		&runner.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if lastSeenAt.Valid {
		t := lastSeenAt.Time
		runner.LastSeenAt = &t
	}
	runner.Ephemeral = ephemeral
	if labelsJSON != "" {
		if err := json.Unmarshal([]byte(labelsJSON), &runner.Labels); err != nil {
			return nil, err
		}
	}
	return &runner, nil
}

func scanRunnerRows(rows *sqlx.Rows) ([]*entity.Runner, error) {
	runners := make([]*entity.Runner, 0)
	for rows.Next() {
		runner, err := scanRunnerRow(rows)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		runners = append(runners, runner)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return runners, nil
}
