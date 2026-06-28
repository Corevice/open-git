package repository_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func TestLabelRepository_AddToIssue(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	repo := repository.NewLabelRepository(sqlxDB)

	repoID := uuid.New()
	labelID := uuid.New()
	issueNumber := 7

	mock.ExpectExec(`INSERT INTO issue_labels`).
		WithArgs(repoID, issueNumber, labelID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.AddToIssue(context.Background(), repoID, issueNumber, labelID); err != nil {
		t.Fatalf("AddToIssue: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestLabelRepository_RemoveFromIssue(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	repo := repository.NewLabelRepository(sqlxDB)

	repoID := uuid.New()
	labelID := uuid.New()
	issueNumber := 3

	mock.ExpectExec(`DELETE FROM issue_labels`).
		WithArgs(repoID, issueNumber, labelID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.RemoveFromIssue(context.Background(), repoID, issueNumber, labelID); err != nil {
		t.Fatalf("RemoveFromIssue: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
