package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/middleware"
	"github.com/open-git/backend/internal/repository"
	authUC "github.com/open-git/backend/internal/usecase/auth"
)

type TokenHandler struct {
	tokens repository.IAccessTokenRepository
	issue  *authUC.IssuePATUsecase
	revoke *authUC.RevokePATUsecase
}

func NewTokenHandler(tokens repository.IAccessTokenRepository, issue *authUC.IssuePATUsecase, revoke *authUC.RevokePATUsecase) *TokenHandler {
	return &TokenHandler{tokens: tokens, issue: issue, revoke: revoke}
}

type createTokenRequest struct {
	Scopes    []string `json:"scopes"`
	ExpiresAt *string  `json:"expires_at,omitempty"`
}

type tokenResponse struct {
	ID        int64    `json:"id"`
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
		Scopes:    req.Scopes,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to create token"})
	}

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

	return c.NoContent(http.StatusNoContent)
}

func toTokenResponse(t *domain.AccessToken) tokenResponse {
	resp := tokenResponse{
		ID:     t.ID,
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
