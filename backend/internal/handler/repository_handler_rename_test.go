package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
)

var (
	renameTestOwnerID   = uuid.MustParse("00000000-0000-0000-0000-000000000007")
	renameTestOtherID   = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	renameTestOwnerLogin = "testuser"
)

type renameMockRepositoryRepo struct {
	byLoginName map[string]*entity.Repository
	byOwnerName map[uuid.UUID]map[string]*entity.Repository
}

func renameRepoKey(ownerLogin, name string) string {
	return ownerLogin + ":" + name
}

func (m *renameMockRepositoryRepo) Create(_ context.Context, repo *entity.Repository) error { return nil }

func (m *renameMockRepositoryRepo) GetByOwnerAndName(_ context.Context, ownerID uuid.UUID, name string) (*entity.Repository, error) {
	if m.byOwnerName == nil {
		return nil, nil
	}
	ownerRepos, ok := m.byOwnerName[ownerID]
	if !ok {
		return nil, nil
	}
	return ownerRepos[name], nil
}

func (m *renameMockRepositoryRepo) GetByOwnerLoginAndName(_ context.Context, ownerLogin, name string) (*entity.Repository, error) {
	if m.byLoginName == nil {
		return nil, nil
	}
	return m.byLoginName[renameRepoKey(ownerLogin, name)], nil
}

func (m *renameMockRepositoryRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *renameMockRepositoryRepo) CountByOrg(context.Context, uuid.UUID) (int, error) { return 0, nil }

func (m *renameMockRepositoryRepo) ListByOwner(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *renameMockRepositoryRepo) CountByOwner(context.Context, uuid.UUID) (int, error) { return 0, nil }

func (m *renameMockRepositoryRepo) UpdateVisibility(context.Context, uuid.UUID, string) error { return nil }

func (m *renameMockRepositoryRepo) UpdateDefaultBranch(_ context.Context, id uuid.UUID, branch string) error {
	for _, repo := range m.byLoginName {
		if repo.ID == id {
			repo.DefaultBranch = branch
			return nil
		}
	}
	return nil
}

func (m *renameMockRepositoryRepo) UpdateName(_ context.Context, id uuid.UUID, newName string) error {
	for key, repo := range m.byLoginName {
		if repo.ID != id {
			continue
		}
		oldName := repo.Name
		ownerLogin := repo.OwnerLogin
		delete(m.byLoginName, key)
		repo.Name = newName
		m.byLoginName[renameRepoKey(ownerLogin, newName)] = repo
		if m.byOwnerName != nil {
			if ownerRepos, ok := m.byOwnerName[repo.OwnerID]; ok {
				delete(ownerRepos, oldName)
				ownerRepos[newName] = repo
			}
		}
		return nil
	}
	return nil
}

func (m *renameMockRepositoryRepo) Delete(context.Context, uuid.UUID) error { return nil }

func newRenameTestEcho(t *testing.T, repos *renameMockRepositoryRepo, userID int64) *echo.Echo {
	t.Helper()

	memberships := &listMockMembershipRepo{}
	listRepos := repoUC.NewListRepositoriesUsecase(repos, memberships, nil)
	create := repoUC.NewCreateRepositoryUsecase(repos)
	get := repoUC.NewGetRepositoryUsecase(repos, nil, memberships)
	listAuditLogs := repoUC.NewListAuditLogsUsecase(&mockListAuditLogsUsecase{})

	h := handler.NewRepositoryHandler(create, get, listRepos, repos, &listMockOrgRepo{}, &listMockAuditLogRepo{}, listAuditLogs)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, authMiddleware(userID))
	return e
}

func seedRenameRepo(ownerID uuid.UUID, ownerLogin, name string) *entity.Repository {
	repo := &entity.Repository{
		ID:             uuid.New(),
		OrganizationID: ownerID,
		OwnerID:        ownerID,
		OwnerLogin:     ownerLogin,
		Name:           name,
		Visibility:     entity.VisibilityPublic,
		DefaultBranch:  "main",
	}
	return repo
}

func TestUpdateRepositoryRenameOK(t *testing.T) {
	repo := seedRenameRepo(renameTestOwnerID, renameTestOwnerLogin, "old-name")
	repos := &renameMockRepositoryRepo{
		byLoginName: map[string]*entity.Repository{
			renameRepoKey(renameTestOwnerLogin, "old-name"): repo,
		},
		byOwnerName: map[uuid.UUID]map[string]*entity.Repository{
			renameTestOwnerID: {"old-name": repo},
		},
	}
	e := newRenameTestEcho(t, repos, listTestUserID)

	body, err := json.Marshal(map[string]string{"name": "new-name"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/repos/testuser/old-name", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["name"] != "new-name" {
		t.Fatalf("name = %v, want new-name", resp["name"])
	}
}

func TestUpdateRepositoryRenameInvalidName(t *testing.T) {
	repo := seedRenameRepo(renameTestOwnerID, renameTestOwnerLogin, "old-name")
	repos := &renameMockRepositoryRepo{
		byLoginName: map[string]*entity.Repository{
			renameRepoKey(renameTestOwnerLogin, "old-name"): repo,
		},
		byOwnerName: map[uuid.UUID]map[string]*entity.Repository{
			renameTestOwnerID: {"old-name": repo},
		},
	}
	e := newRenameTestEcho(t, repos, listTestUserID)

	body, err := json.Marshal(map[string]string{"name": "invalid name!"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/repos/testuser/old-name", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestUpdateRepositoryDefaultBranchOK(t *testing.T) {
	repo := seedRenameRepo(renameTestOwnerID, "alice", "myrepo")
	repos := &renameMockRepositoryRepo{
		byLoginName: map[string]*entity.Repository{
			renameRepoKey("alice", "myrepo"): repo,
		},
		byOwnerName: map[uuid.UUID]map[string]*entity.Repository{
			renameTestOwnerID: {"myrepo": repo},
		},
	}
	e := newRenameTestEcho(t, repos, listTestUserID)

	body, err := json.Marshal(map[string]string{"default_branch": "dev"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/repos/alice/myrepo", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["default_branch"] != "dev" {
		t.Fatalf("default_branch = %v, want dev", resp["default_branch"])
	}
}

func TestUpdateRepositoryRenameNonOwner(t *testing.T) {
	repo := seedRenameRepo(renameTestOtherID, renameTestOwnerLogin, "old-name")
	repos := &renameMockRepositoryRepo{
		byLoginName: map[string]*entity.Repository{
			renameRepoKey(renameTestOwnerLogin, "old-name"): repo,
		},
		byOwnerName: map[uuid.UUID]map[string]*entity.Repository{
			renameTestOtherID: {"old-name": repo},
		},
	}
	e := newRenameTestEcho(t, repos, listTestUserID)

	body, err := json.Marshal(map[string]string{"name": "new-name"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/repos/testuser/old-name", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
