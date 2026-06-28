package issue_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
	"github.com/google/uuid"
)

type mockIssueRepo struct {
	nextNumber int
	issues     []*entity.Issue
	createErr  error
}

func (m *mockIssueRepo) Create(_ context.Context, issue *entity.Issue) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.issues = append(m.issues, issue)
	return nil
}

func (m *mockIssueRepo) GetByNumber(_ context.Context, _ uuid.UUID, number int) (*entity.Issue, error) {
	for _, issue := range m.issues {
		if issue.Number == number {
			return issue, nil
		}
	}
	return nil, errors.New("issue not found")
}

func (m *mockIssueRepo) ListByRepo(_ context.Context, _ repository.ListIssuesFilter) ([]*entity.Issue, int, error) {
	return m.issues, len(m.issues), nil
}

func (m *mockIssueRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) {
	m.nextNumber++
	return m.nextNumber, nil
}

type mockAuditLogRepo struct {
	calls []auditLogCall
}

type auditLogCall struct {
	orgID      uuid.UUID
	actorID    uuid.UUID
	action     string
	targetType string
	targetID   uuid.UUID
}

func (m *mockAuditLogRepo) InsertAuditLog(
	_ context.Context,
	orgID, actorID uuid.UUID,
	action, targetType string,
	targetID uuid.UUID,
	_ json.RawMessage,
) error {
	m.calls = append(m.calls, auditLogCall{
		orgID:      orgID,
		actorID:    actorID,
		action:     action,
		targetType: targetType,
		targetID:   targetID,
	})
	return nil
}

func (m *mockAuditLogRepo) Create(context.Context, *entity.AuditLog) error {
	return nil
}

func (m *mockAuditLogRepo) List(context.Context, uuid.UUID, string, int, int) ([]*entity.AuditLog, int, error) {
	return nil, 0, nil
}

type mockTxManager struct{}

func (mockTxManager) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestEmptyTitle(t *testing.T) {
	uc := issueusecase.NewCreateIssueUsecase(&mockIssueRepo{}, &mockAuditLogRepo{}, mockTxManager{})

	_, err := uc.Execute(context.Background(), issueusecase.CreateIssueInput{
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		ActorID:        uuid.New(),
		Title:          "",
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestNumberIncrement(t *testing.T) {
	repoID := uuid.New()
	issueRepo := &mockIssueRepo{}
	uc := issueusecase.NewCreateIssueUsecase(issueRepo, &mockAuditLogRepo{}, mockTxManager{})

	first, err := uc.Execute(context.Background(), issueusecase.CreateIssueInput{
		OrganizationID: uuid.New(),
		RepositoryID:   repoID,
		ActorID:        uuid.New(),
		Title:          "First issue",
	})
	if err != nil {
		t.Fatalf("create first issue: %v", err)
	}
	if first.Number != 1 {
		t.Fatalf("expected first issue number 1, got %d", first.Number)
	}

	second, err := uc.Execute(context.Background(), issueusecase.CreateIssueInput{
		OrganizationID: uuid.New(),
		RepositoryID:   repoID,
		ActorID:        uuid.New(),
		Title:          "Second issue",
	})
	if err != nil {
		t.Fatalf("create second issue: %v", err)
	}
	if second.Number != 2 {
		t.Fatalf("expected second issue number 2, got %d", second.Number)
	}
}

func TestAuditLogCalled(t *testing.T) {
	auditRepo := &mockAuditLogRepo{}
	orgID := uuid.New()
	actorID := uuid.New()
	uc := issueusecase.NewCreateIssueUsecase(&mockIssueRepo{}, auditRepo, mockTxManager{})

	issue, err := uc.Execute(context.Background(), issueusecase.CreateIssueInput{
		OrganizationID: orgID,
		RepositoryID:   uuid.New(),
		ActorID:        actorID,
		Title:          "Audit me",
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if len(auditRepo.calls) != 1 {
		t.Fatalf("expected 1 audit log call, got %d", len(auditRepo.calls))
	}

	call := auditRepo.calls[0]
	if call.orgID != orgID || call.actorID != actorID {
		t.Fatalf("unexpected audit actor/org")
	}
	if call.action != "issue.create" || call.targetType != "issue" || call.targetID != issue.ID {
		t.Fatalf("unexpected audit payload: %+v", call)
	}
}

func TestUniqueConflictRetry(t *testing.T) {
	repoID := uuid.New()
	issueRepo := &mockIssueRepo{
		createErr: errors.New("unique constraint violation (23505)"),
	}
	uc := issueusecase.NewCreateIssueUsecase(issueRepo, &mockAuditLogRepo{}, mockTxManager{})

	_, err := uc.Execute(context.Background(), issueusecase.CreateIssueInput{
		OrganizationID: uuid.New(),
		RepositoryID:   repoID,
		ActorID:        uuid.New(),
		Title:          "Retry issue",
	})
	if err == nil {
		t.Fatal("expected retry exhaustion error")
	}
	if issueRepo.nextNumber <= 1 {
		t.Fatalf("expected NextNumber to be retried, got %d", issueRepo.nextNumber)
	}
}
