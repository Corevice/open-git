package entity_test

import (
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestImportJobStatusConstantsNonEmpty(t *testing.T) {
	tests := []struct {
		name  string
		value entity.ImportJobStatus
		want  string
	}{
		{"ImportJobStatusQueued", entity.ImportJobStatusQueued, "queued"},
		{"ImportJobStatusRunning", entity.ImportJobStatusRunning, "running"},
		{"ImportJobStatusPaused", entity.ImportJobStatusPaused, "paused"},
		{"ImportJobStatusCompleted", entity.ImportJobStatusCompleted, "completed"},
		{"ImportJobStatusFailed", entity.ImportJobStatusFailed, "failed"},
		{"ImportJobStatusCancelled", entity.ImportJobStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Fatalf("%s must not be empty", tt.name)
			}
			if tt.value != entity.ImportJobStatus(tt.want) {
				t.Fatalf("%s = %q, want %q", tt.name, tt.value, tt.want)
			}
		})
	}
}

func TestImportJobPhaseConstantsNonEmpty(t *testing.T) {
	tests := []struct {
		name  string
		value entity.ImportJobPhase
		want  string
	}{
		{"ImportJobPhaseClone", entity.ImportJobPhaseClone, "clone"},
		{"ImportJobPhaseMetadata", entity.ImportJobPhaseMetadata, "metadata"},
		{"ImportJobPhaseIssues", entity.ImportJobPhaseIssues, "issues"},
		{"ImportJobPhasePullRequests", entity.ImportJobPhasePullRequests, "pull_requests"},
		{"ImportJobPhaseWiki", entity.ImportJobPhaseWiki, "wiki"},
		{"ImportJobPhaseDone", entity.ImportJobPhaseDone, "done"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Fatalf("%s must not be empty", tt.name)
			}
			if tt.value != entity.ImportJobPhase(tt.want) {
				t.Fatalf("%s = %q, want %q", tt.name, tt.value, tt.want)
			}
		})
	}
}

func TestImportJobZeroValueStatus(t *testing.T) {
	job := entity.ImportJob{}
	if job.Status != "" {
		t.Fatalf("Status = %q, want empty string", job.Status)
	}
}

func TestImportProgressUnknownPhaseReturnsZeroValue(t *testing.T) {
	progress := entity.ImportProgress{}
	p := progress["unknown"]
	if p.Done != 0 || p.Total != 0 {
		t.Fatalf("unknown phase progress = %+v, want zero value", p)
	}
}
