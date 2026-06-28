package entity_test

import (
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestRunnerConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"RunnerTypeAct", entity.RunnerTypeAct, "act"},
		{"RunnerTypeOfficial", entity.RunnerTypeOfficial, "official"},
		{"RunnerStatusOnline", entity.RunnerStatusOnline, "online"},
		{"RunnerStatusOffline", entity.RunnerStatusOffline, "offline"},
		{"RunnerStatusBusy", entity.RunnerStatusBusy, "busy"},
		{"WorkflowJobStatusQueued", entity.WorkflowJobStatusQueued, "queued"},
		{"WorkflowJobStatusInProgress", entity.WorkflowJobStatusInProgress, "in_progress"},
		{"WorkflowJobStatusCompleted", entity.WorkflowJobStatusCompleted, "completed"},
		{"WorkflowJobStatusFailed", entity.WorkflowJobStatusFailed, "failed"},
		{"WorkflowJobStatusCancelled", entity.WorkflowJobStatusCancelled, "cancelled"},
		{"WorkflowJobConclusionSuccess", entity.WorkflowJobConclusionSuccess, "success"},
		{"WorkflowJobConclusionFailure", entity.WorkflowJobConclusionFailure, "failure"},
		{"WorkflowJobConclusionCancelled", entity.WorkflowJobConclusionCancelled, "cancelled"},
		{"WorkflowJobConclusionQuotaExceeded", entity.WorkflowJobConclusionQuotaExceeded, "quota_exceeded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Fatalf("%s = %q, want %q", tt.name, tt.value, tt.want)
			}
		})
	}
}

func TestRunnerZeroValueStatus(t *testing.T) {
	runner := entity.Runner{Status: entity.RunnerStatusOffline}
	if runner.Status != entity.RunnerStatusOffline {
		t.Fatalf("Status = %q, want %q", runner.Status, entity.RunnerStatusOffline)
	}
	if runner.Status != "offline" {
		t.Fatalf("Status = %q, want offline", runner.Status)
	}
}

func TestWorkflowJobDefaultStatus(t *testing.T) {
	job := entity.WorkflowJob{Status: entity.WorkflowJobStatusQueued}
	if job.Status != entity.WorkflowJobStatusQueued {
		t.Fatalf("Status = %q, want %q", job.Status, entity.WorkflowJobStatusQueued)
	}
	if job.Status != "queued" {
		t.Fatalf("Status = %q, want queued", job.Status)
	}
}
