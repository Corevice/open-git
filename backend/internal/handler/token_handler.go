package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/middleware"
	"github.com/open-git/backend/internal/repository"
	authUC "github.com/open-git/backend/internal/usecase/auth"
)

type entityUserLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
}

type TokenHandler struct {
	tokens   repository.IAccessTokenRepository
	issue    *authUC.IssuePATUsecase
	revoke   *authUC.RevokePATUsecase
	auditLog domainrepo.IAuditLogRepository
	users    entityUserLookup
}

func NewTokenHandler(
	tokens repository.IAccessTokenRepository,
	issue *authUC.IssuePATUsecase,
	revoke *authUC.RevokePATUsecase,
	auditLog domainrepo.IAuditLogRepository,
	users entityUserLookup,
) *TokenHandler {
	return &TokenHandler{
		tokens:   tokens,
		issue:    issue,
		revoke:   revoke,
		auditLog: auditLog,
		users:    users,
	}
}

type createTokenRequest struct {
	Note      string   `json:"note"`
	Scopes    []string `json:"scopes"`
	ExpiresAt *string  `json:"expires_at,omitempty"`
}

type tokenResponse struct {
	ID        int64    `json:"id"`
	Note      string   `json:"note,omitempty"`
	Scopes    []string `json:"scopes"`
	ExpiresAt *string  `json:"expires_at,omitempty"`
	RevokedAt *string  `json:"revoked_at,omitempty"`
}

type createTokenResponse struct {
	Token string        `json:"token"`
	Meta  tokenResponse `json:"meta"`
}

func (h *TokenHandler) List(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	tokens, err := h.tokens.ListByUserID(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to list tokens"})
	}

	resp := make([]tokenResponse, 0, len(tokens))
	for _, t := range tokens {
		resp = append(resp, toTokenResponse(t))
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *TokenHandler) Create(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	var req createTokenRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid request"})
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid expires_at"})
		}
		expiresAt = &parsed
	}

	out, err := h.issue.Execute(c.Request().Context(), authUC.IssuePATInput{
		UserID:    userID,
		Note:      req.Note,
		Scopes:    req.Scopes,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to create token"})
	}

	h.recordAudit(c.Request().Context(), middleware.Int64ToUUID(userID), out.Record.ID, "token.create")

	return c.JSON(http.StatusCreated, createTokenResponse{
		Token: out.Token,
		Meta:  toTokenResponse(out.Record),
	})
}

func (h *TokenHandler) Revoke(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	tokenID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid token id"})
	}

	if err := h.revoke.Execute(c.Request().Context(), userID, tokenID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to revoke token"})
	}

	h.recordAudit(c.Request().Context(), middleware.Int64ToUUID(userID), tokenID, "token.revoke")

	return c.NoContent(http.StatusNoContent)
}

func (h *TokenHandler) recordAudit(ctx context.Context, actorID uuid.UUID, tokenID int64, action string) {
	if h.auditLog == nil {
		return
	}

	login := ""
	if h.users != nil {
		user, err := h.users.GetByID(ctx, actorID)
		if err == nil && user != nil {
			login = user.Login
		}
	}

	metadata := map[string]any{}
	if login != "" {
		metadata["actor_login"] = login
	}

	entry := entity.AuditLog{
		OrganizationID: uuid.Nil,
		ActorID:        actorID,
		Action:         action,
		TargetType:     "token",
		TargetID:       strconv.FormatInt(tokenID, 10),
		Metadata:       metadata,
		CreatedAt:      time.Now().UTC(),
	}

	if err := h.auditLog.Create(ctx, &entry); err != nil {
		slog.Error("failed to record audit log", "error", err, "action", action)
	}
}

func toTokenResponse(t *domain.AccessToken) tokenResponse {
	resp := tokenResponse{
		ID:     t.ID,
		Note:   t.Note,
		Scopes: t.Scopes,
	}
	if t.ExpiresAt != nil {
		formatted := t.ExpiresAt.UTC().Format(time.RFC3339)
		resp.ExpiresAt = &formatted
	}
	if t.RevokedAt != nil {
		formatted := t.RevokedAt.UTC().Format(time.RFC3339)
		resp.RevokedAt = &formatted
	}
	return resp
}
