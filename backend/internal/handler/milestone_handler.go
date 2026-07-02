package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	milestoneusecase "github.com/open-git/backend/internal/usecase/milestone"
)

type MilestoneHandler struct {
	listMilestonesUC  *milestoneusecase.ListMilestonesUsecase
	createMilestoneUC *milestoneusecase.CreateMilestoneUsecase
	updateMilestoneUC *milestoneusecase.UpdateMilestoneUsecase
	deleteMilestoneUC *milestoneusecase.DeleteMilestoneUsecase
	resolveRepo       func(c echo.Context, owner, repo string) (*entity.Repository, error)
	access *RepoAccess
}

func NewMilestoneHandler(
	listMilestonesUC *milestoneusecase.ListMilestonesUsecase,
	createMilestoneUC *milestoneusecase.CreateMilestoneUsecase,
	updateMilestoneUC *milestoneusecase.UpdateMilestoneUsecase,
	deleteMilestoneUC *milestoneusecase.DeleteMilestoneUsecase,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *MilestoneHandler {
	return &MilestoneHandler{
		listMilestonesUC:  listMilestonesUC,
		createMilestoneUC: createMilestoneUC,
		updateMilestoneUC: updateMilestoneUC,
		deleteMilestoneUC: deleteMilestoneUC,
		resolveRepo:       resolveRepo,
	}
}

func (h *MilestoneHandler) SetAccess(a *RepoAccess) { h.access = a }

func (h *MilestoneHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	repoScope := middleware.RequireScope("repo")
	g.GET("/repos/:owner/:repo/milestones", h.ListMilestones, auth, repoScope)
	g.POST("/repos/:owner/:repo/milestones", h.CreateMilestone, auth, repoScope)
	g.PATCH("/repos/:owner/:repo/milestones/:number", h.UpdateMilestone, auth, repoScope)
	g.DELETE("/repos/:owner/:repo/milestones/:number", h.DeleteMilestone, auth, repoScope)
}

type createMilestoneRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	DueOn       *string `json:"due_on"`
}

type updateMilestoneRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	State       *string `json:"state"`
	DueOn       *string `json:"due_on"`
}

type milestoneResponse struct {
	ID           uuid.UUID  `json:"id"`
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	State        string     `json:"state"`
	DueOn        *time.Time `json:"due_on"`
	OpenIssues   int        `json:"open_issues"`
	ClosedIssues int        `json:"closed_issues"`
	NodeID       string     `json:"node_id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ClosedAt     *time.Time `json:"closed_at"`
}

func (h *MilestoneHandler) ListMilestones(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	state := c.QueryParam("state")
	if state == "" {
		state = "open"
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	output, err := h.listMilestonesUC.Execute(c.Request().Context(), milestoneusecase.ListMilestonesInput{
		RepositoryID: repo.ID,
		State:        state,
		Page:         page,
		PerPage:      perPage,
	})
	if err != nil {
		return err
	}

	setPaginationHeaders(c, output.Page, output.PerPage, output.Total)
	return c.JSON(http.StatusOK, toMilestoneResponses(output.Milestones))
}

func (h *MilestoneHandler) CreateMilestone(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}
	if err := h.access.EnsureWrite(c, repo); err != nil {
		return err
	}

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	var req createMilestoneRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	dueOn, err := parseMilestoneDueOn(req.DueOn)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid due_on")
	}

	milestone, err := h.createMilestoneUC.Execute(c.Request().Context(), milestoneusecase.CreateMilestoneInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		ActorID:        actorID,
		Title:          req.Title,
		Description:    req.Description,
		DueOn:          dueOn,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return err
	}

	return c.JSON(http.StatusCreated, toMilestoneResponse(milestone))
}

func (h *MilestoneHandler) UpdateMilestone(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}
	if err := h.access.EnsureWrite(c, repo); err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid milestone number")
	}

	var req updateMilestoneRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	var dueOn *time.Time
	if req.DueOn != nil {
		dueOn, err = parseMilestoneDueOn(req.DueOn)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid due_on")
		}
	}

	milestone, err := h.updateMilestoneUC.Execute(c.Request().Context(), milestoneusecase.UpdateMilestoneInput{
		RepositoryID: repo.ID,
		Number:       number,
		Title:        req.Title,
		Description:  req.Description,
		State:        req.State,
		DueOn:        dueOn,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.JSON(http.StatusOK, toMilestoneResponse(milestone))
}

func (h *MilestoneHandler) DeleteMilestone(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}
	if err := h.access.EnsureWrite(c, repo); err != nil {
		return err
	}

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid milestone number")
	}

	err = h.deleteMilestoneUC.Execute(c.Request().Context(), milestoneusecase.DeleteMilestoneInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		ActorID:        actorID,
		Number:         number,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func parseMilestoneDueOn(raw *string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func toMilestoneResponse(milestone *entity.Milestone) milestoneResponse {
	return milestoneResponse{
		ID:           milestone.ID,
		Number:       milestone.Number,
		Title:        milestone.Title,
		Description:  milestone.Description,
		State:        milestone.State,
		DueOn:        milestone.DueOn,
		OpenIssues:   milestone.OpenIssues,
		ClosedIssues: milestone.ClosedIssues,
		NodeID:       MilestoneNodeID(milestone.ID),
		CreatedAt:    milestone.CreatedAt,
		UpdatedAt:    milestone.UpdatedAt,
		ClosedAt:     milestone.ClosedAt,
	}
}

func toMilestoneResponses(milestones []*entity.Milestone) []milestoneResponse {
	result := make([]milestoneResponse, 0, len(milestones))
	for _, milestone := range milestones {
		result = append(result, toMilestoneResponse(milestone))
	}
	return result
}

func MilestoneNodeID(id uuid.UUID) string { return NodeID("Milestone", id.String()) }
