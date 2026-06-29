package pr_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	prusecase "github.com/open-git/backend/internal/usecase/pr"
)

type recordingPRMergeableEnqueuer struct {
	calls []prusecase.PRMergeableEnqueuePayload
	err   error
}

func (m *recordingPRMergeableEnqueuer) Enqueue(_ context.Context, payload prusecase.PRMergeableEnqueuePayload) error {
	m.calls = append(m.calls, payload)
	return m.err
}

func TestCreatePREnqueuesMergeableCheck(t *testing.T) {
	enqueuer := &recordingPRMergeableEnqueuer{}
	uc := prusecase.NewCreatePRUsecase(
		&mockPullRequestRepo{},
		&mockAuditLogRepo{},
		&mockGitService{},
		mockTxManager{},
		&mockMembershipRepo{},
		enqueuer,
	)

	orgID := uuid.New()
	repoID := uuid.New()
	actorID := uuid.New()

	pr, err := uc.Execute(context.Background(), prusecase.CreatePRInput{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		GitPath:        "/tmp/repo.git",
		ActorID:        actorID,
		Title:          "Test PR",
		Body:           "body",
		HeadRef:        "feature",
		BaseRef:        "main",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if pr == nil {
		t.Fatal("Execute() returned nil PR")
	}
	if len(enqueuer.calls) != 1 {
		t.Fatalf("expected 1 enqueue call, got %d", len(enqueuer.calls))
	}

	call := enqueuer.calls[0]
	if call.GitPath != "/tmp/repo.git" {
		t.Errorf("GitPath = %q, want %q", call.GitPath, "/tmp/repo.git")
	}
	if call.HeadRef != "feature" {
		t.Errorf("HeadRef = %q, want %q", call.HeadRef, "feature")
	}
	if call.BaseRef != "main" {
		t.Errorf("BaseRef = %q, want %q", call.BaseRef, "main")
	}
	if call.PRID != pr.ID {
		t.Errorf("PRID = %v, want %v", call.PRID, pr.ID)
	}
}

func TestCreatePREnqueueFailureNonFatal(t *testing.T) {
	enqueuer := &recordingPRMergeableEnqueuer{err: errors.New("redis unavailable")}
	uc := prusecase.NewCreatePRUsecase(
		&mockPullRequestRepo{},
		&mockAuditLogRepo{},
		&mockGitService{},
		mockTxManager{},
		&mockMembershipRepo{},
		enqueuer,
	)

	pr, err := uc.Execute(context.Background(), prusecase.CreatePRInput{
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		GitPath:        "/tmp/repo.git",
		ActorID:        uuid.New(),
		Title:          "Test PR",
		HeadRef:        "feature",
		BaseRef:        "main",
	})
	if err != nil {
		t.Fatalf("Execute() should not fail on enqueue error, got %v", err)
	}
	if pr == nil {
		t.Fatal("Execute() returned nil PR")
	}
	if len(enqueuer.calls) != 1 {
		t.Fatalf("expected 1 enqueue call, got %d", len(enqueuer.calls))
	}
}
