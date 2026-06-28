package label_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	labelusecase "github.com/open-git/backend/internal/usecase/label"
)

type mockLabelRepo struct {
	labels        map[string]*entity.Label
	createErr     error
	addToIssueErr error
}

func (m *mockLabelRepo) Create(_ context.Context, label *entity.Label) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.labels == nil {
		m.labels = make(map[string]*entity.Label)
	}
	m.labels[label.Name] = label
	return nil
}

func (m *mockLabelRepo) ListByRepo(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.Label, int, error) {
	return nil, 0, nil
}

func (m *mockLabelRepo) GetByName(_ context.Context, _ uuid.UUID, name string) (*entity.Label, error) {
	if m.labels == nil {
		return nil, nil
	}
	return m.labels[name], nil
}

func (m *mockLabelRepo) Update(_ context.Context, _ *entity.Label) error {
	return nil
}

func (m *mockLabelRepo) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockLabelRepo) AddToIssue(_ context.Context, _ uuid.UUID, _ int, _ uuid.UUID) error {
	return m.addToIssueErr
}

func (m *mockLabelRepo) RemoveFromIssue(_ context.Context, _ uuid.UUID, _ int, _ uuid.UUID) error {
	return nil
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

func TestCreateLabelInvalidColor(t *testing.T) {
	uc := labelusecase.NewCreateLabelUsecase(&mockLabelRepo{})

	_, err := uc.Execute(context.Background(), labelusecase.CreateLabelInput{
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		Name:           "bug",
		Color:          "gg0000",
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestCreateLabelDuplicateName(t *testing.T) {
	repo := &mockLabelRepo{createErr: apperror.ErrConflict}
	uc := labelusecase.NewCreateLabelUsecase(repo)

	_, err := uc.Execute(context.Background(), labelusecase.CreateLabelInput{
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		Name:           "bug",
		Color:          "ff0000",
	})
	if !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestCreateLabelSuccess(t *testing.T) {
	repo := &mockLabelRepo{}
	uc := labelusecase.NewCreateLabelUsecase(repo)

	label, err := uc.Execute(context.Background(), labelusecase.CreateLabelInput{
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		Name:           "bug",
		Color:          "ff0000",
		Description:    "Bug reports",
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	if label.Name != "bug" || label.Color != "ff0000" {
		t.Fatalf("unexpected label: %+v", label)
	}
	if repo.labels["bug"] == nil {
		t.Fatal("expected label stored in repo")
	}
}

func TestDeleteLabelNotFound(t *testing.T) {
	uc := labelusecase.NewDeleteLabelUsecase(&mockLabelRepo{}, &mockAuditLogRepo{})

	err := uc.Execute(context.Background(), labelusecase.DeleteLabelInput{
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		ActorID:        uuid.New(),
		Name:           "missing",
	})
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteLabelSuccessWithAudit(t *testing.T) {
	labelID := uuid.New()
	orgID := uuid.New()
	actorID := uuid.New()
	repoID := uuid.New()
	repo := &mockLabelRepo{
		labels: map[string]*entity.Label{
			"bug": {
				ID:             labelID,
				OrganizationID: orgID,
				RepositoryID:   repoID,
				Name:           "bug",
				Color:          "ff0000",
			},
		},
	}
	auditRepo := &mockAuditLogRepo{}
	uc := labelusecase.NewDeleteLabelUsecase(repo, auditRepo)

	err := uc.Execute(context.Background(), labelusecase.DeleteLabelInput{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		ActorID:        actorID,
		Name:           "bug",
	})
	if err != nil {
		t.Fatalf("delete label: %v", err)
	}
	if len(auditRepo.calls) != 1 {
		t.Fatalf("expected 1 audit log call, got %d", len(auditRepo.calls))
	}
	call := auditRepo.calls[0]
	if call.orgID != orgID || call.actorID != actorID {
		t.Fatalf("unexpected audit actor/org")
	}
	if call.action != "label.delete" || call.targetType != "label" || call.targetID != labelID {
		t.Fatalf("unexpected audit payload: %+v", call)
	}
}

func TestAddIssueLabelsUnknownName(t *testing.T) {
	uc := labelusecase.NewAddIssueLabelsUsecase(&mockLabelRepo{})

	err := uc.Execute(context.Background(), labelusecase.AddIssueLabelsInput{
		RepositoryID: uuid.New(),
		IssueNumber:  1,
		Names:        []string{"missing"},
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}
