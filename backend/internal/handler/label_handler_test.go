package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
)

type mockListLabelsUC struct {
	output *handler.ListLabelsOutput
}

func (m *mockListLabelsUC) Execute(_ context.Context, _ handler.ListLabelsInput) (*handler.ListLabelsOutput, error) {
	return m.output, nil
}

type mockCreateLabelUC struct {
	label *handler.LabelDTO
}

func (m *mockCreateLabelUC) Execute(_ context.Context, _ handler.CreateLabelInput) (*handler.LabelDTO, error) {
	return m.label, nil
}

type mockUpdateLabelUC struct{}

func (m *mockUpdateLabelUC) Execute(_ context.Context, _ handler.UpdateLabelInput) (*handler.LabelDTO, error) {
	return nil, nil
}

type mockDeleteLabelUC struct{}

func (m *mockDeleteLabelUC) Execute(_ context.Context, _ handler.DeleteLabelInput) error {
	return nil
}

type mockAddIssueLabelsUC struct{}

func (m *mockAddIssueLabelsUC) Execute(_ context.Context, _ handler.AddIssueLabelsInput) ([]*handler.LabelDTO, error) {
	return nil, nil
}

type mockRemoveIssueLabelUC struct{}

func (m *mockRemoveIssueLabelUC) Execute(_ context.Context, _ handler.RemoveIssueLabelInput) ([]*handler.LabelDTO, error) {
	return nil, nil
}

func newLabelHandlerEcho(t *testing.T, list *mockListLabelsUC, create *mockCreateLabelUC) *echo.Echo {
	t.Helper()

	repoID := uuid.New()
	orgID := uuid.New()

	e := echo.New()
	h := handler.NewLabelHandler(
		list,
		create,
		&mockUpdateLabelUC{},
		&mockDeleteLabelUC{},
		&mockAddIssueLabelsUC{},
		&mockRemoveIssueLabelUC{},
		func(_ echo.Context, _, _ string) (*entity.Repository, error) {
			return &entity.Repository{ID: repoID, OrganizationID: orgID}, nil
		},
	)

	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, 42, []string{"repo"})
			return next(c)
		}
	}

	g := e.Group("")
	h.RegisterRoutes(g, auth)
	return e
}

func TestLabelHandlerListLabels(t *testing.T) {
	labelID := uuid.New()

	e := newLabelHandlerEcho(t,
		&mockListLabelsUC{
			output: &handler.ListLabelsOutput{
				Labels: []*handler.LabelDTO{
					{ID: labelID, Name: "bug", Color: "ff0000", Description: "Bug reports"},
				},
				Total:   1,
				Page:    1,
				PerPage: 30,
			},
		},
		&mockCreateLabelUC{},
	)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/labels", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("len = %d, want 1", len(resp))
	}
	if resp[0]["name"] != "bug" {
		t.Fatalf("name = %v, want bug", resp[0]["name"])
	}
}

func TestLabelHandlerCreateLabelInvalidColor(t *testing.T) {
	e := newLabelHandlerEcho(t,
		&mockListLabelsUC{output: &handler.ListLabelsOutput{}},
		&mockCreateLabelUC{},
	)

	body := bytes.NewBufferString(`{"name":"bug","color":"red","description":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/labels", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	errors, ok := resp["errors"].([]any)
	if !ok || len(errors) == 0 {
		t.Fatalf("errors = %v, want non-empty GitHub-style errors", resp["errors"])
	}
}

func TestLabelHandlerCreateLabelValid(t *testing.T) {
	labelID := uuid.New()

	e := newLabelHandlerEcho(t,
		&mockListLabelsUC{output: &handler.ListLabelsOutput{}},
		&mockCreateLabelUC{
			label: &handler.LabelDTO{
				ID:          labelID,
				Name:        "bug",
				Color:       "ff0000",
				Description: "Bug reports",
			},
		},
	)

	body := bytes.NewBufferString(`{"name":"bug","color":"ff0000","description":"Bug reports"}`)
	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/labels", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["name"] != "bug" {
		t.Fatalf("name = %v, want bug", resp["name"])
	}
}

func TestLabelHandlerDeleteLabel(t *testing.T) {
	e := newLabelHandlerEcho(t,
		&mockListLabelsUC{output: &handler.ListLabelsOutput{}},
		&mockCreateLabelUC{},
	)

	req := httptest.NewRequest(http.MethodDelete, "/repos/alice/demo/labels/bug", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}
