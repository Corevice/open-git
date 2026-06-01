package auth_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/usecase/auth"
)

type mockOAuthAppRepo struct {
	apps map[string]*domain.OAuthApp
}

func (m *mockOAuthAppRepo) GetByClientID(_ context.Context, clientID string) (*domain.OAuthApp, error) {
	if m.apps == nil {
		return nil, nil
	}
	return m.apps[clientID], nil
}

type mockOAuthCodeStore struct {
	values map[string]string
}

func (m *mockOAuthCodeStore) Set(_ context.Context, key, value string, _ time.Duration) error {
	if m.values == nil {
		m.values = map[string]string{}
	}
	m.values[key] = value
	return nil
}

func (m *mockOAuthCodeStore) GetDel(_ context.Context, key string) (string, error) {
	value, ok := m.values[key]
	if !ok {
		return "", nil
	}
	delete(m.values, key)
	return value, nil
}

func TestRedirectURIMismatch(t *testing.T) {
	apps := &mockOAuthAppRepo{
		apps: map[string]*domain.OAuthApp{
			"client-1": {
				ClientID:     "client-1",
				RedirectURIs: []string{"https://example.com/callback"},
			},
		},
	}
	uc := auth.NewOAuthAuthorizeUsecase(apps, &mockOAuthCodeStore{})

	_, err := uc.Execute(context.Background(), auth.OAuthAuthorizeInput{
		UserID:      1,
		ClientID:    "client-1",
		RedirectURI: "https://example.com/callback/",
		Scope:       "read",
		State:       "state-123",
	})
	if err == nil {
		t.Fatal("expected redirect_uri mismatch error")
	}
	if err != auth.ErrRedirectURIMismatch {
		t.Fatalf("expected ErrRedirectURIMismatch, got %v", err)
	}
}

func TestMissingState(t *testing.T) {
	apps := &mockOAuthAppRepo{
		apps: map[string]*domain.OAuthApp{
			"client-1": {
				ClientID:     "client-1",
				RedirectURIs: []string{"https://example.com/callback"},
			},
		},
	}
	uc := auth.NewOAuthAuthorizeUsecase(apps, &mockOAuthCodeStore{})

	_, err := uc.Execute(context.Background(), auth.OAuthAuthorizeInput{
		UserID:      1,
		ClientID:    "client-1",
		RedirectURI: "https://example.com/callback",
		Scope:       "read",
		State:       "",
	})
	if err == nil {
		t.Fatal("expected missing state error")
	}
	if err != auth.ErrMissingState {
		t.Fatalf("expected ErrMissingState, got %v", err)
	}
}

func TestValidAuthorize(t *testing.T) {
	apps := &mockOAuthAppRepo{
		apps: map[string]*domain.OAuthApp{
			"client-1": {
				ClientID:     "client-1",
				RedirectURIs: []string{"https://example.com/callback"},
			},
		},
	}
	store := &mockOAuthCodeStore{}
	uc := auth.NewOAuthAuthorizeUsecase(apps, store)

	code, err := uc.Execute(context.Background(), auth.OAuthAuthorizeInput{
		UserID:      1,
		ClientID:    "client-1",
		RedirectURI: "https://example.com/callback",
		Scope:       "read",
		State:       "state-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code) != 20 {
		t.Fatalf("expected 20-char code, got %q", code)
	}

	raw, ok := store.values["oauth:code:"+code]
	if !ok {
		t.Fatal("expected code to be stored in redis mock")
	}

	var payload auth.OAuthCodePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.UserID != 1 {
		t.Fatalf("expected user_id 1, got %d", payload.UserID)
	}
	if payload.ClientID != "client-1" {
		t.Fatalf("expected client_id client-1, got %q", payload.ClientID)
	}
}
