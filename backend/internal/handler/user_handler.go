package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/middleware"
	userUC "github.com/open-git/backend/internal/usecase/user"
)

type UserHandler struct {
	getCurrentUser *userUC.GetCurrentUserUsecase
	getUserByLogin *userUC.GetUserByLoginUsecase
}

func NewUserHandler(
	getCurrentUser *userUC.GetCurrentUserUsecase,
	getUserByLogin *userUC.GetUserByLoginUsecase,
) *UserHandler {
	return &UserHandler{
		getCurrentUser: getCurrentUser,
		getUserByLogin: getUserByLogin,
	}
}

type userResponse struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Email string `json:"email,omitempty"`
	Type  string `json:"type"`
}

type githubErrorBody struct {
	Message          string `json:"message"`
	DocumentationURL string `json:"documentation_url,omitempty"`
}

func RespondGitHubOK(c echo.Context, data any) error {
	c.Response().Header().Set("X-GitHub-Media-Type", "github.v3")
	return c.JSON(http.StatusOK, data)
}

func RespondGitHubError(c echo.Context, status int, message string, _ any) error {
	return c.JSON(status, githubErrorBody{Message: message})
}

func (h *UserHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.GET("/user", h.GetCurrentUser, authMiddleware)
	g.GET("/users/:username", h.GetUserByLogin, middleware.OptionalAuth())
}

func (h *UserHandler) GetCurrentUser(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	user, err := h.getCurrentUser.Execute(c.Request().Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	return RespondGitHubOK(c, toUserResponse(user, true))
}

func (h *UserHandler) GetUserByLogin(c echo.Context) error {
	user, err := h.getUserByLogin.Execute(c.Request().Context(), c.Param("username"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	includeEmail := middleware.UserIDFromContext(c) != 0
	return RespondGitHubOK(c, toUserResponse(user, includeEmail))
}

func toUserResponse(u *domain.User, includeEmail bool) userResponse {
	resp := userResponse{
		ID:    u.ID,
		Login: u.Login,
		Type:  "User",
	}
	if includeEmail {
		resp.Email = u.Email
	}
	return resp
}
