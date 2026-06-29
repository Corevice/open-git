package entity_test

import (
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestWorkflowJobStatusConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"WorkflowJobStatusQueued", entity.WorkflowJobStatusQueued, "queued"},
		{"WorkflowJobStatusInProgress", entity.WorkflowJobStatusInProgress, "in_progress"},
		{"WorkflowJobStatusCompleted", entity.WorkflowJobStatusCompleted, "completed"},
		{"WorkflowJobStatusFailed", entity.WorkflowJobStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Fatalf("%s = %q, want %q", tt.name, tt.value, tt.want)
			}
		})
	}
}

func TestWorkflowJobZeroValueConclusion(t *testing.T) {
	job := entity.WorkflowJob{}
	if job.Conclusion != "" {
		t.Fatalf("Conclusion = %q, want empty string", job.Conclusion)
	}
}
