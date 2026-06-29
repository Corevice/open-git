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
	createLabelUC      *labelusecase.CreateLabelUsecase
	listLabelsUC       *labelusecase.ListLabelsUsecase
	updateLabelUC      *labelusecase.UpdateLabelUsecase
	deleteLabelUC      *labelusecase.DeleteLabelUsecase
	addIssueLabelUC    *labelusecase.AddIssueLabelsUsecase
	removeIssueLabelUC *labelusecase.RemoveIssueLabelUsecase
	resolveRepo        func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

func NewLabelHandler(
	createLabelUC *labelusecase.CreateLabelUsecase,
	listLabelsUC *labelusecase.ListLabelsUsecase,
	updateLabelUC *labelusecase.UpdateLabelUsecase,
	deleteLabelUC *labelusecase.DeleteLabelUsecase,
	addIssueLabelUC *labelusecase.AddIssueLabelsUsecase,
	removeIssueLabelUC *labelusecase.RemoveIssueLabelUsecase,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *LabelHandler {
	return &LabelHandler{
		createLabelUC:      createLabelUC,
		listLabelsUC:       listLabelsUC,
		updateLabelUC:      updateLabelUC,
		deleteLabelUC:      deleteLabelUC,
		addIssueLabelUC:    addIssueLabelUC,
		removeIssueLabelUC: removeIssueLabelUC,
		resolveRepo:        resolveRepo,
	}
}

func (h *LabelHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	repoScope := middleware.RequireScope("repo")
	g.GET("/repos/:owner/:repo/labels", h.ListLabels, auth, repoScope)
	g.POST("/repos/:owner/:repo/labels", h.CreateLabel, auth, repoScope)
	g.PATCH("/repos/:owner/:repo/labels/:name", h.UpdateLabel, auth, repoScope)
	g.DELETE("/repos/:owner/:repo/labels/:name", h.DeleteLabel, auth, repoScope)
	g.POST("/repos/:owner/:repo/issues/:number/labels", h.AddIssueLabels, auth, repoScope)
	g.DELETE("/repos/:owner/:repo/issues/:number/labels/:name", h.RemoveIssueLabel, auth, repoScope)
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

type addIssueLabelsRequest struct {
	Labels []string `json:"labels"`
}

type labelResponse struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	URL         string `json:"url"`
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
	return c.JSON(http.StatusOK, toLabelResponses(output.Labels, c.Param("owner"), c.Param("repo"), c.Request().Host))
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

	return c.JSON(http.StatusCreated, toLabelResponse(label, c.Param("owner"), c.Param("repo"), c.Request().Host))
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

	return c.JSON(http.StatusOK, toLabelResponse(label, c.Param("owner"), c.Param("repo"), c.Request().Host))
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

func (h *LabelHandler) AddIssueLabels(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid issue number")
	}

	var req addIssueLabelsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := h.addIssueLabelUC.Execute(c.Request().Context(), labelusecase.AddIssueLabelsInput{
		RepositoryID: repo.ID,
		IssueNumber:  number,
		Names:        req.Labels,
	}); err != nil {
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return err
	}

	output, err := h.listLabelsUC.Execute(c.Request().Context(), labelusecase.ListLabelsInput{
		RepositoryID: repo.ID,
		Page:         1,
		PerPage:      100,
	})
	if err != nil {
		return err
	}

	nameSet := make(map[string]struct{}, len(req.Labels))
	for _, name := range req.Labels {
		nameSet[name] = struct{}{}
	}

	added := make([]*entity.Label, 0, len(req.Labels))
	for _, label := range output.Labels {
		if _, ok := nameSet[label.Name]; ok {
			added = append(added, label)
		}
	}

	return c.JSON(http.StatusOK, toLabelResponses(added, c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *LabelHandler) RemoveIssueLabel(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid issue number")
	}

	err = h.removeIssueLabelUC.Execute(c.Request().Context(), labelusecase.RemoveIssueLabelInput{
		RepositoryID: repo.ID,
		IssueNumber:  number,
		Name:         c.Param("name"),
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func toLabelResponse(label *entity.Label, owner, repoName, host string) labelResponse {
	return labelResponse{
		ID:          middleware.UUIDToInt64(label.ID),
		NodeID:      LabelNodeID(label.ID),
		Name:        label.Name,
		Color:       label.Color,
		Description: label.Description,
		URL:         "https://" + host + "/repos/" + owner + "/" + repoName + "/labels/" + label.Name,
	}
}

func toLabelResponses(labels []*entity.Label, owner, repoName, host string) []labelResponse {
	result := make([]labelResponse, 0, len(labels))
	for _, label := range labels {
		result = append(result, toLabelResponse(label, owner, repoName, host))
	}
	return result
}

func LabelNodeID(id uuid.UUID) string { return NodeID("Label", id.String()) }
