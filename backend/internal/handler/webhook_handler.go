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
	webhookusecase "github.com/open-git/backend/internal/usecase/webhook"
)

type WebhookHandler struct {
	createWebhookUC *webhookusecase.CreateWebhookUsecase
	listWebhooksUC  *webhookusecase.ListWebhooksUsecase
	getWebhookUC    *webhookusecase.GetWebhookUsecase
	updateWebhookUC *webhookusecase.UpdateWebhookUsecase
	deleteWebhookUC *webhookusecase.DeleteWebhookUsecase
	resolveRepo     func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

func NewWebhookHandler(
	createWebhookUC *webhookusecase.CreateWebhookUsecase,
	listWebhooksUC *webhookusecase.ListWebhooksUsecase,
	getWebhookUC *webhookusecase.GetWebhookUsecase,
	updateWebhookUC *webhookusecase.UpdateWebhookUsecase,
	deleteWebhookUC *webhookusecase.DeleteWebhookUsecase,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *WebhookHandler {
	return &WebhookHandler{
		createWebhookUC: createWebhookUC,
		listWebhooksUC:  listWebhooksUC,
		getWebhookUC:    getWebhookUC,
		updateWebhookUC: updateWebhookUC,
		deleteWebhookUC: deleteWebhookUC,
		resolveRepo:     resolveRepo,
	}
}

func (h *WebhookHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	writeScope := middleware.RequireScope("write")
	adminScope := middleware.RequireScope("admin")

	g.GET("/repos/:owner/:repo/hooks", h.ListHooks, auth, writeScope)
	g.POST("/repos/:owner/:repo/hooks", h.CreateHook, auth, adminScope)
	g.GET("/repos/:owner/:repo/hooks/:hook_id", h.GetHook, auth, writeScope)
	g.PATCH("/repos/:owner/:repo/hooks/:hook_id", h.UpdateHook, auth, adminScope)
	g.DELETE("/repos/:owner/:repo/hooks/:hook_id", h.DeleteHook, auth, adminScope)
	g.POST("/repos/:owner/:repo/hooks/:hook_id/pings", h.PingHook, auth, adminScope)
	g.POST("/repos/:owner/:repo/hooks/:hook_id/tests", h.TestHook, auth, adminScope)
}

type webhookConfigRequest struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret"`
}

type createHookRequest struct {
	Name   string               `json:"name"`
	Active bool                 `json:"active"`
	Events []string             `json:"events"`
	Config webhookConfigRequest `json:"config"`
}

type updateHookRequest struct {
	Active *bool                 `json:"active"`
	Events []string              `json:"events"`
	Config *webhookConfigRequest `json:"config"`
}

type webhookConfigResponse struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret"`
}

type webhookResponse struct {
	ID     uuid.UUID             `json:"id"`
	Name   string                `json:"name"`
	Active bool                  `json:"active"`
	Events []string              `json:"events"`
	Config webhookConfigResponse `json:"config"`
}

func (h *WebhookHandler) ListHooks(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	output, err := h.listWebhooksUC.Execute(c.Request().Context(), webhookusecase.ListWebhooksInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   &repo.ID,
		Page:           page,
		PerPage:        perPage,
	})
	if err != nil {
		return err
	}

	setPaginationHeaders(c, output.Page, output.PerPage, int(output.Total))
	responses := make([]webhookResponse, 0, len(output.Webhooks))
	for _, hook := range output.Webhooks {
		responses = append(responses, toWebhookResponse(hook))
	}
	return c.JSON(http.StatusOK, responses)
}

func (h *WebhookHandler) CreateHook(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	var req createHookRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	webhook, err := h.createWebhookUC.Execute(
		c.Request().Context(),
		repo.OrganizationID,
		repo.ID,
		webhookusecase.CreateWebhookInput{
			ActorID:     middleware.UserUUIDFromContext(c),
			URL:         req.Config.URL,
			ContentType: req.Config.ContentType,
			Secret:      req.Config.Secret,
			Events:      req.Events,
			Active:      req.Active,
		},
	)
	if err != nil {
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return err
	}

	resp := toWebhookResponse(webhook)
	if req.Name != "" {
		resp.Name = req.Name
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *WebhookHandler) GetHook(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	hookID, err := uuid.Parse(c.Param("hook_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid hook_id")
	}

	webhook, err := h.getWebhookUC.Execute(c.Request().Context(), repo.OrganizationID, hookID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.JSON(http.StatusOK, toWebhookResponse(webhook))
}

func (h *WebhookHandler) UpdateHook(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	hookID, err := uuid.Parse(c.Param("hook_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid hook_id")
	}

	var req updateHookRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	input := webhookusecase.UpdateWebhookInput{
		ActorID: middleware.UserUUIDFromContext(c),
	}
	if req.Active != nil {
		input.Active = req.Active
	}
	if req.Events != nil {
		input.Events = req.Events
	}
	if req.Config != nil {
		if req.Config.URL != "" {
			url := req.Config.URL
			input.URL = &url
		}
		if req.Config.ContentType != "" {
			contentType := req.Config.ContentType
			input.ContentType = &contentType
		}
		if req.Config.Secret != "" {
			secret := req.Config.Secret
			input.Secret = &secret
		}
	}

	webhook, err := h.updateWebhookUC.Execute(c.Request().Context(), repo.OrganizationID, hookID, input)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return err
	}

	return c.JSON(http.StatusOK, toWebhookResponse(webhook))
}

func (h *WebhookHandler) DeleteHook(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	hookID, err := uuid.Parse(c.Param("hook_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid hook_id")
	}

	err = h.deleteWebhookUC.Execute(
		c.Request().Context(),
		repo.OrganizationID,
		hookID,
		webhookusecase.DeleteWebhookInput{
			ActorID: middleware.UserUUIDFromContext(c),
		},
	)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *WebhookHandler) PingHook(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	hookID, err := uuid.Parse(c.Param("hook_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid hook_id")
	}

	if _, err := h.getWebhookUC.Execute(c.Request().Context(), repo.OrganizationID, hookID); err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *WebhookHandler) TestHook(c echo.Context) error {
	return h.PingHook(c)
}

func toWebhookResponse(webhook *entity.Webhook) webhookResponse {
	name := "web"
	return webhookResponse{
		ID:     webhook.ID,
		Name:   name,
		Active: webhook.Active,
		Events: webhook.Events,
		Config: webhookConfigResponse{
			URL:         webhook.URL,
			ContentType: webhook.ContentType,
			Secret:      "",
		},
	}
}
