package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	userUC "github.com/open-git/backend/internal/usecase/user"
)

type UserHandler struct {
	getCurrentUser  *userUC.GetCurrentUserUsecase
	getUserByLogin  *userUC.GetUserByLoginUsecase
	updateCurrentUser *userUC.UpdateUserUsecase
}

func NewUserHandler(
	getCurrentUser *userUC.GetCurrentUserUsecase,
	getUserByLogin *userUC.GetUserByLoginUsecase,
	updateCurrentUser *userUC.UpdateUserUsecase,
) *UserHandler {
	return &UserHandler{
		getCurrentUser:    getCurrentUser,
		getUserByLogin:    getUserByLogin,
		updateCurrentUser: updateCurrentUser,
	}
}

type userResponse struct {
	ID        int64  `json:"id"`
	NodeID    string `json:"node_id"`
	Login     string `json:"login"`
	HTMLURL   string `json:"html_url"`
	Email     string `json:"email,omitempty"`
	Name      string `json:"name,omitempty"`
	Bio       string `json:"bio,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Type      string `json:"type"`
}

type patchUserRequest struct {
	Name      string `json:"name"`
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

func (h *UserHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.GET("/user", h.GetCurrentUser, authMiddleware)
	g.PATCH("/user", h.UpdateCurrentUser, authMiddleware)
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
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", "", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", "", nil)
	}

	return RespondGitHubOK(c, toUserResponse(user, true, c.Request().Host))
}

func (h *UserHandler) GetUserByLogin(c echo.Context) error {
	user, err := h.getUserByLogin.Execute(c.Request().Context(), c.Param("username"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", "", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", "", nil)
	}

	includeEmail := middleware.UserIDFromContext(c) != 0
	return RespondGitHubOK(c, toUserResponse(user, includeEmail, c.Request().Host))
}

func (h *UserHandler) UpdateCurrentUser(c echo.Context) error {
	userUUID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	var req patchUserRequest
	if err := c.Bind(&req); err != nil {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", "", nil)
	}

	user, err := h.updateCurrentUser.Execute(c.Request().Context(), userUUID, userUC.UpdateUserInput{
		Name:      req.Name,
		Bio:       req.Bio,
		AvatarURL: req.AvatarURL,
		Email:     req.Email,
	})
	if err != nil {
		if err.Error() == "invalid email" {
			return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", "", nil)
		}
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", "", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", "", nil)
	}

	return RespondGitHubOK(c, entityToUserResponse(user, true, c.Request().Host))
}

func toUserResponse(u *domain.User, includeEmail bool, host string) userResponse {
	resp := userResponse{
		ID:      u.ID,
		NodeID:  UserNodeID(u.ID),
		Login:   u.Login,
		HTMLURL: "https://" + host + "/" + u.Login,
		Type:    "User",
	}
	if includeEmail {
		resp.Email = u.Email
	}
	return resp
}

func entityToUserResponse(u *entity.User, includeEmail bool, host string) userResponse {
	id := middleware.UUIDToInt64(u.ID)
	resp := userResponse{
		ID:        id,
		NodeID:    UserNodeID(id),
		Login:     u.Login,
		HTMLURL:   "https://" + host + "/" + u.Login,
		Name:      u.Name,
		Bio:       u.Bio,
		AvatarURL: u.AvatarURL,
		Type:      "User",
	}
	if includeEmail {
		resp.Email = u.Email
	}
	return resp
}
