package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type CommentHandler struct {
	listCommentsUC  listCommentsUC
	updateCommentUC updateCommentUC
	deleteCommentUC deleteCommentUC
	resolveRepo     func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

type ListCommentsInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	IssueNumber    int
	Page           int
	PerPage        int
}

type ListCommentsOutput struct {
	Comments []*CommentDTO
	Total    int
	Page     int
	PerPage  int
}

type UpdateCommentInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	CommentID      uuid.UUID
	ActorID        uuid.UUID
	Body           string
}

type DeleteCommentInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	CommentID      uuid.UUID
	ActorID        uuid.UUID
}

type CommentDTO struct {
	ID          uuid.UUID
	Body        string
	AuthorLogin string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type listCommentsUC interface {
	Execute(ctx context.Context, input ListCommentsInput) (*ListCommentsOutput, error)
}

type updateCommentUC interface {
	Execute(ctx context.Context, input UpdateCommentInput) (*CommentDTO, error)
}

type deleteCommentUC interface {
	Execute(ctx context.Context, input DeleteCommentInput) error
}

func NewCommentHandler(
	listCommentsUC listCommentsUC,
	updateCommentUC updateCommentUC,
	deleteCommentUC deleteCommentUC,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *CommentHandler {
	return &CommentHandler{
		listCommentsUC:  listCommentsUC,
		updateCommentUC: updateCommentUC,
		deleteCommentUC: deleteCommentUC,
		resolveRepo:     resolveRepo,
	}
}

func (h *CommentHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	repoScope := middleware.RequireScope("repo")
	g.GET("/repos/:owner/:repo/issues/:number/comments", h.ListComments, auth, repoScope)
	g.PATCH("/repos/:owner/:repo/issues/comments/:comment_id", h.UpdateComment, auth, repoScope)
	g.DELETE("/repos/:owner/:repo/issues/comments/:comment_id", h.DeleteComment, auth, repoScope)
}

type issueCommentResponse struct {
	ID   string `json:"id"`
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type updateCommentRequest struct {
	Body string `json:"body"`
}

func (h *CommentHandler) ListComments(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	if _, err := middleware.GetUserUUID(c); err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid issue number")
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	output, err := h.listCommentsUC.Execute(c.Request().Context(), ListCommentsInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		IssueNumber:    number,
		Page:           page,
		PerPage:        perPage,
	})
	if err != nil {
		return err
	}

	setPaginationHeaders(c, output.Page, output.PerPage, output.Total)
	return c.JSON(http.StatusOK, toIssueCommentResponses(output.Comments))
}

func (h *CommentHandler) UpdateComment(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	commentID, err := parseCommentID(c.Param("comment_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid comment id")
	}

	var req updateCommentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	comment, err := h.updateCommentUC.Execute(c.Request().Context(), UpdateCommentInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		CommentID:      commentID,
		ActorID:        actorID,
		Body:           req.Body,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Not Found")
		}
		return err
	}

	return c.JSON(http.StatusOK, toIssueCommentResponse(comment))
}

func (h *CommentHandler) DeleteComment(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	commentID, err := parseCommentID(c.Param("comment_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid comment id")
	}

	err = h.deleteCommentUC.Execute(c.Request().Context(), DeleteCommentInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		CommentID:      commentID,
		ActorID:        actorID,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Not Found")
		}
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func parseCommentID(raw string) (uuid.UUID, error) {
	if id, err := uuid.Parse(raw); err == nil {
		return id, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return uuid.Nil, err
	}
	return middleware.Int64ToUUID(n), nil
}

func toIssueCommentResponse(comment *CommentDTO) issueCommentResponse {
	resp := issueCommentResponse{
		ID:        formatResourceID(comment.ID),
		Body:      comment.Body,
		CreatedAt: formatTimestamp(comment.CreatedAt),
		UpdatedAt: formatTimestamp(comment.UpdatedAt),
	}
	resp.User.Login = comment.AuthorLogin
	return resp
}

func toIssueCommentResponses(comments []*CommentDTO) []issueCommentResponse {
	result := make([]issueCommentResponse, 0, len(comments))
	for _, comment := range comments {
		result = append(result, toIssueCommentResponse(comment))
	}
	return result
}

func formatResourceID(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return strconv.FormatInt(middleware.UUIDToInt64(id), 10)
}

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
