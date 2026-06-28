package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
)

const issueNextNumberMaxRetries = 5

type sqlxIssueRepository struct {
	*sqlx.DB
}

var _ domainrepo.IIssueRepository = (*sqlxIssueRepository)(nil)

func NewIssueRepository(db *sqlx.DB) *sqlxIssueRepository {
	return &sqlxIssueRepository{DB: db}
}

const issueSelectBase = `
	SELECT
		i.id, i.organization_id, i.repository_id, i.number, i.title, i.body,
		i.state, i.state_reason, i.author_id, COALESCE(u.login, '') AS author_login,
		i.milestone_id, i.comments_count, i.created_at, i.updated_at, i.closed_at,
		l.id AS label_id, l.repository_id AS label_repository_id, l.organization_id AS label_organization_id,
		l.name AS label_name, l.color AS label_color, l.description AS label_description, l.created_at AS label_created_at
	FROM issues i
	LEFT JOIN users u ON i.author_id = u.id
	LEFT JOIN issue_labels il ON i.id = il.issue_id
	LEFT JOIN labels l ON il.label_id = l.id
`

func (r *sqlxIssueRepository) Create(ctx context.Context, issue *entity.Issue) error {
	if issue.ID == uuid.Nil {
		issue.ID = uuid.New()
	}

	const query = `
		INSERT INTO issues (id, organization_id, repository_id, number, title, body, state, author_id, created_at)
		VALUES (:id, :organization_id, :repository_id, :number, :title, :body, :state, :author_id, :created_at)
	`

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":              issue.ID,
		"organization_id": issue.OrganizationID,
		"repository_id":   issue.RepositoryID,
		"number":          issue.Number,
		"title":           issue.Title,
		"body":            issue.Body,
		"state":           issue.State,
		"author_id":       issue.AuthorID,
		"created_at":      time.Now().UTC(),
	})
	return err
}

func (r *sqlxIssueRepository) GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.Issue, error) {
	query := issueSelectBase + `
		WHERE i.repository_id = $1 AND i.number = $2
	`

	rows, err := r.DB.QueryxContext(ctx, query, repoID, number)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	issues, err := scanIssuesWithLabels(rows)
	if err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		return nil, nil
	}
	return issues[0], nil
}

func (r *sqlxIssueRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Issue, error) {
	query := issueSelectBase + `
		WHERE i.id = $1
	`

	rows, err := r.DB.QueryxContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	issues, err := scanIssuesWithLabels(rows)
	if err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		return nil, nil
	}
	return issues[0], nil
}

