package entity_test

import (
	"strings"
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestPullRequestValidateTitle(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{
			name:    "rejects empty string",
			title:   "",
			wantErr: true,
		},
		{
			name:    "accepts 256 rune title",
			title:   strings.Repeat("a", 256),
			wantErr: false,
		},
		{
			name:    "rejects 257 rune title",
			title:   strings.Repeat("a", 257),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &entity.PullRequest{Title: tt.title}
			err := pr.ValidateTitle()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for title length %d", len(tt.title))
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for title length %d: %v", len(tt.title), err)
			}
		})
	}
}

func TestValidateBaseHeadRefs(t *testing.T) {
	if err := entity.ValidateBaseHeadRefs("main", "main"); err == nil {
		t.Fatal("expected error when base and head are the same")
	}
	if err := entity.ValidateBaseHeadRefs("main", "feature"); err != nil {
		t.Fatalf("unexpected error for different refs: %v", err)
	}
}

func TestPullRequestDefaultMergeableState(t *testing.T) {
	if entity.MergeableStateUnknown != "unknown" {
		t.Fatalf("expected MergeableStateUnknown to be %q, got %q", "unknown", entity.MergeableStateUnknown)
	}

	pr := &entity.PullRequest{MergeableState: entity.MergeableStateUnknown}
	if pr.MergeableState != "unknown" {
		t.Fatalf("expected default mergeable state %q, got %q", "unknown", pr.MergeableState)
	}
}
