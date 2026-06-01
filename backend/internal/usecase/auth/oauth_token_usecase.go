package auth

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

var ErrInvalidCode = errors.New("invalid code")

type OAuthTokenInput struct {
	Code string
}

type OAuthTokenOutput struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type OAuthTokenUsecase struct {
	codes OAuthCodeStore
	issue *IssuePATUsecase
}

func NewOAuthTokenUsecase(codes OAuthCodeStore, issue *IssuePATUsecase) *OAuthTokenUsecase {
	return &OAuthTokenUsecase{codes: codes, issue: issue}
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

	scopes := splitScopes(payload.Scope)
	out, err := u.issue.Execute(ctx, IssuePATInput{
		UserID: payload.UserID,
		Scopes: scopes,
	})
	if err != nil {
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
