package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

var commentRowColumns = []string{
	"id", "issue_id", "organization_id", "author_id", "author_login", "body", "created_at", "updated_at",
}

func TestCommentRepository_Create(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	repo := repository.NewCommentRepository(sqlxDB)

	comment := &entity.Comment{
		ID:             uuid.New(),
		IssueID:        uuid.New(),
		OrganizationID: uuid.New(),
		AuthorID:       uuid.New(),
		Body:           "hello world",
	}

	mock.ExpectExec(`INSERT INTO comments`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.Create(context.Background(), comment); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestCommentRepository_ListByIssuePagination(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	repo := repository.NewCommentRepository(sqlxDB)

	issueID := uuid.New()
	orgID := uuid.New()
	authorID := uuid.New()
	now := time.Now().UTC()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM comments WHERE issue_id = \$1`).
		WithArgs(issueID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	mock.ExpectQuery(`FROM comments c`).
		WithArgs(issueID, 10, 10).
		WillReturnRows(sqlmock.NewRows(commentRowColumns).
			AddRow(uuid.New(), issueID, orgID, authorID, "alice", "first", now, now).
			AddRow(uuid.New(), issueID, orgID, authorID, "alice", "second", now.Add(time.Minute), now.Add(time.Minute)))

	comments, total, err := repo.ListByIssue(context.Background(), issueID, 2, 10)
	if err != nil {
		t.Fatalf("ListByIssue: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
