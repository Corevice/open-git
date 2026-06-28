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

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
)

var (
	collabTestRepoID    = uuid.MustParse("00000000-0000-0000-0000-000000000101")
	collabTestOwnerID   = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	collabTestUserID    = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	collabTestOtherID   = uuid.MustParse("00000000-0000-0000-0000-000000000007")
	collabTestOwnerInt  = middleware.UUIDToInt64(collabTestOwnerID)
	collabTestOtherInt  = middleware.UUIDToInt64(collabTestOtherID)
)

type mockCollabRepoRepo struct {
	repository *entity.Repository
}

func (m *mockCollabRepoRepo) Create(context.Context, *entity.Repository) error { return nil }

func (m *mockCollabRepoRepo) GetByOwnerAndName(context.Context, uuid.UUID, string) (*entity.Repository, error) {
	return nil, nil
}

func (m *mockCollabRepoRepo) GetByOwnerLoginAndName(_ context.Context, ownerLogin, name string) (*entity.Repository, error) {
	if m.repository != nil && m.repository.OwnerLogin == ownerLogin && m.repository.Name == name {
		copyRepo := *m.repository
		return &copyRepo, nil
	}
	return nil, nil
}

func (m *mockCollabRepoRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *mockCollabRepoRepo) CountByOrg(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockCollabRepoRepo) ListByOwner(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *mockCollabRepoRepo) CountByOwner(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockCollabRepoRepo) UpdateVisibility(context.Context, uuid.UUID, string) error { return nil }

func (m *mockCollabRepoRepo) UpdateName(context.Context, uuid.UUID, string) error { return nil }

func (m *mockCollabRepoRepo) UpdateDefaultBranch(context.Context, uuid.UUID, string) error { return nil }

func (m *mockCollabRepoRepo) Delete(context.Context, uuid.UUID) error { return nil }

type mockCollabUserRepo struct {
	users map[string]*entity.User
	byID  map[uuid.UUID]*entity.User
}

func (m *mockCollabUserRepo) Create(context.Context, *entity.User) error { return nil }

func (m *mockCollabUserRepo) Update(context.Context, *entity.User) error { return nil }

func (m *mockCollabUserRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	if m.byID == nil {
		return nil, domain.ErrNotFound
	}
	user, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func (m *mockCollabUserRepo) GetByLogin(_ context.Context, login string) (*entity.User, error) {
	if m.users == nil {
		return nil, domain.ErrNotFound
	}
	user, ok := m.users[login]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func (m *mockCollabUserRepo) GetByEmail(context.Context, string) (*entity.User, error) {
	return nil, domain.ErrNotFound
}

type mockCollaboratorRepo struct {
	entries map[uuid.UUID]map[uuid.UUID]string
}

func (m *mockCollaboratorRepo) AddCollaborator(_ context.Context, repoID, userID uuid.UUID, permission string) error {
	if m.entries == nil {
		m.entries = map[uuid.UUID]map[uuid.UUID]string{}
	}
	if m.entries[repoID] == nil {
		m.entries[repoID] = map[uuid.UUID]string{}
	}
	m.entries[repoID][userID] = permission
	return nil
}

func (m *mockCollaboratorRepo) RemoveCollaborator(_ context.Context, repoID, userID uuid.UUID) error {
	if m.entries == nil {
		return nil
	}
	delete(m.entries[repoID], userID)
	return nil
}

func (m *mockCollaboratorRepo) GetPermission(_ context.Context, repoID, userID uuid.UUID) (string, error) {
	if m.entries == nil {
		return "", nil
	}
	return m.entries[repoID][userID], nil
}

func (m *mockCollaboratorRepo) ListCollaborators(_ context.Context, repoID uuid.UUID) ([]*entity.RepositoryCollaborator, error) {
	users := m.entries[repoID]
	result := make([]*entity.RepositoryCollaborator, 0, len(users))
	for userID, permission := range users {
		result = append(result, &entity.RepositoryCollaborator{
			RepositoryID: repoID,
			UserID:       userID,
			Permission:   permission,
		})
	}
	return result, nil
}

func collabTestRepository() *entity.Repository {
	return &entity.Repository{
		ID:         collabTestRepoID,
		OwnerID:    collabTestOwnerID,
		OwnerLogin: "alice",
		Name:       "demo",
	}
}

func collabTestUsers() *mockCollabUserRepo {
	bob := &entity.User{ID: collabTestUserID, Login: "bob"}
	return &mockCollabUserRepo{
		users: map[string]*entity.User{"bob": bob},
		byID:  map[uuid.UUID]*entity.User{collabTestUserID: bob},
	}
}

func newCollaboratorHandlerEcho(t *testing.T, ownerInt int64, users *mockCollabUserRepo, collaborators *mockCollaboratorRepo) *echo.Echo {
	t.Helper()

	h := handler.NewCollaboratorHandler(
		nil,
		&mockCollabRepoRepo{repository: collabTestRepository()},
		collaborators,
		users,
	)

	e := echo.New()
	g := e.Group("")
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, ownerInt, []string{"repo"})
			return next(c)
		}
	}
	h.RegisterRoutes(g, auth)
	return e
}

