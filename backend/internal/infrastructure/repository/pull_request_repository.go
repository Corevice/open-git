package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
)

type sqlxPullRequestRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IPullRequestRepository = (*sqlxPullRequestRepository)(nil)

func NewPullRequestRepository(db *sqlx.DB) domainrepo.IPullRequestRepository {
	return &sqlxPullRequestRepository{db: db}
}

const pullRequestSelectBase = `
	SELECT
		id, organization_id, repository_id, number, title, body, draft,
		head_ref, base_ref, head_sha, base_sha, state, merged_at, merged_by,
		merge_commit_sha, mergeable, mergeable_state, author_id, created_at, updated_at
	FROM pull_requests
`

func (r *sqlxPullRequestRepository) Create(ctx context.Context, pr *entity.PullRequest) error {
	if pr.ID == uuid.Nil {
		pr.ID = uuid.New()
	}
	now := time.Now().UTC()
	if pr.CreatedAt.IsZero() {
		pr.CreatedAt = now
	}
	if pr.UpdatedAt.IsZero() {
		pr.UpdatedAt = now
	}
	if pr.State == "" {
		pr.State = entity.PullRequestStateOpen
	}
	if pr.MergeableState == "" {
		pr.MergeableState = entity.MergeableStateUnknown
	}

	const query = `
		INSERT INTO pull_requests (
			id, organization_id, repository_id, number, title, body, draft,
			head_ref, base_ref, head_sha, base_sha, state, merged_at, merged_by,
			merge_commit_sha, mergeable, mergeable_state, author_id, created_at, updated_at
		) VALUES (
			:id, :organization_id, :repository_id, :number, :title, :body, :draft,
			:head_ref, :base_ref, :head_sha, :base_sha, :state, :merged_at, :merged_by,
			:merge_commit_sha, :mergeable, :mergeable_state, :author_id, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":               pr.ID,
		"organization_id":  pr.OrganizationID,
		"repository_id":    pr.RepositoryID,
		"number":           pr.Number,
		"title":            pr.Title,
		"body":             pr.Body,
		"draft":            boolToInt(pr.Draft),
		"head_ref":         pr.HeadRef,
		"base_ref":         pr.BaseRef,
		"head_sha":         pr.HeadSHA,
		"base_sha":         pr.BaseSHA,
		"state":            pr.State,
		"merged_at":        pr.MergedAt,
		"merged_by":        pr.MergedBy,
		"merge_commit_sha": nullString(pr.MergeCommitSHA),
		"mergeable":        boolPtrToNullInt(pr.Mergeable),
		"mergeable_state":  pr.MergeableState,
		"author_id":        pr.AuthorID,
		"created_at":       pr.CreatedAt,
		"updated_at":       pr.UpdatedAt,
	})
	return err
}

func (r *sqlxPullRequestRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.PullRequest, error) {
	query := pullRequestSelectBase + ` WHERE id = $1`
	return r.scanOne(ctx, query, id)
}

func (r *sqlxPullRequestRepository) GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.PullRequest, error) {
	query := pullRequestSelectBase + ` WHERE repository_id = $1 AND number = $2`
	return r.scanOne(ctx, query, repoID, number)
}

func (r *sqlxPullRequestRepository) ListByRepo(
	ctx context.Context,
	repoID uuid.UUID,
	filter domainrepo.ListPullRequestsFilter,
) ([]*entity.PullRequest, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	where, args := buildListPullRequestsWhere(repoID, filter)

	countQuery := `SELECT COUNT(*) FROM pull_requests` + where
	var total int
	if err := r.db.QueryRowxContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := pullRequestSelectBase + where + `
		ORDER BY number DESC
		LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)

	listArgs := append(append([]any{}, args...), perPage, offset)
	rows, err := r.db.QueryxContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	prs, err := scanPullRequests(rows)
	if err != nil {
		return nil, 0, err
	}
	return prs, total, nil
}

func (r *sqlxPullRequestRepository) NextNumber(ctx context.Context, repoID uuid.UUID) (int, error) {
	const query = `SELECT COALESCE(MAX(number), 0) + 1 FROM pull_requests WHERE repository_id = $1`

	var next int
	if err := r.db.QueryRowxContext(ctx, query, repoID).Scan(&next); err != nil {
		return 0, err
	}
	return next, nil
}

