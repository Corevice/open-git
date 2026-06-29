package handler_test

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/handler"
)

func TestValidateOwnerRepo(t *testing.T) {
	tests := []struct {
		name    string
		owner   string
		repo    string
		wantErr bool
	}{
		{name: "valid my-org", owner: "my-org", repo: "demo"},
		{name: "valid my.repo", owner: "my", repo: "my.repo"},
		{name: "valid My_Repo", owner: "My_Repo", repo: "test"},
		{name: "dotdot in owner", owner: "..", repo: "demo", wantErr: true},
		{name: "leading dot in repo", owner: "org", repo: ".hidden", wantErr: true},
		{name: "trailing dot in owner", owner: "org.", repo: "repo", wantErr: true},
		{name: "slash in owner", owner: "org/foo", repo: "repo", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.ValidateOwnerRepo(tt.owner, tt.repo)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateOwnerRepo(%q, %q) error = %v, wantErr = %v", tt.owner, tt.repo, err, tt.wantErr)
			}
			if err == nil {
				return
			}

			he, ok := err.(*echo.HTTPError)
			if !ok {
				t.Fatalf("expected *echo.HTTPError, got %T", err)
			}
			if he.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", he.Code, http.StatusBadRequest)
			}
			msg, ok := he.Message.(map[string]string)
			if !ok || msg["message"] != "Invalid owner or repository name" {
				t.Fatalf("message = %v, want Invalid owner or repository name", he.Message)
			}
		})
	}
}
