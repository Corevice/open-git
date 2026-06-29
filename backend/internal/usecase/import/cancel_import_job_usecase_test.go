package importjob_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain/entity"
	importjob "github.com/open-git/backend/internal/usecase/import"
)

func TestCancelImportJobRunningToCancelled(t *testing.T) {
	jobID := uuid.New()
	importJobs := &mockImportJobRepo{
		jobs: map[uuid.UUID]*entity.ImportJob{
			jobID: {
				ID:             jobID,
				OrganizationID: testOrgID,
				Status:         entity.ImportJobStatusRunning,
			},
		},
	}
	memberships := &mockMembershipRepo{
		roles: map[string]string{
			membershipKey(testOrgID, testCallerID): entity.RoleAdmin,
		},
	}
	uc := importjob.NewCancelImportJobUsecase(importJobs, memberships)

	job, err := uc.Execute(context.Background(), importjob.CancelImportJobInput{
		OrganizationID: testOrgID,
		JobID:          jobID,
		CallerID:       testCallerID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.Status != entity.ImportJobStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", job.Status)
	}
}

func TestCancelImportJobCompletedInvalidTransition(t *testing.T) {
	jobID := uuid.New()
	importJobs := &mockImportJobRepo{
		jobs: map[uuid.UUID]*entity.ImportJob{
			jobID: {
				ID:             jobID,
				OrganizationID: testOrgID,
				Status:         entity.ImportJobStatusCompleted,
			},
		},
	}
	memberships := &mockMembershipRepo{
		roles: map[string]string{
			membershipKey(testOrgID, testCallerID): entity.RoleAdmin,
		},
	}
	uc := importjob.NewCancelImportJobUsecase(importJobs, memberships)

	_, err := uc.Execute(context.Background(), importjob.CancelImportJobInput{
		OrganizationID: testOrgID,
		JobID:          jobID,
		CallerID:       testCallerID,
	})
	if !errors.Is(err, importjob.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}
