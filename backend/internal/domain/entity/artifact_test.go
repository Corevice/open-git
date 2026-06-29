package entity_test

import (
	"testing"
	"time"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestArtifactIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		status    entity.ArtifactStatus
		want      bool
	}{
		{
			name:      "past ExpiresAt",
			expiresAt: time.Now().Add(-1 * time.Hour),
			status:    entity.ArtifactStatusCompleted,
			want:      true,
		},
		{
			name:      "future ExpiresAt",
			expiresAt: time.Now().Add(24 * time.Hour),
			status:    entity.ArtifactStatusCompleted,
			want:      false,
		},
		{
			name:      "status expired regardless of time",
			expiresAt: time.Now().Add(24 * time.Hour),
			status:    entity.ArtifactStatusExpired,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &entity.Artifact{
				ExpiresAt: tt.expiresAt,
				Status:    tt.status,
			}
			if got := a.IsExpired(); got != tt.want {
				t.Fatalf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArtifactStorageKey(t *testing.T) {
	got := entity.ArtifactStorageKey("myorg", "myrepo", "runid", "artid", "name")
	want := "org/myorg/repo/myrepo/runs/runid/artid/name.zip"
	if got != want {
		t.Fatalf("ArtifactStorageKey() = %q, want %q", got, want)
	}
}
