package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	actionsusecase "github.com/open-git/backend/internal/usecase/actions"
)

type CreateRegistrationTokenExecutor interface {
	Execute(ctx context.Context, orgID uuid.UUID, actorRole string) (*entity.RunnerRegistrationToken, string, error)
}

type RegisterRunnerExecutor interface {
	Execute(ctx context.Context, orgID uuid.UUID, req actionsusecase.RegisterRunnerRequest) (*entity.Runner, error)
}

type ListRunnersExecutor interface {
	Execute(ctx context.Context, orgID uuid.UUID) ([]*entity.Runner, error)
}

type DeleteRunnerExecutor interface {
	Execute(ctx context.Context, orgID uuid.UUID, runnerID uuid.UUID, actorRole string) error
}

type HeartbeatRunnerExecutor interface {
	Execute(ctx context.Context, orgID uuid.UUID, runnerID uuid.UUID, status string, runningJobID *string) error
}

type RunnerHandler struct {
	createTokenUC     CreateRegistrationTokenExecutor
	registerRunnerUC  RegisterRunnerExecutor
	listRunnersUC     ListRunnersExecutor
	deleteRunnerUC    DeleteRunnerExecutor
	heartbeatRunnerUC HeartbeatRunnerExecutor
	resolveOrgID      func(c echo.Context) (uuid.UUID, error)
	resolveActorRole  func(c echo.Context, orgID uuid.UUID) (string, error)
}

func NewRunnerHandler(
	createToken *actionsusecase.CreateRegistrationTokenUsecase,
	registerRunner *actionsusecase.RegisterRunnerUsecase,
	listRunners *actionsusecase.ListRunnersUsecase,
	deleteRunner *actionsusecase.DeleteRunnerUsecase,
	heartbeatRunner *actionsusecase.HeartbeatRunnerUsecase,
) *RunnerHandler {
	return NewRunnerHandlerWithDeps(createToken, registerRunner, listRunners, deleteRunner, heartbeatRunner, nil, nil)
}

func NewRunnerHandlerWithDeps(
	createToken CreateRegistrationTokenExecutor,
	registerRunner RegisterRunnerExecutor,
	listRunners ListRunnersExecutor,
	deleteRunner DeleteRunnerExecutor,
	heartbeatRunner HeartbeatRunnerExecutor,
	resolveOrgID func(c echo.Context) (uuid.UUID, error),
	resolveActorRole func(c echo.Context, orgID uuid.UUID) (string, error),
) *RunnerHandler {
	return &RunnerHandler{
		createTokenUC:     createToken,
		registerRunnerUC:  registerRunner,
		listRunnersUC:     listRunners,
		deleteRunnerUC:    deleteRunner,
		heartbeatRunnerUC: heartbeatRunner,
		resolveOrgID:      resolveOrgID,
		resolveActorRole:  resolveActorRole,
	}
}

func (h *RunnerHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/runners/registration-token", h.createRegistrationToken)
	g.POST("/runners", h.registerRunner)
	g.GET("/runners", h.listRunners)
	g.DELETE("/runners/:runner_id", h.deleteRunner)
	g.POST("/runners/:runner_id/heartbeat", h.heartbeatRunner)
}

type registrationTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

type registerRunnerRequest struct {
	RegistrationToken string   `json:"registration_token"`
	Name              string   `json:"name"`
	Labels            []string `json:"labels"`
	OS                string   `json:"os"`
	Arch              string   `json:"arch"`
	RunnerType        string   `json:"runner_type"`
}

type registerRunnerResponse struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Status string   `json:"status"`
	Labels []string `json:"labels"`
}

type listRunnersResponse struct {
	Runners []runnerResponse `json:"runners"`
}

type runnerResponse struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	Labels     []string `json:"labels"`
	LastSeenAt *string  `json:"last_seen_at,omitempty"`
	RunnerType string   `json:"runner_type"`
}

