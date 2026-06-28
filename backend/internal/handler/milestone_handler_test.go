package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
)

type mockListMilestonesUC struct {
	output *handler.ListMilestonesOutput
}

func (m *mockListMilestonesUC) Execute(_ context.Context, input handler.ListMilestonesInput) (*handler.ListMilestonesOutput, error) {
	if input.State != "" && m.output != nil {
		filtered := make([]*handler.MilestoneDTO, 0)
		for _, milestone := range m.output.Milestones {
			if milestone.State == input.State {
				filtered = append(filtered, milestone)
			}
		}
		return &handler.ListMilestonesOutput{
			Milestones: filtered,
			Total:      len(filtered),
			Page:       m.output.Page,
			PerPage:    m.output.PerPage,
		}, nil
	}
	return m.output, nil
}

type mockCreateMilestoneUC struct {
	milestone *handler.MilestoneDTO
}

func (m *mockCreateMilestoneUC) Execute(_ context.Context, _ handler.CreateMilestoneInput) (*handler.MilestoneDTO, error) {
	return m.milestone, nil
}

type mockUpdateMilestoneUC struct{}

func (m *mockUpdateMilestoneUC) Execute(_ context.Context, _ handler.UpdateMilestoneInput) (*handler.MilestoneDTO, error) {
	return nil, nil
}

type mockDeleteMilestoneUC struct{}

func (m *mockDeleteMilestoneUC) Execute(_ context.Context, _ handler.DeleteMilestoneInput) error {
	return nil
}

func newMilestoneHandlerEcho(t *testing.T, list *mockListMilestonesUC, create *mockCreateMilestoneUC) *echo.Echo {
	t.Helper()

	repoID := uuid.New()
	orgID := uuid.New()

	e := echo.New()
	h := handler.NewMilestoneHandler(
		list,
		create,
		&mockUpdateMilestoneUC{},
		&mockDeleteMilestoneUC{},
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

func TestMilestoneHandlerListMilestones(t *testing.T) {
	milestoneID := uuid.New()
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	e := newMilestoneHandlerEcho(t,
		&mockListMilestonesUC{
			output: &handler.ListMilestonesOutput{
				Milestones: []*handler.MilestoneDTO{
					{
						ID:           milestoneID,
						Number:       1,
						Title:        "v1.0",
						Description:  "First release",
						State:        "open",
						OpenIssues:   3,
						ClosedIssues: 2,
						CreatedAt:    created,
						UpdatedAt:    created,
					},
				},
				Total:   1,
				Page:    1,
				PerPage: 30,
			},
		},
		&mockCreateMilestoneUC{},
	)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/milestones?state=open", nil)
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
	if resp[0]["open_issues"] != float64(3) {
		t.Fatalf("open_issues = %v, want 3", resp[0]["open_issues"])
	}
}

func TestMilestoneHandlerCreateMilestone(t *testing.T) {
	milestoneID := uuid.New()
	created := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	e := newMilestoneHandlerEcho(t,
		&mockListMilestonesUC{output: &handler.ListMilestonesOutput{}},
		&mockCreateMilestoneUC{
			milestone: &handler.MilestoneDTO{
				ID:           milestoneID,
				Number:       2,
				Title:        "v2.0",
				Description:  "Second release",
				State:        "open",
				OpenIssues:   0,
				ClosedIssues: 0,
				CreatedAt:    created,
				UpdatedAt:    created,
			},
		},
	)

	body := bytes.NewBufferString(`{"title":"v2.0","description":"Second release","state":"open"}`)
	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/milestones", body)
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
	if resp["number"] != float64(2) {
		t.Fatalf("number = %v, want 2", resp["number"])
	}
}

func TestMilestoneHandlerDeleteMilestone(t *testing.T) {
	e := newMilestoneHandlerEcho(t,
		&mockListMilestonesUC{output: &handler.ListMilestonesOutput{}},
		&mockCreateMilestoneUC{},
	)

	req := httptest.NewRequest(http.MethodDelete, "/repos/alice/demo/milestones/1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}
