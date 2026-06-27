package oauth

import (
	"context"
	"os"
	"strings"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/repository"
)

type EnvAppRepository struct {
	apps map[string]*domain.OAuthApp
}

func NewEnvAppRepository() repository.IOAuthAppRepository {
	clientID := strings.TrimSpace(os.Getenv("OAUTH_CLIENT_ID"))
	if clientID == "" {
		return &EnvAppRepository{apps: map[string]*domain.OAuthApp{}}
	}

	redirectURIs := make([]string, 0)
	for _, uri := range strings.Split(os.Getenv("OAUTH_REDIRECT_URIS"), ",") {
		if trimmed := strings.TrimSpace(uri); trimmed != "" {
			redirectURIs = append(redirectURIs, trimmed)
		}
	}

	return &EnvAppRepository{
		apps: map[string]*domain.OAuthApp{
			clientID: {
				ClientID:     clientID,
				RedirectURIs: redirectURIs,
			},
		},
	}
}

func (r *EnvAppRepository) GetByClientID(_ context.Context, clientID string) (*domain.OAuthApp, error) {
	app, ok := r.apps[clientID]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return app, nil
}
