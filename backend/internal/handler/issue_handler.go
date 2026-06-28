package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type IssueHandler struct {
	createIssueUC    *issueusecase.CreateIssueUsecase
	listIssuesUC     *issueusecase.ListIssuesUsecase
	createCommentUC  *issueusecase.CreateCommentUsecase
	updateIssueUC    *issueusecase.UpdateIssueUsecase
	getIssueUC       *issueusecase.GetIssueUsecase
	listCommentsUC   *issueusecase.ListCommentsUsecase
	updateCommentUC  *issueusecase.UpdateCommentUsecase
	deleteCommentUC  *issueusecase.DeleteCommentUsecase
	resolveRepo      func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

func NewIssueHandler(
	createIssueUC *issueusecase.CreateIssueUsecase,
	listIssuesUC *issueusecase.ListIssuesUsecase,
	createCommentUC *issueusecase.CreateCommentUsecase,
	updateIssueUC *issueusecase.UpdateIssueUsecase,
	getIssueUC *issueusecase.GetIssueUsecase,
	listCommentsUC *issueusecase.ListCommentsUsecase,
	updateCommentUC *issueusecase.UpdateCommentUsecase,
	deleteCommentUC *issueusecase.DeleteCommentUsecase,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *IssueHandler {
	return &IssueHandler{
		createIssueUC:   createIssueUC,
		listIssuesUC:    listIssuesUC,
		createCommentUC: createCommentUC,
		updateIssueUC:   updateIssueUC,
		getIssueUC:      getIssueUC,
		listCommentsUC:  listCommentsUC,
		updateCommentUC: updateCommentUC,
		deleteCommentUC: deleteCommentUC,
		resolveRepo:     resolveRepo,
	}
}

func (h *IssueHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	repoScope := middleware.RequireScope("repo")
	g.GET("/repos/:owner/:repo/issues", h.ListIssues, auth, repoScope)
	g.POST("/repos/:owner/:repo/issues", h.CreateIssue, auth, repoScope)
	g.GET("/repos/:owner/:repo/issues/:number", h.GetIssue, auth, repoScope)
	g.PATCH("/repos/:owner/:repo/issues/:number", h.UpdateIssue, auth, repoScope)
	g.GET("/repos/:owner/:repo/issues/:number/comments", h.ListComments, auth, repoScope)
	g.POST("/repos/:owner/:repo/issues/:number/comments", h.CreateComment, auth, repoScope)
	g.PATCH("/repos/:owner/:repo/issues/:number/comments/:comment_id", h.UpdateComment, auth, repoScope)
	g.DELETE("/repos/:owner/:repo/issues/:number/comments/:comment_id", h.DeleteComment, auth, repoScope)
}

type createIssueRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type createCommentRequest struct {
	Body string `json:"body"`
}

type updateIssueRequest struct {
	State string  `json:"state"`
	Title *string `json:"title"`
	Body  *string `json:"body"`
}

type updateCommentRequest struct {
	Body string `json:"body"`
}

func (h *IssueHandler) ListIssues(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	output, err := h.listIssuesUC.Execute(c.Request().Context(), issueusecase.ListIssuesInput{
		OrganizationID: actor.OrganizationID,
		RepositoryID:   repo.ID,
		State:          c.QueryParam("state"),
		Labels:         splitLabels(c.QueryParam("labels")),
		Page:           page,
		PerPage:        perPage,
	})
	if err != nil {
		return err
	}

	setPaginationHeaders(c, output.Page, output.PerPage, output.Total)
	return c.JSON(http.StatusOK, toIssueResponses(output.Issues, c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *IssueHandler) CreateIssue(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	var req createIssueRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	issue, err := h.createIssueUC.Execute(c.Request().Context(), issueusecase.CreateIssueInput{
		OrganizationID: actor.OrganizationID,
		RepositoryID:   repo.ID,
		ActorID:        actor.UserID,
		Title:          req.Title,
		Body:           req.Body,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return err
	}

	return c.JSON(http.StatusCreated, toIssueResponse(issue, c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *IssueHandler) UpdateIssue(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid issue number")
	}

	var req updateIssueRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.State != "" && req.State != "open" && req.State != "closed" {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid state value")
	}

	var statePtr *string
	if req.State != "" {
		statePtr = &req.State
	}

	issue, err := h.updateIssueUC.Execute(c.Request().Context(), issueusecase.UpdateIssueInput{
		OrganizationID: actor.OrganizationID,
		RepositoryID:   repo.ID,
		IssueNumber:    number,
		ActorID:        actor.UserID,
		Title:          req.Title,
		Body:           req.Body,
		State:          statePtr,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return err
	}

	return c.JSON(http.StatusOK, toIssueResponse(issue, c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *IssueHandler) GetIssue(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid issue number")
	}

	issue, err := h.getIssueUC.Execute(c.Request().Context(), issueusecase.GetIssueInput{
		RepositoryID: repo.ID,
		IssueNumber:  number,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.JSON(http.StatusOK, toIssueResponse(issue, c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *IssueHandler) ListComments(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid issue number")
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	output, err := h.listCommentsUC.Execute(c.Request().Context(), issueusecase.ListCommentsInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		IssueNumber:    number,
		Page:           page,
		PerPage:        perPage,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	setPaginationHeaders(c, output.Page, output.PerPage, output.Total)
	return c.JSON(http.StatusOK, toCommentResponses(output.Comments, c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *IssueHandler) CreateComment(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid issue number")
	}

	var req createCommentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	comment, err := h.createCommentUC.Execute(c.Request().Context(), issueusecase.CreateCommentInput{
		OrganizationID: actor.OrganizationID,
		RepositoryID:   repo.ID,
		IssueNumber:    number,
		ActorID:        actor.UserID,
		Body:           req.Body,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrGone) {
			return echo.NewHTTPError(http.StatusGone, err.Error())
		}
		return err
	}

	return c.JSON(http.StatusCreated, toCommentResponse(comment, c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *IssueHandler) UpdateComment(c echo.Context) error {
	_, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	commentID, err := strconv.ParseInt(c.Param("comment_id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid comment id")
	}

	var req updateCommentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	comment, err := h.updateCommentUC.Execute(c.Request().Context(), issueusecase.UpdateCommentInput{
		CommentID:      middleware.Int64ToUUID(commentID),
		OrganizationID: actor.OrganizationID,
		ActorID:        actor.UserID,
		Body:           req.Body,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		if errors.Is(err, domain.ErrForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
		}
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return err
	}

	return c.JSON(http.StatusOK, toCommentResponse(comment, c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *IssueHandler) DeleteComment(c echo.Context) error {
	_, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	commentID, err := strconv.ParseInt(c.Param("comment_id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid comment id")
	}

	err = h.deleteCommentUC.Execute(c.Request().Context(), issueusecase.DeleteCommentInput{
		CommentID:      middleware.Int64ToUUID(commentID),
		OrganizationID: actor.OrganizationID,
		ActorID:        actor.UserID,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		if errors.Is(err, domain.ErrForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
		}
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func splitLabels(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		label := strings.TrimSpace(part)
		if label != "" {
			labels = append(labels, label)
		}
	}
	return labels
}

func setPaginationHeaders(c echo.Context, page, perPage, total int) {
	if perPage <= 0 {
		return
	}
	lastPage := (total + perPage - 1) / perPage
	if lastPage < 1 {
		lastPage = 1
	}

	baseURL := c.Request().URL
	query := baseURL.Query()
	links := make([]string, 0, 4)

	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(perPage))
	baseURL.RawQuery = query.Encode()
	links = append(links, `<`+baseURL.String()+`>; rel="self"`)

	if page > 1 {
		query.Set("page", "1")
		baseURL.RawQuery = query.Encode()
		links = append(links, `<`+baseURL.String()+`>; rel="first"`)
		query.Set("page", strconv.Itoa(page-1))
		baseURL.RawQuery = query.Encode()
		links = append(links, `<`+baseURL.String()+`>; rel="prev"`)
	}
	if page < lastPage {
		query.Set("page", strconv.Itoa(page+1))
		baseURL.RawQuery = query.Encode()
		links = append(links, `<`+baseURL.String()+`>; rel="next"`)
		query.Set("page", strconv.Itoa(lastPage))
		baseURL.RawQuery = query.Encode()
		links = append(links, `<`+baseURL.String()+`>; rel="last"`)
	}

	c.Response().Header().Set("Link", strings.Join(links, ", "))
}

type issueUserResponse struct {
	Login string `json:"login"`
}

type issueResponse struct {
	ID      uuid.UUID         `json:"id"`
	Number  int               `json:"number"`
	Title   string            `json:"title"`
	Body    string            `json:"body"`
	State   string            `json:"state"`
	NodeID  string            `json:"node_id"`
	HTMLURL string            `json:"html_url"`
	User    issueUserResponse `json:"user"`
}

type commentResponse struct {
	ID        int64             `json:"id"`
	NodeID    string            `json:"node_id"`
	Body      string            `json:"body"`
	User      issueUserResponse `json:"user"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	URL       string            `json:"url"`
}

func toIssueResponse(issue *entity.Issue, owner, repoName, host string) issueResponse {
	return issueResponse{
		ID:      issue.ID,
		Number:  issue.Number,
		Title:   issue.Title,
		Body:    issue.Body,
		State:   issue.State,
		NodeID:  IssueNodeID(issue.ID),
		HTMLURL: "https://" + host + "/" + owner + "/" + repoName + "/issues/" + strconv.Itoa(issue.Number),
		User: issueUserResponse{
			Login: issue.AuthorLogin,
		},
	}
}

func toIssueResponses(issues []*entity.Issue, owner, repoName, host string) []issueResponse {
	result := make([]issueResponse, 0, len(issues))
	for _, issue := range issues {
		result = append(result, toIssueResponse(issue, owner, repoName, host))
	}
	return result
}

func CommentNodeID(id uuid.UUID) string { return NodeID("IssueComment", id.String()) }

func toCommentResponse(comment *entity.Comment, owner, repoName, host string) commentResponse {
	commentID := middleware.UUIDToInt64(comment.ID)
	return commentResponse{
		ID:        commentID,
		NodeID:    CommentNodeID(comment.ID),
		Body:      comment.Body,
		User:      issueUserResponse{Login: comment.AuthorLogin},
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
		URL:       "https://" + host + "/repos/" + owner + "/" + repoName + "/issues/comments/" + strconv.FormatInt(commentID, 10),
	}
}

func toCommentResponses(comments []*entity.Comment, owner, repoName, host string) []commentResponse {
	result := make([]commentResponse, 0, len(comments))
	for _, comment := range comments {
		result = append(result, toCommentResponse(comment, owner, repoName, host))
	}
	return result
}
