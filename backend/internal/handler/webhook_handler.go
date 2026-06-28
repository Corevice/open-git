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
	webhookusecase "github.com/open-git/backend/internal/usecase/webhook"
)

type WebhookHandler struct {
	createWebhookUC  *webhookusecase.CreateWebhookUsecase
	listWebhooksUC   *webhookusecase.ListWebhooksUsecase
	getWebhookUC     *webhookusecase.GetWebhookUsecase
	updateWebhookUC  *webhookusecase.UpdateWebhookUsecase
	deleteWebhookUC  *webhookusecase.DeleteWebhookUsecase
	listDeliveriesUC *webhookusecase.ListDeliveriesUsecase
	getDeliveryUC    *webhookusecase.GetDeliveryUsecase
	redeliverUC      *webhookusecase.RedeliverWebhookUsecase
	pingWebhookUC    *webhookusecase.PingWebhookUsecase
	resolveRepo      func(c echo.Context, owner, repo string) (*entity.Repository, error)
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

func NewWebhookHandlerWithDeliveries(
	createWebhookUC *webhookusecase.CreateWebhookUsecase,
	listWebhooksUC *webhookusecase.ListWebhooksUsecase,
	getWebhookUC *webhookusecase.GetWebhookUsecase,
	updateWebhookUC *webhookusecase.UpdateWebhookUsecase,
	deleteWebhookUC *webhookusecase.DeleteWebhookUsecase,
	listDeliveriesUC *webhookusecase.ListDeliveriesUsecase,
	getDeliveryUC *webhookusecase.GetDeliveryUsecase,
	redeliverUC *webhookusecase.RedeliverWebhookUsecase,
	pingWebhookUC *webhookusecase.PingWebhookUsecase,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *WebhookHandler {
	return &WebhookHandler{
		createWebhookUC:  createWebhookUC,
		listWebhooksUC:   listWebhooksUC,
		getWebhookUC:     getWebhookUC,
		updateWebhookUC:  updateWebhookUC,
		deleteWebhookUC:  deleteWebhookUC,
		listDeliveriesUC: listDeliveriesUC,
		getDeliveryUC:    getDeliveryUC,
		redeliverUC:      redeliverUC,
		pingWebhookUC:    pingWebhookUC,
		resolveRepo:      resolveRepo,
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
	if h.listDeliveriesUC != nil {
		g.GET("/repos/:owner/:repo/hooks/:hook_id/deliveries", h.ListDeliveries, auth, writeScope)
		g.GET("/repos/:owner/:repo/hooks/:hook_id/deliveries/:delivery_id", h.GetDelivery, auth, writeScope)
		g.POST("/repos/:owner/:repo/hooks/:hook_id/deliveries/:delivery_id/attempts", h.RedeliverDelivery, auth, adminScope)
	}
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

type deliverySummaryResponse struct {
	ID          uuid.UUID  `json:"id"`
	Event       string     `json:"event"`
	Status      string     `json:"status"`
	StatusCode  *int       `json:"status_code"`
	DeliveredAt *time.Time `json:"delivered_at"`
	DurationMs  *int       `json:"duration_ms"`
	Redelivery  bool       `json:"redelivery"`
}

type deliveryDetailResponse struct {
	ID               uuid.UUID           `json:"id"`
	Event            string              `json:"event"`
	Status           string              `json:"status"`
	StatusCode       *int                `json:"status_code"`
	RequestHeaders   map[string][]string `json:"request_headers"`
	RequestBody      string              `json:"request_body"`
	ResponseHeaders  map[string][]string `json:"response_headers"`
	ResponseBody     *string             `json:"response_body"`
	DurationMs       *int                `json:"duration_ms"`
	Redelivery       bool                `json:"redelivery"`
	ParentDeliveryID *uuid.UUID          `json:"parent_delivery_id"`
	DeliveredAt      *time.Time          `json:"delivered_at"`
	CreatedAt        time.Time           `json:"created_at"`
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

func (h *WebhookHandler) ListDeliveries(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	hookID, err := uuid.Parse(c.Param("hook_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid hook_id")
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	deliveries, total, err := h.listDeliveriesUC.Execute(
		c.Request().Context(),
		hookID,
		repo.OrganizationID,
		page,
		perPage,
	)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	setPaginationHeaders(c, page, perPage, int(total))

	responses := make([]deliverySummaryResponse, 0, len(deliveries))
	for _, delivery := range deliveries {
		responses = append(responses, toDeliverySummaryResponse(delivery))
	}
	return c.JSON(http.StatusOK, responses)
}

func (h *WebhookHandler) GetDelivery(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	hookID, err := uuid.Parse(c.Param("hook_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid hook_id")
	}

	deliveryID, err := uuid.Parse(c.Param("delivery_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid delivery_id")
	}

	delivery, err := h.getDeliveryUC.Execute(
		c.Request().Context(),
		deliveryID,
		hookID,
		repo.OrganizationID,
	)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.JSON(http.StatusOK, toDeliveryDetailResponse(delivery))
}

func (h *WebhookHandler) RedeliverDelivery(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	hookID, err := uuid.Parse(c.Param("hook_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid hook_id")
	}

	deliveryID, err := uuid.Parse(c.Param("delivery_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid delivery_id")
	}

	delivery, err := h.redeliverUC.Execute(
		c.Request().Context(),
		deliveryID,
		hookID,
		repo.OrganizationID,
	)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.JSON(http.StatusCreated, toDeliveryDetailResponse(delivery))
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

	if h.pingWebhookUC != nil {
		if _, err := h.pingWebhookUC.Execute(c.Request().Context(), hookID, repo.OrganizationID); err != nil {
			if errors.Is(err, apperror.ErrNotFound) {
				return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
			}
			return err
		}
		return c.NoContent(http.StatusNoContent)
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

func toDeliverySummaryResponse(delivery *entity.WebhookDelivery) deliverySummaryResponse {
	return deliverySummaryResponse{
		ID:          delivery.ID,
		Event:       delivery.Event,
		Status:      delivery.Status,
		StatusCode:  delivery.StatusCode,
		DeliveredAt: delivery.DeliveredAt,
		DurationMs:  delivery.DurationMs,
		Redelivery:  delivery.Redelivery,
	}
}

func toDeliveryDetailResponse(delivery *entity.WebhookDelivery) deliveryDetailResponse {
	return deliveryDetailResponse{
		ID:               delivery.ID,
		Event:            delivery.Event,
		Status:           delivery.Status,
		StatusCode:       delivery.StatusCode,
		RequestHeaders:   delivery.RequestHeaders,
		RequestBody:      delivery.RequestBody,
		ResponseHeaders:  delivery.ResponseHeaders,
		ResponseBody:     delivery.ResponseBody,
		DurationMs:       delivery.DurationMs,
		Redelivery:       delivery.Redelivery,
		ParentDeliveryID: delivery.ParentDeliveryID,
		DeliveredAt:      delivery.DeliveredAt,
		CreatedAt:        delivery.CreatedAt,
	}
}