func (r *sqlxPullRequestRepository) Update(ctx context.Context, pr *entity.PullRequest) error {
	const query = `
		UPDATE pull_requests
		SET title = :title, body = :body, state = :state, draft = :draft,
			head_sha = :head_sha, mergeable = :mergeable, mergeable_state = :mergeable_state,
			updated_at = :updated_at
		WHERE id = :id
	`

	pr.UpdatedAt = time.Now().UTC()
	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":               pr.ID,
		"title":            pr.Title,
		"body":             pr.Body,
		"state":            pr.State,
		"draft":            boolToInt(pr.Draft),
		"head_sha":         pr.HeadSHA,
		"mergeable":        boolPtrToNullInt(pr.Mergeable),
		"mergeable_state":  pr.MergeableState,
		"updated_at":       pr.UpdatedAt,
	})
	return err
}

func (r *sqlxPullRequestRepository) SetMerged(ctx context.Context, id uuid.UUID, mergedAt time.Time, mergedBy uuid.UUID, sha string) error {
	const query = `
		UPDATE pull_requests
		SET state = 'merged', merged_at = :merged_at, merged_by = :merged_by,
			merge_commit_sha = :merge_commit_sha, updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":               id,
		"merged_at":        mergedAt,
		"merged_by":        mergedBy,
		"merge_commit_sha": sha,
		"updated_at":       time.Now().UTC(),
	})
	return err
}

func (r *sqlxPullRequestRepository) scanOne(ctx context.Context, query string, args ...any) (*entity.PullRequest, error) {
	row := r.db.QueryRowxContext(ctx, query, args...)
	pr, err := scanPullRequestRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func buildListPullRequestsWhere(repoID uuid.UUID, filter domainrepo.ListPullRequestsFilter) (string, []any) {
	query := " WHERE repository_id = $1"
	args := []any{repoID}
	idx := 2

	state := filter.State
	if state == "" {
		state = entity.PullRequestStateOpen
	}
	if state != "all" {
		query += " AND state = $" + itoa(idx)
		args = append(args, state)
		idx++
	}

	if filter.HeadRef != "" {
		query += " AND head_ref = $" + itoa(idx)
		args = append(args, filter.HeadRef)
		idx++
	}

	if filter.BaseRef != "" {
		query += " AND base_ref = $" + itoa(idx)
		args = append(args, filter.BaseRef)
	}

	return query, args
}

func scanPullRequests(rows *sqlx.Rows) ([]*entity.PullRequest, error) {
	prs := make([]*entity.PullRequest, 0)
	for rows.Next() {
		pr, err := scanPullRequestRow(rows)
		if err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return prs, nil
}

type pullRequestScanner interface {
	Scan(dest ...any) error
}

func scanPullRequestRow(scanner pullRequestScanner) (*entity.PullRequest, error) {
	var (
		pr             entity.PullRequest
		draft          int
		mergedAt       sql.NullTime
		mergedBy       sql.NullString
		mergeCommitSHA sql.NullString
		mergeable      sql.NullInt64
	)

	if err := scanner.Scan(
		&pr.ID,
		&pr.OrganizationID,
		&pr.RepositoryID,
		&pr.Number,
		&pr.Title,
		&pr.Body,
		&draft,
		&pr.HeadRef,
		&pr.BaseRef,
		&pr.HeadSHA,
		&pr.BaseSHA,
		&pr.State,
		&mergedAt,
		&mergedBy,
		&mergeCommitSHA,
		&mergeable,
		&pr.MergeableState,
		&pr.AuthorID,
		&pr.CreatedAt,
		&pr.UpdatedAt,
	); err != nil {
		return nil, err
	}

	pr.Draft = draft != 0
	if mergedAt.Valid {
		t := mergedAt.Time
		pr.MergedAt = &t
	}
	if mergedBy.Valid {
		id, err := uuid.Parse(mergedBy.String)
		if err != nil {
			return nil, err
		}
		pr.MergedBy = &id
	}
	if mergeCommitSHA.Valid {
		pr.MergeCommitSHA = mergeCommitSHA.String
	}
	if mergeable.Valid {
		v := mergeable.Int64 != 0
		pr.Mergeable = &v
	}

	return &pr, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func boolPtrToNullInt(v *bool) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(boolToInt(*v)), Valid: true}
}

func nullString(v string) sql.NullString {
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: v, Valid: true}
}
