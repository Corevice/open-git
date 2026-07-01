package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
	authUC "github.com/open-git/backend/internal/usecase/auth"
)

type OAuthHandler struct {
	authorize *authUC.OAuthAuthorizeUsecase
	token     *authUC.OAuthTokenUsecase
}

func NewOAuthHandler(authorize *authUC.OAuthAuthorizeUsecase, token *authUC.OAuthTokenUsecase) *OAuthHandler {
	return &OAuthHandler{authorize: authorize, token: token}
}

func (h *OAuthHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	g.GET("/login/oauth/authorize", h.Authorize, auth)
	g.POST("/login/oauth/authorize", h.Authorize, auth)
	g.POST("/login/oauth/access_token", h.AccessToken)
}

func (h *OAuthHandler) Authorize(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	param := func(name string) string {
		if v := c.QueryParam(name); v != "" {
			return v
		}
		return c.FormValue(name)
	}
	clientID := param("client_id")
	redirectURI := param("redirect_uri")
	scope := param("scope")
	state := param("state")

	code, err := h.authorize.Execute(c.Request().Context(), authUC.OAuthAuthorizeInput{
		UserID:      userID,
		ClientID:    clientID,
		RedirectURI: redirectURI,
		Scope:       scope,
		State:       state,
	})
	if err != nil {
		switch {
		case errors.Is(err, authUC.ErrMissingState):
			return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "state is required"})
		case errors.Is(err, authUC.ErrRedirectURIMismatch):
			return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "redirect_uri mismatch"})
		case errors.Is(err, authUC.ErrInvalidClient):
			return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid client"})
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "authorization failed"})
		}
	}

	target, err := url.Parse(redirectURI)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid redirect_uri"})
	}
	query := target.Query()
	query.Set("code", code)
	query.Set("state", state)
	target.RawQuery = query.Encode()

	// The web consent page calls this endpoint via XHR with a Bearer token (a
	// full-page navigation cannot carry the Authorization header), so when the
	// client asks for JSON, hand back the redirect target instead of a 302 the
	// XHR would transparently follow cross-origin.
	if strings.Contains(c.Request().Header.Get("Accept"), "application/json") {
		return c.JSON(http.StatusOK, map[string]string{"redirect_url": target.String()})
	}

	return c.Redirect(http.StatusFound, target.String())
}

func (h *OAuthHandler) AccessToken(c echo.Context) error {
	param := func(name string) string {
		if v := c.FormValue(name); v != "" {
			return v
		}
		return c.QueryParam(name)
	}

	out, err := h.token.Execute(c.Request().Context(), authUC.OAuthTokenInput{
		Code:         param("code"),
		ClientID:     param("client_id"),
		ClientSecret: param("client_secret"),
	})
	if err != nil {
		if errors.Is(err, authUC.ErrInvalidCode) {
			return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid code"})
		}
		if errors.Is(err, authUC.ErrInvalidClient) {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "invalid client credentials"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "token exchange failed"})
	}

	return c.JSON(http.StatusOK, out)
}
