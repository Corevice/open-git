package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
)

// oauthTokenRevoker is the slice of the PAT repository the handler needs to
// revoke tokens a user obtained through an OAuth app. Kept local so the shared
// IAccessTokenRepository interface (and all its mocks) stay untouched.
type oauthTokenRevoker interface {
	RevokeAllByOAuthApp(ctx context.Context, userID int64, oauthAppID string) error
}

// OAuthAppHandler serves the OAuth application management API used by the
// settings/developers pages, plus the user's authorized-applications list.
type OAuthAppHandler struct {
	apps           repo.IOAuthAppRepository
	oauthTokens    repo.IOAuthAccessTokenRepository
	authorizations repo.IOAuthAuthorizationRepository
	patRevoker     oauthTokenRevoker
}

func NewOAuthAppHandler(
	apps repo.IOAuthAppRepository,
	oauthTokens repo.IOAuthAccessTokenRepository,
	authorizations repo.IOAuthAuthorizationRepository,
	patRevoker oauthTokenRevoker,
) *OAuthAppHandler {
	return &OAuthAppHandler{
		apps:           apps,
		oauthTokens:    oauthTokens,
		authorizations: authorizations,
		patRevoker:     patRevoker,
	}
}

func (h *OAuthAppHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	g.GET("/user/oauth-apps", h.ListMine, auth)
	g.POST("/oauth-apps", h.Create, auth)
	g.GET("/oauth-apps/:id", h.Get, auth)
	g.PATCH("/oauth-apps/:id", h.Update, auth)
	g.DELETE("/oauth-apps/:id", h.Delete, auth)
	g.POST("/oauth-apps/:id/secret", h.RegenerateSecret, auth)
	g.GET("/user/installations/authorizations", h.ListAuthorizations, auth)
	g.DELETE("/user/authorizations/:app_id", h.RevokeAuthorization, auth)
}

type oauthAppResponse struct {
	ID           string    `json:"id"`
	ClientID     string    `json:"client_id"`
	Name         string    `json:"name"`
	HomepageURL  string    `json:"homepage_url"`
	CallbackURLs []string  `json:"callback_urls"`
	OwnerType    string    `json:"owner_type"`
	CreatedAt    time.Time `json:"created_at"`
}

type oauthAppWithSecretResponse struct {
	oauthAppResponse
	ClientSecret string `json:"client_secret"`
}

type oauthAppCreateRequest struct {
	Name         string   `json:"name"`
	HomepageURL  string   `json:"homepage_url"`
	CallbackURLs []string `json:"callback_urls"`
	OwnerType    string   `json:"owner_type"`
}

type oauthAppUpdateRequest struct {
	Name         *string   `json:"name"`
	HomepageURL  *string   `json:"homepage_url"`
	CallbackURLs *[]string `json:"callback_urls"`
}

