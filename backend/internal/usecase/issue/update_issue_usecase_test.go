package issue_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
)

type updateIssueRepo struct {
	issue *entity.Issue
}

func (m *updateIssueRepo) Create(_ context.Context, _ *entity.Issue) error {
	return nil
}

func (m *updateIssueRepo) GetByNumber(_ context.Context, _ uuid.UUID, _ int) (*entity.Issue, error) {
	return m.issue, nil
}

func (m *updateIssueRepo) ListByRepo(_ context.Context, _ repository.ListIssuesFilter) ([]*entity.Issue, int, error) {
	return nil, 0, nil
}

func (m *updateIssueRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *updateIssueRepo) Update(_ context.Context, issue *entity.Issue) error {
	m.issue = issue
	return nil
}

type updateLabelRepo struct {
	labels map[string]*entity.Label
}

func (m *updateLabelRepo) GetByName(_ context.Context, _ uuid.UUID, name string) (*entity.Label, error) {
	if m.labels == nil {
		return nil, nil
	}
	return m.labels[name], nil
}

type updateMilestoneRepo struct {
	milestones map[int]*entity.Milestone
}

func (m *updateMilestoneRepo) GetByNumber(_ context.Context, _ uuid.UUID, number int) (*entity.Milestone, error) {
	if m.milestones == nil {
		return nil, nil
	}
	return m.milestones[number], nil
}

type updateAuditLogRepo struct {
	calls []auditLogCall
}

func (m *updateAuditLogRepo) InsertAuditLog(
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

func TestUpdateIssueTitleTooLong(t *testing.T) {
	repoID := uuid.New()
	issue := &entity.Issue{
		ID:           uuid.New(),
		RepositoryID: repoID,
		Number:       1,
		Title:        "Original",
		State:        "open",
	}
	longTitle := strings.Repeat("a", 257)

	uc := issueusecase.NewUpdateIssueUsecase(
		&updateIssueRepo{issue: issue},
		&updateLabelRepo{},
		&updateMilestoneRepo{},
		&updateAuditLogRepo{},
	)

	_, err := uc.Execute(context.Background(), issueusecase.UpdateIssueInput{
		OrganizationID: uuid.New(),
		RepositoryID:   repoID,
		IssueNumber:    1,
		ActorID:        uuid.New(),
		Title:          &longTitle,
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestUpdateIssueUnknownLabel(t *testing.T) {
	repoID := uuid.New()
	issue := &entity.Issue{
		ID:           uuid.New(),
		RepositoryID: repoID,
		Number:       1,
		Title:        "Original",
		State:        "open",
	}

	uc := issueusecase.NewUpdateIssueUsecase(
		&updateIssueRepo{issue: issue},
		&updateLabelRepo{labels: map[string]*entity.Label{}},
		&updateMilestoneRepo{},
		&updateAuditLogRepo{},
	)

	_, err := uc.Execute(context.Background(), issueusecase.UpdateIssueInput{
		OrganizationID: uuid.New(),
		RepositoryID:   repoID,
		IssueNumber:    1,
		ActorID:        uuid.New(),
		LabelNames:     []string{"missing"},
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestUpdateIssueSuccess(t *testing.T) {
	repoID := uuid.New()
	orgID := uuid.New()
	actorID := uuid.New()
	issue := &entity.Issue{
		ID:           uuid.New(),
		RepositoryID: repoID,
		Number:       1,
		Title:        "Original",
		State:        "open",
	}
	issueRepo := &updateIssueRepo{issue: issue}
	auditRepo := &updateAuditLogRepo{}
	newTitle := "Updated title"

	uc := issueusecase.NewUpdateIssueUsecase(
		issueRepo,
		&updateLabelRepo{},
		&updateMilestoneRepo{},
		auditRepo,
	)

	got, err := uc.Execute(context.Background(), issueusecase.UpdateIssueInput{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		IssueNumber:    1,
		ActorID:        actorID,
		Title:          &newTitle,
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if got.Title != newTitle {
		t.Fatalf("expected title %q, got %q", newTitle, got.Title)
	}
	if len(auditRepo.calls) != 1 {
		t.Fatalf("expected 1 audit log call, got %d", len(auditRepo.calls))
	}
	call := auditRepo.calls[0]
	if call.action != "issue.update" || call.targetType != "issue" || call.targetID != issue.ID {
		t.Fatalf("unexpected audit payload: %+v", call)
	}
}

func TestUpdateIssueCloseSetsClosedAt(t *testing.T) {
	repoID := uuid.New()
	issue := &entity.Issue{
		ID:           uuid.New(),
		RepositoryID: repoID,
		Number:       1,
		Title:        "Open issue",
		State:        "open",
	}
	issueRepo := &updateIssueRepo{issue: issue}
	closed := "closed"

	uc := issueusecase.NewUpdateIssueUsecase(
		issueRepo,
		&updateLabelRepo{},
		&updateMilestoneRepo{},
		&updateAuditLogRepo{},
	)

	got, err := uc.Execute(context.Background(), issueusecase.UpdateIssueInput{
		OrganizationID: uuid.New(),
		RepositoryID:   repoID,
		IssueNumber:    1,
		ActorID:        uuid.New(),
		State:          &closed,
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if got.ClosedAt == nil {
		t.Fatal("expected ClosedAt to be set when closing issue")
	}
	if got.State != "closed" {
		t.Fatalf("expected state closed, got %q", got.State)
	}
}
