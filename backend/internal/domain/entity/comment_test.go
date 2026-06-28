package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

func TestCommentFields(t *testing.T) {
	comment := entity.Comment{}
	if comment.ID != uuid.Nil {
		t.Fatalf("expected zero-value ID, got %v", comment.ID)
	}

	comment.Body = "hello world"
	if comment.Body != "hello world" {
		t.Fatalf("expected Body %q, got %q", "hello world", comment.Body)
	}
}