type oauthAuthorizationResponse struct {
	OAuthAppID    string    `json:"oauth_app_id"`
	AppName       string    `json:"app_name"`
	HomepageURL   string    `json:"homepage_url,omitempty"`
	GrantedScopes []string  `json:"granted_scopes"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toOAuthAppResponse(app *domain.OAuthApp) oauthAppResponse {
	callbacks := app.RedirectURIs
	if callbacks == nil {
		callbacks = []string{}
	}
	return oauthAppResponse{
		ID:           app.ID,
		ClientID:     app.ClientID,
		Name:         app.Name,
		HomepageURL:  app.HomepageURL,
		CallbackURLs: callbacks,
		OwnerType:    app.OwnerType,
		CreatedAt:    app.CreatedAt,
	}
}

func generateOAuthClientCredential(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashOAuthClientSecret(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func mapOAuthAppErr(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	case errors.Is(err, domain.ErrValidation):
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
	case errors.Is(err, domain.ErrConflict):
		return echo.NewHTTPError(http.StatusConflict, map[string]string{"message": "conflict"})
	default:
		return err
	}
}

func (h *OAuthAppHandler) ListMine(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	page, perPage, err := middleware.ParsePaginationParams(c)
	if err != nil {
		return err
	}

	apps, err := h.apps.ListByOwnerUser(c.Request().Context(), userID, page, perPage)
	if err != nil {
		return mapOAuthAppErr(c, err)
	}

	resp := make([]oauthAppResponse, 0, len(apps))
	for _, app := range apps {
		resp = append(resp, toOAuthAppResponse(app))
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *OAuthAppHandler) Create(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	var req oauthAppCreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid request body"})
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "name is required"})
	}

	ownerType := req.OwnerType
	if ownerType == "" {
		ownerType = "user"
	}
	if ownerType != "user" {
		// Organization-owned apps need org-role checks that this endpoint does
		// not implement yet; reject rather than silently mis-own the app.
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "only user-owned OAuth apps are supported"})
	}

	clientID, err := generateOAuthClientCredential(10)
	if err != nil {
		return err
	}
	clientSecret, err := generateOAuthClientCredential(20)
	if err != nil {
		return err
	}

	app := &domain.OAuthApp{
		ClientID:         clientID,
		ClientSecretHash: hashOAuthClientSecret(clientSecret),
		RedirectURIs:     req.CallbackURLs,
		Name:             req.Name,
		HomepageURL:      req.HomepageURL,
		OwnerType:        ownerType,
		OwnerUserID:      userID,
	}
	if err := h.apps.Create(c.Request().Context(), app); err != nil {
		return mapOAuthAppErr(c, err)
	}

	return c.JSON(http.StatusCreated, oauthAppWithSecretResponse{
		oauthAppResponse: toOAuthAppResponse(app),
		ClientSecret:     clientSecret,
	})
}

// getApp resolves :id as either the app's ID or its client_id. The consent
// page only knows the client_id, while the management pages use the app ID.
func (h *OAuthAppHandler) getApp(ctx context.Context, idOrClientID string) (*domain.OAuthApp, error) {
	app, err := h.apps.GetByID(ctx, idOrClientID)
	if err != nil {
		return nil, err
	}
	if app != nil {
		return app, nil
	}
	return h.apps.GetByClientID(ctx, idOrClientID)
}

func (h *OAuthAppHandler) Get(c echo.Context) error {
	app, err := h.getApp(c.Request().Context(), c.Param("id"))
	if err != nil {
		return mapOAuthAppErr(c, err)
	}
	if app == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	// App metadata (name, homepage, callback URLs) is shown to any signed-in
	// user on the consent screen; the response never includes the secret.
	return c.JSON(http.StatusOK, toOAuthAppResponse(app))
}

func (h *OAuthAppHandler) Update(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	app, err := h.getApp(c.Request().Context(), c.Param("id"))
	if err != nil {
		return mapOAuthAppErr(c, err)
	}
	if app == nil || app.OwnerUserID != userID {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	var req oauthAppUpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid request body"})
	}
	if req.Name != nil {
		if *req.Name == "" {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "name is required"})
		}
		app.Name = *req.Name
	}
	if req.HomepageURL != nil {
		app.HomepageURL = *req.HomepageURL
	}
	if req.CallbackURLs != nil {
		app.RedirectURIs = *req.CallbackURLs
	}

	if err := h.apps.Update(c.Request().Context(), app); err != nil {
		return mapOAuthAppErr(c, err)
	}
	return c.JSON(http.StatusOK, toOAuthAppResponse(app))
}

func (h *OAuthAppHandler) Delete(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	app, err := h.getApp(c.Request().Context(), c.Param("id"))
	if err != nil {
		return mapOAuthAppErr(c, err)
	}
	if app == nil || app.OwnerUserID != userID {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	// Revoke every token the app holds before removing it, so a deleted app
	// cannot keep acting with previously issued credentials. (PATs issued via
	// the OAuth flow are removed by the oauth_application_id FK cascade.)
	if err := h.oauthTokens.RevokeAllByAppID(c.Request().Context(), app.ID, userID); err != nil {
		return mapOAuthAppErr(c, err)
	}
	if err := h.apps.Delete(c.Request().Context(), app.ID, userID); err != nil {
		return mapOAuthAppErr(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *OAuthAppHandler) RegenerateSecret(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	app, err := h.getApp(c.Request().Context(), c.Param("id"))
	if err != nil {
		return mapOAuthAppErr(c, err)
	}
	if app == nil || app.OwnerUserID != userID {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	clientSecret, err := generateOAuthClientCredential(20)
	if err != nil {
		return err
	}
	if err := h.apps.UpdateSecretHash(c.Request().Context(), app.ID, hashOAuthClientSecret(clientSecret)); err != nil {
		return mapOAuthAppErr(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"client_secret": clientSecret})
}

func (h *OAuthAppHandler) ListAuthorizations(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	auths, err := h.authorizations.ListByUser(ctx, userID)
	if err != nil {
		return mapOAuthAppErr(c, err)
	}

	resp := make([]oauthAuthorizationResponse, 0, len(auths))
	for _, auth := range auths {
		entry := oauthAuthorizationResponse{
			OAuthAppID:    auth.OAuthAppID,
			GrantedScopes: auth.GrantedScopes,
			UpdatedAt:     auth.UpdatedAt,
		}
		if entry.GrantedScopes == nil {
			entry.GrantedScopes = []string{}
		}
		if app, err := h.apps.GetByID(ctx, auth.OAuthAppID); err == nil && app != nil {
			entry.AppName = app.Name
			entry.HomepageURL = app.HomepageURL
		}
		resp = append(resp, entry)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *OAuthAppHandler) RevokeAuthorization(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	appID := c.Param("app_id")
	ctx := c.Request().Context()

	// Revoke everything the app holds for this user: OAuth-issued access
	// tokens in both stores, then the authorization record itself.
	if err := h.patRevoker.RevokeAllByOAuthApp(ctx, userID, appID); err != nil {
		return mapOAuthAppErr(c, err)
	}
	if err := h.oauthTokens.RevokeByUserAndApp(ctx, userID, appID); err != nil {
		return mapOAuthAppErr(c, err)
	}
	if err := h.authorizations.Delete(ctx, userID, appID); err != nil {
		return mapOAuthAppErr(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
