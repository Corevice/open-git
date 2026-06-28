package milestone_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	milestoneusecase "github.com/open-git/backend/internal/usecase/milestone"
)

type mockMilestoneRepo struct {
	nextNumber int
	milestones map[int]*entity.Milestone
}

func (m *mockMilestoneRepo) Create(_ context.Context, milestone *entity.Milestone) error {
	if m.milestones == nil {
		m.milestones = make(map[int]*entity.Milestone)
	}
	m.milestones[milestone.Number] = milestone
	return nil
}

func (m *mockMilestoneRepo) ListByRepo(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*entity.Milestone, int, error) {
	return nil, 0, nil
}

func (m *mockMilestoneRepo) GetByNumber(_ context.Context, _ uuid.UUID, number int) (*entity.Milestone, error) {
	if m.milestones == nil {
		return nil, nil
	}
	return m.milestones[number], nil
}

func (m *mockMilestoneRepo) Update(_ context.Context, milestone *entity.Milestone) error {
	if m.milestones == nil {
		m.milestones = make(map[int]*entity.Milestone)
	}
	m.milestones[milestone.Number] = milestone
	return nil
}

func (m *mockMilestoneRepo) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (m *mockMilestoneRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) {
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

func TestCreateMilestoneNumberAllocation(t *testing.T) {
	repoID := uuid.New()
	milestoneRepo := &mockMilestoneRepo{}
	auditRepo := &mockAuditLogRepo{}
	uc := milestoneusecase.NewCreateMilestoneUsecase(milestoneRepo, auditRepo)

	first, err := uc.Execute(context.Background(), milestoneusecase.CreateMilestoneInput{
		OrganizationID: uuid.New(),
		RepositoryID:   repoID,
		ActorID:        uuid.New(),
		Title:          "v1.0",
	})
	if err != nil {
		t.Fatalf("create first milestone: %v", err)
	}
	if first.Number != 1 {
		t.Fatalf("expected first milestone number 1, got %d", first.Number)
	}

	second, err := uc.Execute(context.Background(), milestoneusecase.CreateMilestoneInput{
		OrganizationID: uuid.New(),
		RepositoryID:   repoID,
		ActorID:        uuid.New(),
		Title:          "v2.0",
	})
	if err != nil {
		t.Fatalf("create second milestone: %v", err)
	}
	if second.Number != 2 {
		t.Fatalf("expected second milestone number 2, got %d", second.Number)
	}
	if milestoneRepo.nextNumber != 2 {
		t.Fatalf("expected NextNumber called twice, got %d", milestoneRepo.nextNumber)
	}
}

func TestUpdateMilestoneClosedSetsClosedAt(t *testing.T) {
	repoID := uuid.New()
	milestoneRepo := &mockMilestoneRepo{
		milestones: map[int]*entity.Milestone{
			1: {
				ID:           uuid.New(),
				RepositoryID: repoID,
				Number:       1,
				Title:        "v1.0",
				State:        "open",
			},
		},
	}
	uc := milestoneusecase.NewUpdateMilestoneUsecase(milestoneRepo)

	closed := "closed"
	updated, err := uc.Execute(context.Background(), milestoneusecase.UpdateMilestoneInput{
		RepositoryID: repoID,
		Number:       1,
		State:        &closed,
	})
	if err != nil {
		t.Fatalf("update milestone: %v", err)
	}
	if updated.State != "closed" {
		t.Fatalf("expected state closed, got %q", updated.State)
	}
	if updated.ClosedAt == nil {
		t.Fatal("expected ClosedAt to be set")
	}
}

func TestDeleteMilestoneNotFound(t *testing.T) {
	uc := milestoneusecase.NewDeleteMilestoneUsecase(&mockMilestoneRepo{}, &mockAuditLogRepo{})

	err := uc.Execute(context.Background(), milestoneusecase.DeleteMilestoneInput{
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		ActorID:        uuid.New(),
		Number:         99,
	})
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteMilestoneSuccessWithAudit(t *testing.T) {
	milestoneID := uuid.New()
	orgID := uuid.New()
	actorID := uuid.New()
	repoID := uuid.New()
	milestoneRepo := &mockMilestoneRepo{
		milestones: map[int]*entity.Milestone{
			1: {
				ID:             milestoneID,
				OrganizationID: orgID,
				RepositoryID:   repoID,
				Number:         1,
				Title:          "v1.0",
				State:          "open",
			},
		},
	}
	auditRepo := &mockAuditLogRepo{}
	uc := milestoneusecase.NewDeleteMilestoneUsecase(milestoneRepo, auditRepo)

	err := uc.Execute(context.Background(), milestoneusecase.DeleteMilestoneInput{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		ActorID:        actorID,
		Number:         1,
	})
	if err != nil {
		t.Fatalf("delete milestone: %v", err)
	}
	if len(auditRepo.calls) != 1 {
		t.Fatalf("expected 1 audit log call, got %d", len(auditRepo.calls))
	}
	call := auditRepo.calls[0]
	if call.orgID != orgID || call.actorID != actorID {
		t.Fatalf("unexpected audit actor/org")
	}
	if call.action != "milestone.delete" || call.targetType != "milestone" || call.targetID != milestoneID {
		t.Fatalf("unexpected audit payload: %+v", call)
	}
}