type heartbeatRequest struct {
	Status       string  `json:"status"`
	RunningJobID *string `json:"running_job_id"`
}

func (h *RunnerHandler) createRegistrationToken(c echo.Context) error {
	orgID, err := h.orgID(c)
	if err != nil {
		return err
	}

	token, raw, err := h.createTokenUC.Execute(c.Request().Context(), orgID, h.actorRole(c, orgID))
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
		}
		return err
	}

	return c.JSON(http.StatusCreated, registrationTokenResponse{
		Token:     raw,
		ExpiresAt: token.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

func (h *RunnerHandler) registerRunner(c echo.Context) error {
	orgID, err := h.orgID(c)
	if err != nil {
		return err
	}

	var req registerRunnerRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}

	runner, err := h.registerRunnerUC.Execute(c.Request().Context(), orgID, actionsusecase.RegisterRunnerRequest{
		RegistrationToken: req.RegistrationToken,
		Name:              req.Name,
		Labels:            req.Labels,
		OS:                req.OS,
		Arch:              req.Arch,
		RunnerType:        req.RunnerType,
	})
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		}
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": err.Error()})
		}
		return err
	}

	return c.JSON(http.StatusCreated, registerRunnerResponse{
		ID:     runner.ID.String(),
		Name:   runner.Name,
		Status: runner.Status,
		Labels: runner.Labels,
	})
}

func (h *RunnerHandler) listRunners(c echo.Context) error {
	orgID, err := h.orgID(c)
	if err != nil {
		return err
	}
	if h.actorRole(c, orgID) == "" {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
	}

	runners, err := h.listRunnersUC.Execute(c.Request().Context(), orgID)
	if err != nil {
		return err
	}

	responses := make([]runnerResponse, 0, len(runners))
	for _, runner := range runners {
		responses = append(responses, toRunnerResponse(runner))
	}

	return c.JSON(http.StatusOK, listRunnersResponse{Runners: responses})
}

func (h *RunnerHandler) deleteRunner(c echo.Context) error {
	orgID, err := h.orgID(c)
	if err != nil {
		return err
	}

	runnerID, err := uuid.Parse(c.Param("runner_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid runner_id"})
	}

	err = h.deleteRunnerUC.Execute(c.Request().Context(), orgID, runnerID, h.actorRole(c, orgID))
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
		}
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *RunnerHandler) heartbeatRunner(c echo.Context) error {
	orgID, err := h.orgID(c)
	if err != nil {
		return err
	}

	runnerID, err := uuid.Parse(c.Param("runner_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid runner_id"})
	}

	var req heartbeatRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}

	err = h.heartbeatRunnerUC.Execute(c.Request().Context(), orgID, runnerID, req.Status, req.RunningJobID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (h *RunnerHandler) orgID(c echo.Context) (uuid.UUID, error) {
	if h.resolveOrgID != nil {
		return h.resolveOrgID(c)
	}
	orgID, err := uuid.Parse(c.Param("org"))
	if err != nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid org"})
	}
	return orgID, nil
}

func (h *RunnerHandler) actorRole(c echo.Context, orgID uuid.UUID) string {
	if h.resolveActorRole != nil {
		role, _ := h.resolveActorRole(c, orgID)
		return role
	}
	for _, scope := range middleware.GetScopes(c) {
		if scope == entity.RoleAdmin {
			return entity.RoleAdmin
		}
	}
	return entity.RoleMember
}

func toRunnerResponse(runner *entity.Runner) runnerResponse {
	resp := runnerResponse{
		ID:         runner.ID.String(),
		Name:       runner.Name,
		Status:     runner.Status,
		Labels:     runner.Labels,
		RunnerType: runner.RunnerType,
	}
	if runner.LastSeenAt != nil {
		lastSeen := runner.LastSeenAt.UTC().Format(time.RFC3339)
		resp.LastSeenAt = &lastSeen
	}
	return resp
}
