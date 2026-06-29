package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	labelusecase "github.com/open-git/backend/internal/usecase/label"
)

type LabelHandler struct {
	listLabelsUC  *labelusecase.ListLabelsUsecase
	createLabelUC *labelusecase.CreateLabelUsecase
	updateLabelUC *labelusecase.UpdateLabelUsecase
	deleteLabelUC *labelusecase.DeleteLabelUsecase
	resolveRepo   func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

func NewLabelHandler(
	listLabelsUC *labelusecase.ListLabelsUsecase,
	createLabelUC *labelusecase.CreateLabelUsecase,
	updateLabelUC *labelusecase.UpdateLabelUsecase,
	deleteLabelUC *labelusecase.DeleteLabelUsecase,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *LabelHandler {
	return &LabelHandler{
		listLabelsUC:  listLabelsUC,
		createLabelUC: createLabelUC,
		updateLabelUC: updateLabelUC,
		deleteLabelUC: deleteLabelUC,
		resolveRepo:   resolveRepo,
	}
}

func (h *LabelHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	repoScope := middleware.RequireScope("repo")
	g.GET("/repos/:owner/:repo/labels", h.ListLabels, auth, repoScope)
	g.POST("/repos/:owner/:repo/labels", h.CreateLabel, auth, repoScope)
	g.PATCH("/repos/:owner/:repo/labels/:name", h.UpdateLabel, auth, repoScope)
	g.DELETE("/repos/:owner/:repo/labels/:name", h.DeleteLabel, auth, repoScope)
}

type createLabelRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

type updateLabelRequest struct {
	NewName     *string `json:"new_name"`
	Color       *string `json:"color"`
	Description *string `json:"description"`
}

type labelResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Description string    `json:"description"`
	NodeID      string    `json:"node_id"`
}

func (h *LabelHandler) ListLabels(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	output, err := h.listLabelsUC.Execute(c.Request().Context(), labelusecase.ListLabelsInput{
		RepositoryID: repo.ID,
		Page:         page,
		PerPage:      perPage,
	})
	if err != nil {
		return err
	}

	setPaginationHeaders(c, output.Page, output.PerPage, output.Total)
	return c.JSON(http.StatusOK, toLabelResponses(output.Labels))
}

func (h *LabelHandler) CreateLabel(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	var req createLabelRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	label, err := h.createLabelUC.Execute(c.Request().Context(), labelusecase.CreateLabelInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		Name:           req.Name,
		Color:          req.Color,
		Description:    req.Description,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrValidation) || errors.Is(err, apperror.ErrConflict) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return err
	}

	return c.JSON(http.StatusCreated, toLabelResponse(label))
}

func (h *LabelHandler) UpdateLabel(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	var req updateLabelRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	label, err := h.updateLabelUC.Execute(c.Request().Context(), labelusecase.UpdateLabelInput{
		RepositoryID: repo.ID,
		CurrentName:  c.Param("name"),
		NewName:      req.NewName,
		Color:        req.Color,
		Description:  req.Description,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		if errors.Is(err, apperror.ErrValidation) || errors.Is(err, apperror.ErrConflict) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return err
	}

	return c.JSON(http.StatusOK, toLabelResponse(label))
}

func (h *LabelHandler) DeleteLabel(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	err = h.deleteLabelUC.Execute(c.Request().Context(), labelusecase.DeleteLabelInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		ActorID:        actorID,
		Name:           c.Param("name"),
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func toLabelResponse(label *entity.Label) labelResponse {
	return labelResponse{
		ID:          label.ID,
		Name:        label.Name,
		Color:       label.Color,
		Description: label.Description,
		NodeID:      LabelNodeID(label.ID),
	}
}

func toLabelResponses(labels []*entity.Label) []labelResponse {
	result := make([]labelResponse, 0, len(labels))
	for _, label := range labels {
		result = append(result, toLabelResponse(label))
	}
	return result
}

func LabelNodeID(id uuid.UUID) string { return NodeID("Label", id.String()) }
