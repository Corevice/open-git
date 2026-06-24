package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/middleware"
	prusecase "github.com/open-git/backend/internal/usecase/pr"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type PullRequestHandler struct {
	createPRUC  *prusecase.CreatePRUsecase
	mergePRUC   *prusecase.MergePRUsecase
	prRepo      repository.IPullRequestRepository
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

func NewPullRequestHandler(
	createPRUC *prusecase.CreatePRUsecase,
	mergePRUC *prusecase.MergePRUsecase,
	prRepo repository.IPullRequestRepository,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *PullRequestHandler {
	return &PullRequestHandler{
		createPRUC:  createPRUC,
		mergePRUC:   mergePRUC,
		prRepo:      prRepo,
		resolveRepo: resolveRepo,
	}
}

func (h *PullRequestHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	repoScope := middleware.RequireScope("repo")
	g.GET("/repos/:owner/:repo/pulls", h.ListPullRequests, auth, repoScope)
	g.POST("/repos/:owner/:repo/pulls", h.CreatePullRequest, auth, repoScope)
	g.GET("/repos/:owner/:repo/pulls/:number", h.GetPullRequest, auth, repoScope)
	g.POST("/repos/:owner/:repo/pulls/:number/merge", h.MergePullRequest, auth, repoScope)
}

type createPullRequestRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
}

type mergePullRequestRequest struct {
	MergeMethod string `json:"merge_method"`
}

type pullRequestResponse struct {
	ID       uuid.UUID `json:"id"`
	Number   int       `json:"number"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	HeadRef  string    `json:"head_ref"`
	BaseRef  string    `json:"base_ref"`
	State    string    `json:"state"`
	MergedAt *string   `json:"merged_at"`
}

type mergePullRequestResponse struct {
	Merged  bool   `json:"merged"`
	Message string `json:"message"`
}

func (h *PullRequestHandler) ListPullRequests(c echo.Context) error {
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
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	pulls, total, err := h.prRepo.ListByRepo(c.Request().Context(), repository.ListPullRequestsFilter{
		OrganizationID: actor.OrganizationID,
		RepositoryID:   repo.ID,
		State:          c.QueryParam("state"),
		Page:           page,
		PerPage:        perPage,
	})
	if err != nil {
		return err
	}

	setPaginationHeaders(c, page, perPage, total)
	return c.JSON(http.StatusOK, toPullRequestResponses(pulls))
}

func (h *PullRequestHandler) CreatePullRequest(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	var req createPullRequestRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	pr, err := h.createPRUC.Execute(c.Request().Context(), prusecase.CreatePRInput{
		OrganizationID: actor.OrganizationID,
		RepositoryID:   repo.ID,
		ActorID:        actor.UserID,
		Title:          req.Title,
		Body:           req.Body,
		HeadRef:        req.Head,
		BaseRef:        req.Base,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return err
	}

	return c.JSON(http.StatusCreated, toPullRequestResponse(pr))
}

func (h *PullRequestHandler) GetPullRequest(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pull request number")
	}

	pr, err := h.prRepo.GetByNumber(c.Request().Context(), repo.ID, number)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, toPullRequestResponse(pr))
}

func (h *PullRequestHandler) MergePullRequest(c echo.Context) error {
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
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pull request number")
	}

	var req mergePullRequestRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.MergeMethod != "" && req.MergeMethod != "merge" && req.MergeMethod != "squash" && req.MergeMethod != "rebase" {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid merge_method")
	}

	_, err = h.mergePRUC.Execute(c.Request().Context(), prusecase.MergePRInput{
		OrganizationID: actor.OrganizationID,
		RepositoryID:   repo.ID,
		ActorID:        actor.UserID,
		Number:         number,
		MergeMethod:    req.MergeMethod,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrAlreadyMerged) || errors.Is(err, apperror.ErrProtectionNotSatisfied) {
			return echo.NewHTTPError(http.StatusMethodNotAllowed, err.Error())
		}
		if errors.Is(err, apperror.ErrConflict) {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return err
	}

	return c.JSON(http.StatusOK, mergePullRequestResponse{
		Merged:  true,
		Message: "Pull Request successfully merged",
	})
}

func toPullRequestResponse(pr *entity.PullRequest) pullRequestResponse {
	resp := pullRequestResponse{
		ID:      pr.ID,
		Number:  pr.Number,
		Title:   pr.Title,
		Body:    pr.Body,
		HeadRef: pr.HeadRef,
		BaseRef: pr.BaseRef,
		State:   pr.State,
	}
	if pr.MergedAt != nil {
		formatted := pr.MergedAt.UTC().Format("2006-01-02T15:04:05Z")
		resp.MergedAt = &formatted
	}
	return resp
}

func toPullRequestResponses(pulls []*entity.PullRequest) []pullRequestResponse {
	result := make([]pullRequestResponse, 0, len(pulls))
	for _, pr := range pulls {
		result = append(result, toPullRequestResponse(pr))
	}
	return result
}