func (r *sqlxIssueRepository) ListByRepo(ctx context.Context, filter domainrepo.ListIssuesFilter) ([]*entity.Issue, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	where, args := buildListIssuesWhere(filter, "i")
	subWhere, _ := buildListIssuesWhere(filter, "i2")
	orderBy := buildListIssuesOrderBy(filter, "i")
	subOrderBy := buildListIssuesOrderBy(filter, "i2")

	countQuery := `
		SELECT COUNT(DISTINCT i.id)
		FROM issues i
	` + where

	var total int
	if err := r.DB.QueryRowxContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `
		SELECT
			i.id, i.organization_id, i.repository_id, i.number, i.title, i.body,
			i.state, i.state_reason, i.author_id, COALESCE(u.login, '') AS author_login,
			i.milestone_id, i.comments_count, i.created_at, i.updated_at, i.closed_at,
			l.id AS label_id, l.repository_id AS label_repository_id, l.organization_id AS label_organization_id,
			l.name AS label_name, l.color AS label_color, l.description AS label_description, l.created_at AS label_created_at
		FROM issues i
		LEFT JOIN users u ON i.author_id = u.id
		LEFT JOIN issue_labels il ON i.id = il.issue_id
		LEFT JOIN labels l ON il.label_id = l.id
		WHERE i.id IN (
			SELECT i2.id
			FROM issues i2
	` + subWhere + `
			` + subOrderBy + `
			LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2) + `
		)
		` + orderBy

	listArgs := append(append([]any{}, args...), perPage, offset)
	rows, err := r.DB.QueryxContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	issues, err := scanIssuesWithLabels(rows)
	if err != nil {
		return nil, 0, err
	}
	return issues, total, nil
}

func (r *sqlxIssueRepository) Update(ctx context.Context, issue *entity.Issue) error {
	const query = `
		UPDATE issues
		SET title = $1, body = $2, state = $3, state_reason = $4, milestone_id = $5, comments_count = $6, updated_at = $7
		WHERE id = $8
	`

	_, err := r.DB.ExecContext(
		ctx,
		query,
		issue.Title,
		issue.Body,
		issue.State,
		issue.StateReason,
		issue.MilestoneID,
		issue.CommentsCount,
		time.Now().UTC(),
		issue.ID,
	)
	return err
}

func (r *sqlxIssueRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE issues SET state = 'deleted' WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *sqlxIssueRepository) Count(ctx context.Context, filter domainrepo.ListIssuesFilter) (int, error) {
	where, args := buildListIssuesWhere(filter, "i")

	query := `
		SELECT COUNT(DISTINCT i.id)
		FROM issues i
	` + where

	var count int
	if err := r.DB.QueryRowxContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *sqlxIssueRepository) NextNumber(ctx context.Context, repoID uuid.UUID) (int, error) {
	const query = `SELECT COALESCE(MAX(number), 0) + 1 FROM issues WHERE repository_id = $1`

	var lastErr error
	for attempt := 0; attempt < issueNextNumberMaxRetries; attempt++ {
		tx, err := r.DB.BeginTxx(ctx, nil)
		if err != nil {
			return 0, err
		}

		var next int
		if err := tx.QueryRowxContext(ctx, query, repoID).Scan(&next); err != nil {
			_ = tx.Rollback()
			return 0, err
		}

		if err := tx.Commit(); err != nil {
			lastErr = err
			if isUniqueViolation(err) {
				continue
			}
			return 0, err
		}
		return next, nil
	}

	if lastErr != nil {
		return 0, lastErr
	}
	return 0, errors.New("failed to allocate issue number")
}

func buildListIssuesWhere(filter domainrepo.ListIssuesFilter, alias string) (string, []any) {
	query := " WHERE " + alias + ".repository_id = $1"
	args := []any{filter.RepositoryID}
	idx := 2

	if filter.OrganizationID != uuid.Nil {
		query += " AND " + alias + ".organization_id = $" + itoa(idx)
		args = append(args, filter.OrganizationID)
		idx++
	}

	if filter.State != "" {
		query += " AND " + alias + ".state = $" + itoa(idx)
		args = append(args, filter.State)
		idx++
	} else {
		query += " AND " + alias + ".state != 'deleted'"
	}

	if len(filter.Labels) > 0 {
		placeholders := make([]string, len(filter.Labels))
		for i, label := range filter.Labels {
			placeholders[i] = "$" + itoa(idx)
			args = append(args, label)
			idx++
		}
		query += `
			AND ` + alias + `.id IN (
				SELECT il.issue_id
				FROM issue_labels il
				JOIN labels l ON il.label_id = l.id
				WHERE l.name IN (` + strings.Join(placeholders, ", ") + `)
				GROUP BY il.issue_id
				HAVING COUNT(DISTINCT l.name) = ` + itoa(len(filter.Labels)) + `
			)`
	}

	if filter.MilestoneNumber != nil {
		query += `
			AND ` + alias + `.milestone_id IN (
				SELECT id FROM milestones WHERE repository_id = $1 AND number = $` + itoa(idx) + `
			)`
		args = append(args, *filter.MilestoneNumber)
		idx++
	}

	if filter.Assignee != "" {
		query += `
			AND ` + alias + `.id IN (
				SELECT ia.issue_id
				FROM issue_assignees ia
				JOIN users au ON ia.user_id = au.id
				WHERE au.login = $` + itoa(idx) + `
			)`
		args = append(args, filter.Assignee)
		idx++
	}

	return query, args
}

func buildListIssuesOrderBy(filter domainrepo.ListIssuesFilter, alias string) string {
	column := alias + ".number"
	switch filter.Sort {
	case "created":
		column = alias + ".created_at"
	case "updated":
		column = alias + ".updated_at"
	case "comments":
		column = alias + ".comments_count"
	}

	direction := "DESC"
	if strings.EqualFold(filter.Direction, "asc") {
		direction = "ASC"
	}

	return "ORDER BY " + column + " " + direction
}

func scanIssuesWithLabels(rows *sqlx.Rows) ([]*entity.Issue, error) {
	issueMap := make(map[uuid.UUID]*entity.Issue)
	order := make([]uuid.UUID, 0)

	for rows.Next() {
		var (
			issue          entity.Issue
			stateReason    sql.NullString
			milestoneID    sql.NullString
			closedAt       sql.NullTime
			labelID        sql.NullString
			labelRepoID    sql.NullString
			labelOrgID     sql.NullString
			labelName      sql.NullString
			labelColor     sql.NullString
			labelDesc      sql.NullString
			labelCreatedAt sql.NullTime
		)

		if err := rows.Scan(
			&issue.ID,
			&issue.OrganizationID,
			&issue.RepositoryID,
			&issue.Number,
			&issue.Title,
			&issue.Body,
			&issue.State,
			&stateReason,
			&issue.AuthorID,
			&issue.AuthorLogin,
			&milestoneID,
			&issue.CommentsCount,
			&issue.CreatedAt,
			&issue.UpdatedAt,
			&closedAt,
			&labelID,
			&labelRepoID,
			&labelOrgID,
			&labelName,
			&labelColor,
			&labelDesc,
			&labelCreatedAt,
		); err != nil {
			return nil, err
		}

		if stateReason.Valid {
			issue.StateReason = &stateReason.String
		}
		if milestoneID.Valid {
			id, err := uuid.Parse(milestoneID.String)
			if err != nil {
				return nil, err
			}
			issue.MilestoneID = &id
		}
		if closedAt.Valid {
			t := closedAt.Time
			issue.ClosedAt = &t
		}

		existing, ok := issueMap[issue.ID]
		if !ok {
			copyIssue := issue
			copyIssue.Labels = []entity.Label{}
			issueMap[issue.ID] = &copyIssue
			order = append(order, issue.ID)
			existing = &copyIssue
		}

		if labelID.Valid {
			id, err := uuid.Parse(labelID.String)
			if err != nil {
				return nil, err
			}
			repoID, err := uuid.Parse(labelRepoID.String)
			if err != nil {
				return nil, err
			}
			orgID, err := uuid.Parse(labelOrgID.String)
			if err != nil {
				return nil, err
			}

			existing.Labels = append(existing.Labels, entity.Label{
				ID:             id,
				RepositoryID:   repoID,
				OrganizationID: orgID,
				Name:           labelName.String,
				Color:          labelColor.String,
				Description:    labelDesc.String,
				CreatedAt:      labelCreatedAt.Time,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	issues := make([]*entity.Issue, 0, len(order))
	for _, id := range order {
		issues = append(issues, issueMap[id])
	}
	return issues, nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