func TestCollaboratorAdd_204(t *testing.T) {
	collaborators := &mockCollaboratorRepo{}
	e := newCollaboratorHandlerEcho(t, collabTestOwnerInt, collabTestUsers(), collaborators)

	req := httptest.NewRequest(http.MethodPut, "/repos/alice/demo/collaborators/bob", bytes.NewReader([]byte(`{}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if got := collaborators.entries[collabTestRepoID][collabTestUserID]; got != entity.CollaboratorPermWrite {
		t.Fatalf("permission = %q, want %q", got, entity.CollaboratorPermWrite)
	}
}

func TestCollaboratorAdd_422_badPermission(t *testing.T) {
	e := newCollaboratorHandlerEcho(t, collabTestOwnerInt, collabTestUsers(), &mockCollaboratorRepo{})

	req := httptest.NewRequest(http.MethodPut, "/repos/alice/demo/collaborators/bob", bytes.NewReader([]byte(`{"permission":"superuser"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestCollaboratorAdd_403_notOwner(t *testing.T) {
	e := newCollaboratorHandlerEcho(t, collabTestOtherInt, collabTestUsers(), &mockCollaboratorRepo{})

	req := httptest.NewRequest(http.MethodPut, "/repos/alice/demo/collaborators/bob", bytes.NewReader([]byte(`{"permission":"read"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestCollaboratorRemove_204(t *testing.T) {
	collaborators := &mockCollaboratorRepo{
		entries: map[uuid.UUID]map[uuid.UUID]string{
			collabTestRepoID: {collabTestUserID: entity.CollaboratorPermRead},
		},
	}
	e := newCollaboratorHandlerEcho(t, collabTestOwnerInt, collabTestUsers(), collaborators)

	req := httptest.NewRequest(http.MethodDelete, "/repos/alice/demo/collaborators/bob", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if len(collaborators.entries[collabTestRepoID]) != 0 {
		t.Fatalf("expected collaborator to be removed")
	}
}

func TestCollaboratorList_200(t *testing.T) {
	collaborators := &mockCollaboratorRepo{
		entries: map[uuid.UUID]map[uuid.UUID]string{
			collabTestRepoID: {collabTestUserID: entity.CollaboratorPermWrite},
		},
	}
	e := newCollaboratorHandlerEcho(t, collabTestOwnerInt, collabTestUsers(), collaborators)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/collaborators", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp []struct {
		Login       string `json:"login"`
		Permissions struct {
			Pull  bool `json:"pull"`
			Push  bool `json:"push"`
			Admin bool `json:"admin"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("len(resp) = %d, want 1", len(resp))
	}
	if resp[0].Login != "bob" {
		t.Fatalf("login = %q, want bob", resp[0].Login)
	}
	if !resp[0].Permissions.Pull || !resp[0].Permissions.Push || resp[0].Permissions.Admin {
		t.Fatalf("permissions = %+v, want pull=true push=true admin=false", resp[0].Permissions)
	}
}

var (
	_ repo.IRepositoryRepository            = (*mockCollabRepoRepo)(nil)
	_ repo.IRepositoryCollaboratorRepository = (*mockCollaboratorRepo)(nil)
)
