package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestGetRepositoryResponseGitHubCompatible(t *testing.T) {
	repoID := uuid.MustParse("00000000-0000-0000-0000-000000000099")
	repos := &listMockRepositoryRepo{
		byLoginName: map[string]*entity.Repository{
			repoLoginKey(listTestOwnerLogin, "demo"): {
				ID:             repoID,
				OrganizationID: listTestUserUUID,
				OwnerID:        listTestUserUUID,
				OwnerLogin:     listTestOwnerLogin,
				Name:           "demo",
				Visibility:     entity.VisibilityPublic,
				DefaultBranch:  "main",
				CreatedAt:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	e := newRepositoryHandlerEcho(t, repos, &listMockOrgRepo{}, &listMockAuditLogRepo{}, authMiddleware(listTestUserID))

	req := httptest.NewRequest(http.MethodGet, "/repos/"+listTestOwnerLogin+"/demo", nil)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["id"].(float64); !ok {
		t.Fatalf("id = %v (%T), want numeric", resp["id"], resp["id"])
	}
	cloneURL, ok := resp["clone_url"].(string)
	if !ok || cloneURL == "" {
		t.Fatalf("clone_url = %v, want non-empty string", resp["clone_url"])
	}
	if cloneURL != "https://git.example.com/"+listTestOwnerLogin+"/demo.git" {
		t.Fatalf("clone_url = %q, want https://git.example.com/%s/demo.git", cloneURL, listTestOwnerLogin)
	}
}
