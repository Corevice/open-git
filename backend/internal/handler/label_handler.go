package handler

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var labelColorPattern = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)

type LabelHandler struct {
	listLabelsUC     listLabelsUC
	createLabelUC      createLabelUC
	updateLabelUC      updateLabelUC
	deleteLabelUC      deleteLabelUC
	addIssueLabelsUC   addIssueLabelsUC
	removeIssueLabelUC removeIssueLabelUC
	resolveRepo        func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

type LabelDTO struct {
	ID          uuid.UUID
	Name        string
	Color       string
	Description string
}

type ListLabelsInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Page           int
	PerPage        int
}

type ListLabelsOutput struct {
	Labels  []*LabelDTO
	Total   int
	Page    int
	PerPage int
}

type CreateLabelInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	ActorID        uuid.UUID
	Name           string
	Color          string
	Description    string
}

type UpdateLabelInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	ActorID        uuid.UUID
	Name           string
	NewName        *string
	Color          *string
	Description    *string
}

type DeleteLabelInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	ActorID        uuid.UUID
	Name           string
}

type AddIssueLabelsInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	IssueNumber    int
	ActorID        uuid.UUID
	Labels         []string
}

type RemoveIssueLabelInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	IssueNumber    int
	ActorID        uuid.UUID
	LabelName      string
}

type listLabelsUC interface {
	Execute(ctx context.Context, input ListLabelsInput) (*ListLabelsOutput, error)
}

type createLabelUC interface {
	Execute(ctx context.Context, input CreateLabelInput) (*LabelDTO, error)
}

type updateLabelUC interface {
	Execute(ctx context.Context, input UpdateLabelInput) (*LabelDTO, error)
}

type deleteLabelUC interface {
	Execute(ctx context.Context, input DeleteLabelInput) error
}

type addIssueLabelsUC interface {
	Execute(ctx context.Context, input AddIssueLabelsInput) ([]*LabelDTO, error)
}

type removeIssueLabelUC interface {
	Execute(ctx context.Context, input RemoveIssueLabelInput) ([]*LabelDTO, error)
}

func NewLabelHandler(
	listLabelsUC listLabelsUC,
	createLabelUC createLabelUC,
	updateLabelUC updateLabelUC,
	deleteLabelUC deleteLabelUC,
	addIssueLabelsUC addIssueLabelsUC,
	removeIssueLabelUC removeIssueLabelUC,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *LabelHandler {
	return &LabelHandler{
		listLabelsUC:       listLabelsUC,
		createLabelUC:      createLabelUC,
		updateLabelUC:      updateLabelUC,
		deleteLabelUC:      deleteLabelUC,
		addIssueLabelsUC:   addIssueLabelsUC,
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

type labelResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
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

func (h *LabelHandler) ListLabels(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	if _, err := middleware.GetUserUUID(c); err != nil {
		return err
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	output, err := h.listLabelsUC.Execute(c.Request().Context(), ListLabelsInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		Page:           page,
		PerPage:        perPage,
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

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	var req createLabelRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if !labelColorPattern.MatchString(req.Color) {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", []GitHubFieldError{
			{Resource: "Label", Field: "color", Code: "invalid"},
		})
	}

	label, err := h.createLabelUC.Execute(c.Request().Context(), CreateLabelInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		ActorID:        actorID,
		Name:           req.Name,
		Color:          req.Color,
		Description:    req.Description,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrValidation) {
			return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", []GitHubFieldError{
				{Resource: "Label", Field: "color", Code: "invalid"},
			})
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

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	var req updateLabelRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Color != nil && !labelColorPattern.MatchString(*req.Color) {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", []GitHubFieldError{
			{Resource: "Label", Field: "color", Code: "invalid"},
		})
	}

	label, err := h.updateLabelUC.Execute(c.Request().Context(), UpdateLabelInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		ActorID:        actorID,
		Name:           c.Param("name"),
		NewName:        req.NewName,
		Color:          req.Color,
		Description:    req.Description,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrValidation) {
			return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", []GitHubFieldError{
				{Resource: "Label", Field: "color", Code: "invalid"},
			})
		}
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Not Found")
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

	err = h.deleteLabelUC.Execute(c.Request().Context(), DeleteLabelInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		ActorID:        actorID,
		Name:           c.Param("name"),
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Not Found")
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

	actorID, err := middleware.GetUserUUID(c)
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

	labels, err := h.addIssueLabelsUC.Execute(c.Request().Context(), AddIssueLabelsInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		IssueNumber:    number,
		ActorID:        actorID,
		Labels:         req.Labels,
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

	return c.JSON(http.StatusOK, toLabelResponses(labels))
}

func (h *LabelHandler) RemoveIssueLabel(c echo.Context) error {
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
		return echo.NewHTTPError(http.StatusBadRequest, "invalid issue number")
	}

	labels, err := h.removeIssueLabelUC.Execute(c.Request().Context(), RemoveIssueLabelInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		IssueNumber:    number,
		ActorID:        actorID,
		LabelName:      c.Param("name"),
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Not Found")
		}
		return err
	}

	return c.JSON(http.StatusOK, toLabelResponses(labels))
}

func toLabelResponse(label *LabelDTO) labelResponse {
	return labelResponse{
		ID:          formatResourceID(label.ID),
		Name:        label.Name,
		Color:       label.Color,
		Description: label.Description,
	}
}

func toLabelResponses(labels []*LabelDTO) []labelResponse {
	result := make([]labelResponse, 0, len(labels))
	for _, label := range labels {
		result = append(result, toLabelResponse(label))
	}
	return result
}
