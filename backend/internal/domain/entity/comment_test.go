package entity_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

func TestCommentFields(t *testing.T) {
	comment := entity.Comment{}
	if comment.ID != uuid.Nil {
		t.Fatalf("expected zero-value ID, got %v", comment.ID)
	}

	id := uuid.New()
	issueID := uuid.New()
	authorID := uuid.New()
	orgID := uuid.New()
	createdAt := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	comment = entity.Comment{
		ID:             id,
		IssueID:        issueID,
		AuthorID:       authorID,
		OrganizationID: orgID,
		Body:           "hello world",
		CreatedAt:      createdAt,
	}

	if comment.ID != id {
		t.Fatalf("expected ID %v, got %v", id, comment.ID)
	}
	if comment.IssueID != issueID {
		t.Fatalf("expected IssueID %v, got %v", issueID, comment.IssueID)
	}
	if comment.AuthorID != authorID {
		t.Fatalf("expected AuthorID %v, got %v", authorID, comment.AuthorID)
	}
	if comment.OrganizationID != orgID {
		t.Fatalf("expected OrganizationID %v, got %v", orgID, comment.OrganizationID)
	}
	if comment.Body != "hello world" {
		t.Fatalf("expected Body %q, got %q", "hello world", comment.Body)
	}
	if !comment.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected CreatedAt %v, got %v", createdAt, comment.CreatedAt)
	}
}
