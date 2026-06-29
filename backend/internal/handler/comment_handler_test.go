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

type mockListCommentsUC struct {
	output *handler.ListCommentsOutput
}

func (m *mockListCommentsUC) Execute(_ context.Context, _ handler.ListCommentsInput) (*handler.ListCommentsOutput, error) {
	return m.output, nil
}

type mockUpdateCommentUC struct {
	comment *handler.CommentDTO
}

func (m *mockUpdateCommentUC) Execute(_ context.Context, _ handler.UpdateCommentInput) (*handler.CommentDTO, error) {
	return m.comment, nil
}

type mockDeleteCommentUC struct{}

func (m *mockDeleteCommentUC) Execute(_ context.Context, _ handler.DeleteCommentInput) error {
	return nil
}

func newCommentHandlerEcho(t *testing.T, list *mockListCommentsUC, update *mockUpdateCommentUC, deleteUC *mockDeleteCommentUC) *echo.Echo {
	t.Helper()

	repoID := uuid.New()
	orgID := uuid.New()

	e := echo.New()
	h := handler.NewCommentHandler(
		list,
		update,
		deleteUC,
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

func TestCommentHandlerListComments(t *testing.T) {
	commentID := uuid.New()
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := time.Date(2026, 1, 2, 4, 0, 0, 0, time.UTC)

	e := newCommentHandlerEcho(t,
		&mockListCommentsUC{
			output: &handler.ListCommentsOutput{
				Comments: []*handler.CommentDTO{
					{
						ID:          commentID,
						Body:        "first comment",
						AuthorLogin: "alice",
						CreatedAt:   created,
						UpdatedAt:   updated,
					},
				},
				Total:   1,
				Page:    1,
				PerPage: 30,
			},
		},
		&mockUpdateCommentUC{},
		&mockDeleteCommentUC{},
	)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/issues/1/comments", nil)
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
	user, ok := resp[0]["user"].(map[string]any)
	if !ok || user["login"] != "alice" {
		t.Fatalf("user.login = %v, want alice", resp[0]["user"])
	}
}

func TestCommentHandlerDeleteComment(t *testing.T) {
	e := newCommentHandlerEcho(t,
		&mockListCommentsUC{output: &handler.ListCommentsOutput{}},
		&mockUpdateCommentUC{},
		&mockDeleteCommentUC{},
	)

	req := httptest.NewRequest(http.MethodDelete, "/repos/alice/demo/issues/comments/42", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

func TestCommentHandlerUpdateComment(t *testing.T) {
	commentID := uuid.New()
	updatedAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	e := newCommentHandlerEcho(t,
		&mockListCommentsUC{output: &handler.ListCommentsOutput{}},
		&mockUpdateCommentUC{
			comment: &handler.CommentDTO{
				ID:          commentID,
				Body:        "updated body",
				AuthorLogin: "alice",
				CreatedAt:   updatedAt.Add(-time.Hour),
				UpdatedAt:   updatedAt,
			},
		},
		&mockDeleteCommentUC{},
	)

	body := bytes.NewBufferString(`{"body":"updated body"}`)
	req := httptest.NewRequest(http.MethodPatch, "/repos/alice/demo/issues/comments/42", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["body"] != "updated body" {
		t.Fatalf("body = %v, want updated body", resp["body"])
	}
}
