package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/middleware"
	userpreferencesUC "github.com/open-git/backend/internal/usecase/user_preferences"
)

type UserPreferencesHandler struct {
	getPreferences    *userpreferencesUC.GetUserPreferencesUsecase
	updatePreferences *userpreferencesUC.UpdateUserPreferencesUsecase
}

func NewUserPreferencesHandler(
	getPreferences *userpreferencesUC.GetUserPreferencesUsecase,
	updatePreferences *userpreferencesUC.UpdateUserPreferencesUsecase,
) *UserPreferencesHandler {
	return &UserPreferencesHandler{
		getPreferences:    getPreferences,
		updatePreferences: updatePreferences,
	}
}

type preferencesResponse struct {
	Theme string `json:"theme"`
}

type updatePreferencesRequest struct {
	Theme string `json:"theme"`
}

func (h *UserPreferencesHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.GET("/user/preferences", h.GetPreferences, authMiddleware)
	g.PUT("/user/preferences", h.UpdatePreferences, authMiddleware)
}

func (h *UserPreferencesHandler) GetPreferences(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	prefs, err := h.getPreferences.Execute(c.Request().Context(), userID)
	if err != nil {
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	return RespondGitHubOK(c, preferencesResponse{Theme: prefs.Theme})
}

func (h *UserPreferencesHandler) UpdatePreferences(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	var req updatePreferencesRequest
	if err := c.Bind(&req); err != nil {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", nil)
	}

	prefs, err := h.updatePreferences.Execute(c.Request().Context(), userID, req.Theme)
	if err != nil {
		if errors.Is(err, domain.ErrValidation) {
			return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	return RespondGitHubOK(c, preferencesResponse{Theme: prefs.Theme})
}
