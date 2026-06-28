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

type MilestoneHandler struct {
	listMilestonesUC  listMilestonesUC
	createMilestoneUC createMilestoneUC
	updateMilestoneUC updateMilestoneUC
	deleteMilestoneUC deleteMilestoneUC
	resolveRepo       func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

type MilestoneDTO struct {
	ID           uuid.UUID
	Number       int
	Title        string
	Description  string
	State        string
	DueOn        *time.Time
	OpenIssues   int
	ClosedIssues int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ClosedAt     *time.Time
}

type ListMilestonesInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	State          string
	Page           int
	PerPage        int
}

type ListMilestonesOutput struct {
	Milestones []*MilestoneDTO
	Total      int
	Page       int
	PerPage    int
}

type CreateMilestoneInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	ActorID        uuid.UUID
	Title          string
	Description    string
	State          string
	DueOn          *string
}

type UpdateMilestoneInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	ActorID        uuid.UUID
	Number         int
	Title          *string
	Description    *string
	State          *string
	DueOn          *string
}

type DeleteMilestoneInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	ActorID        uuid.UUID
	Number         int
}

type listMilestonesUC interface {
	Execute(ctx context.Context, input ListMilestonesInput) (*ListMilestonesOutput, error)
}

type createMilestoneUC interface {
	Execute(ctx context.Context, input CreateMilestoneInput) (*MilestoneDTO, error)
}

type updateMilestoneUC interface {
	Execute(ctx context.Context, input UpdateMilestoneInput) (*MilestoneDTO, error)
}

type deleteMilestoneUC interface {
	Execute(ctx context.Context, input DeleteMilestoneInput) error
}

func NewMilestoneHandler(
	listMilestonesUC listMilestonesUC,
	createMilestoneUC createMilestoneUC,
	updateMilestoneUC updateMilestoneUC,
	deleteMilestoneUC deleteMilestoneUC,
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

func (h *MilestoneHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	repoScope := middleware.RequireScope("repo")
	g.GET("/repos/:owner/:repo/milestones", h.ListMilestones, auth, repoScope)
	g.POST("/repos/:owner/:repo/milestones", h.CreateMilestone, auth, repoScope)
	g.PATCH("/repos/:owner/:repo/milestones/:number", h.UpdateMilestone, auth, repoScope)
	g.DELETE("/repos/:owner/:repo/milestones/:number", h.DeleteMilestone, auth, repoScope)
}

type milestoneResponse struct {
	ID           string  `json:"id"`
	Number       int     `json:"number"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	State        string  `json:"state"`
	DueOn        *string `json:"due_on"`
	OpenIssues   int     `json:"open_issues"`
	ClosedIssues int     `json:"closed_issues"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	ClosedAt     *string `json:"closed_at"`
}

type createMilestoneRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	State       string  `json:"state"`
	DueOn       *string `json:"due_on"`
}

type updateMilestoneRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	State       *string `json:"state"`
	DueOn       *string `json:"due_on"`
}

func (h *MilestoneHandler) ListMilestones(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	if _, err := middleware.GetUserUUID(c); err != nil {
		return err
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	output, err := h.listMilestonesUC.Execute(c.Request().Context(), ListMilestonesInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		State:          c.QueryParam("state"),
		Page:           page,
		PerPage:        perPage,
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

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	var req createMilestoneRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	milestone, err := h.createMilestoneUC.Execute(c.Request().Context(), CreateMilestoneInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		ActorID:        actorID,
		Title:          req.Title,
		Description:    req.Description,
		State:          req.State,
		DueOn:          req.DueOn,
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

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
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

	milestone, err := h.updateMilestoneUC.Execute(c.Request().Context(), UpdateMilestoneInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		ActorID:        actorID,
		Number:         number,
		Title:          req.Title,
		Description:    req.Description,
		State:          req.State,
		DueOn:          req.DueOn,
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

	return c.JSON(http.StatusOK, toMilestoneResponse(milestone))
}

func (h *MilestoneHandler) DeleteMilestone(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
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

	err = h.deleteMilestoneUC.Execute(c.Request().Context(), DeleteMilestoneInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		ActorID:        actorID,
		Number:         number,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Not Found")
		}
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func toMilestoneResponse(milestone *MilestoneDTO) milestoneResponse {
	resp := milestoneResponse{
		ID:           formatResourceID(milestone.ID),
		Number:       milestone.Number,
		Title:        milestone.Title,
		Description:  milestone.Description,
		State:        milestone.State,
		OpenIssues:   milestone.OpenIssues,
		ClosedIssues: milestone.ClosedIssues,
		CreatedAt:    formatTimestamp(milestone.CreatedAt),
		UpdatedAt:    formatTimestamp(milestone.UpdatedAt),
	}
	if milestone.DueOn != nil {
		formatted := milestone.DueOn.UTC().Format("2006-01-02T15:04:05Z")
		resp.DueOn = &formatted
	}
	if milestone.ClosedAt != nil {
		formatted := milestone.ClosedAt.UTC().Format(time.RFC3339)
		resp.ClosedAt = &formatted
	}
	return resp
}

func toMilestoneResponses(milestones []*MilestoneDTO) []milestoneResponse {
	result := make([]milestoneResponse, 0, len(milestones))
	for _, milestone := range milestones {
		result = append(result, toMilestoneResponse(milestone))
	}
	return result
}
