package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/repository"
)

var ErrInvalidCode = errors.New("invalid code")

type OAuthTokenInput struct {
	Code string
	// ClientID and ClientSecret authenticate the application performing the
	// exchange; a stolen code alone must not be enough to obtain a token.
	ClientID     string
	ClientSecret string
}

type OAuthTokenOutput struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type OAuthTokenUsecase struct {
	codes          OAuthCodeStore
	issue          *IssuePATUsecase
	apps           repository.IOAuthAppRepository
	authorizations repository.IOAuthAuthorizationRepository
}

func NewOAuthTokenUsecase(
	codes OAuthCodeStore,
	issue *IssuePATUsecase,
	apps repository.IOAuthAppRepository,
	authorizations repository.IOAuthAuthorizationRepository,
) *OAuthTokenUsecase {
	return &OAuthTokenUsecase{codes: codes, issue: issue, apps: apps, authorizations: authorizations}
}

func (u *OAuthTokenUsecase) Execute(ctx context.Context, input OAuthTokenInput) (*OAuthTokenOutput, error) {
	code := strings.TrimSpace(input.Code)
	if code == "" {
		return nil, ErrInvalidCode
	}

	raw, err := u.codes.GetDel(ctx, oauthCodeKey(code))
	if err != nil || raw == "" {
		return nil, ErrInvalidCode
	}

	var payload OAuthCodePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, ErrInvalidCode
	}

	// Authenticate the exchanging application: the client_id must match the
	// one the code was issued for, and the client_secret must match the app's
	// stored secret hash.
	if input.ClientID != payload.ClientID {
		return nil, ErrInvalidClient
	}
	app, err := u.apps.GetByClientID(ctx, payload.ClientID)
	if err != nil || app == nil {
		return nil, ErrInvalidClient
	}
	secretHash := hashToken(strings.TrimSpace(input.ClientSecret))
	if subtle.ConstantTimeCompare([]byte(secretHash), []byte(app.ClientSecretHash)) != 1 {
		return nil, ErrInvalidClient
	}

	scopes := splitScopes(payload.Scope)
	out, err := u.issue.Execute(ctx, IssuePATInput{
		UserID:     payload.UserID,
		Note:       fmt.Sprintf("OAuth: %s", app.Name),
		OAuthAppID: app.ID,
		Scopes:     scopes,
	})
	if err != nil {
		return nil, err
	}

	// Record (or refresh) the user's authorization of this app so it shows up
	// under the user's authorized applications and can be revoked there.
	if err := u.authorizations.Upsert(ctx, &domain.OAuthAuthorization{
		OAuthAppID:    app.ID,
		UserID:        payload.UserID,
		GrantedScopes: scopes,
	}); err != nil {
		return nil, err
	}

	return &OAuthTokenOutput{
		AccessToken: out.Token,
		Scope:       payload.Scope,
		TokenType:   "bearer",
	}, nil
}

func splitScopes(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		scope := strings.TrimSpace(part)
		if scope != "" {
			scopes = append(scopes, scope)
		}
	}
	return scopes
}
