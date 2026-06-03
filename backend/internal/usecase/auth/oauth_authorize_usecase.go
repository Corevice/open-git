package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Corevice/open-git/backend/internal/repository"
)

const oauthCodeTTL = 10 * time.Minute

var (
	ErrInvalidClient       = errors.New("invalid client")
	ErrRedirectURIMismatch = errors.New("redirect_uri mismatch")
	ErrMissingState        = errors.New("state is required")
)

type OAuthCodeStore interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	GetDel(ctx context.Context, key string) (string, error)
}

type OAuthCodePayload struct {
	UserID      int64  `json:"user_id"`
	ClientID    string `json:"client_id"`
	Scope       string `json:"scope"`
	RedirectURI string `json:"redirect_uri"`
}

type OAuthAuthorizeInput struct {
	UserID      int64
	ClientID    string
	RedirectURI string
	Scope       string
	State       string
}

type OAuthAuthorizeUsecase struct {
	apps  repository.IOAuthAppRepository
	codes OAuthCodeStore
}

func NewOAuthAuthorizeUsecase(apps repository.IOAuthAppRepository, codes OAuthCodeStore) *OAuthAuthorizeUsecase {
	return &OAuthAuthorizeUsecase{apps: apps, codes: codes}
}

func (u *OAuthAuthorizeUsecase) Execute(ctx context.Context, input OAuthAuthorizeInput) (string, error) {
	if input.State == "" {
		return "", ErrMissingState
	}

	app, err := u.apps.GetByClientID(ctx, input.ClientID)
	if err != nil || app == nil {
		return "", ErrInvalidClient
	}

	if !redirectURIAllowed(app.RedirectURIs, input.RedirectURI) {
		return "", ErrRedirectURIMismatch
	}

	code, err := generateOAuthCode()
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(OAuthCodePayload{
		UserID:      input.UserID,
		ClientID:    input.ClientID,
		Scope:       input.Scope,
		RedirectURI: input.RedirectURI,
	})
	if err != nil {
		return "", err
	}

	key := oauthCodeKey(code)
	if err := u.codes.Set(ctx, key, string(payload), oauthCodeTTL); err != nil {
		return "", err
	}

	return code, nil
}

func redirectURIAllowed(allowed []string, redirectURI string) bool {
	for _, uri := range allowed {
		if uri == redirectURI {
			return true
		}
	}
	return false
}

func generateOAuthCode() (string, error) {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func oauthCodeKey(code string) string {
	return fmt.Sprintf("oauth:code:%s", code)
}
