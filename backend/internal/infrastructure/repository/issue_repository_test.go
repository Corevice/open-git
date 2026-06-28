package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

var issueRowColumns = []string{
	"id", "organization_id", "repository_id", "number", "title", "body",
	"state", "state_reason", "author_id", "author_login",
	"milestone_id", "comments_count", "created_at", "updated_at", "closed_at",
	"label_id", "label_repository_id", "label_organization_id",
	"label_name", "label_color", "label_description", "label_created_at",
}

func TestIssueRepository_UpdateSuccess(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	repo := repository.NewIssueRepository(sqlxDB)

	issueID := uuid.New()
	milestoneID := uuid.New()
	stateReason := "completed"
	issue := &entity.Issue{
		ID:            issueID,
		Title:         "updated title",
		Body:          "updated body",
		State:         "closed",
		StateReason:   &stateReason,
		MilestoneID:   &milestoneID,
		CommentsCount: 3,
	}

	mock.ExpectExec(`UPDATE issues`).
		WithArgs(issue.Title, issue.Body, issue.State, issue.StateReason, issue.MilestoneID, issue.CommentsCount, sqlmock.AnyArg(), issueID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.Update(context.Background(), issue); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestIssueRepository_UpdateError(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	repo := repository.NewIssueRepository(sqlxDB)

	issue := &entity.Issue{
		ID:    uuid.New(),
		Title: "title",
		Body:  "body",
		State: "open",
	}

	mock.ExpectExec(`UPDATE issues`).
		WillReturnError(errors.New("update failed"))

	if err := repo.Update(context.Background(), issue); err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestIssueRepository_DeleteSuccess(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	repo := repository.NewIssueRepository(sqlxDB)

	issueID := uuid.New()

	mock.ExpectExec(`UPDATE issues SET state = 'deleted'`).
		WithArgs(issueID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.Delete(context.Background(), issueID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestIssueRepository_Count(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	repo := repository.NewIssueRepository(sqlxDB)

	repoID := uuid.New()
	filter := domainrepo.ListIssuesFilter{
		RepositoryID: repoID,
		State:        "open",
	}

	mock.ExpectQuery(`SELECT COUNT\(DISTINCT i\.id\)`).
		WithArgs(repoID, "open").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	count, err := repo.Count(context.Background(), filter)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 42 {
		t.Fatalf("expected count 42, got %d", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestIssueRepository_GetByIDReturnsAuthorLogin(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	repo := repository.NewIssueRepository(sqlxDB)

	issueID := uuid.New()
	orgID := uuid.New()
	repoID := uuid.New()
	authorID := uuid.New()
	now := time.Now().UTC()

	mock.ExpectQuery(`FROM issues i`).
		WithArgs(issueID).
		WillReturnRows(sqlmock.NewRows(issueRowColumns).
			AddRow(issueID, orgID, repoID, 1, "title", "body", "open", nil, authorID, "alice", nil, 0, now, now, nil, nil, nil, nil, nil, nil, nil, nil))

	got, err := repo.GetByID(context.Background(), issueID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected issue, got nil")
	}
	if got.AuthorLogin != "alice" {
		t.Fatalf("expected author login alice, got %q", got.AuthorLogin)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
