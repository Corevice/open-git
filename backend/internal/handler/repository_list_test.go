package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
)

var (
	listTestUserID     = int64(7)
	listTestUserUUID   = uuid.MustParse("00000000-0000-0000-0000-000000000007")
	listTestOwnerLogin = "testuser"
	listTestOrgID      = int64(42)
	listTestOrgUUID    = uuid.MustParse("00000000-0000-0000-0000-00000000002a")
)

type listMockRepositoryRepo struct {
	byOwner      map[uuid.UUID][]*entity.Repository
	byOrg        map[uuid.UUID][]*entity.Repository
	byLoginName  map[string]*entity.Repository
}

func repoLoginKey(ownerLogin, name string) string {
	return ownerLogin + ":" + name
}

func (m *listMockRepositoryRepo) Create(_ context.Context, repo *entity.Repository) error {
	if m.byOwner == nil {
		m.byOwner = map[uuid.UUID][]*entity.Repository{}
	}
	if m.byLoginName == nil {
		m.byLoginName = map[string]*entity.Repository{}
	}
	if repo.ID == uuid.Nil {
		repo.ID = uuid.New()
	}
	m.byOwner[repo.OwnerID] = append(m.byOwner[repo.OwnerID], repo)
	m.byLoginName[repoLoginKey(listTestOwnerLogin, repo.Name)] = repo
	return nil
}

func (m *listMockRepositoryRepo) GetByOwnerAndName(_ context.Context, ownerID uuid.UUID, name string) (*entity.Repository, error) {
	for _, r := range m.byOwner[ownerID] {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *listMockRepositoryRepo) GetByOwnerLoginAndName(_ context.Context, ownerLogin, name string) (*entity.Repository, error) {
	if m.byLoginName == nil {
		return nil, nil
	}
	return m.byLoginName[repoLoginKey(ownerLogin, name)], nil
}

func (m *listMockRepositoryRepo) ListByOrg(_ context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.Repository, error) {
	repos := m.byOrg[orgID]
	return paginateRepos(repos, page, perPage), nil
}

func (m *listMockRepositoryRepo) CountByOrg(_ context.Context, orgID uuid.UUID) (int, error) {
	return len(m.byOrg[orgID]), nil
}

func (m *listMockRepositoryRepo) ListByOwner(_ context.Context, ownerID uuid.UUID, page, perPage int) ([]*entity.Repository, error) {
	repos := m.byOwner[ownerID]
	return paginateRepos(repos, page, perPage), nil
}

func (m *listMockRepositoryRepo) CountByOwner(_ context.Context, ownerID uuid.UUID) (int, error) {
	return len(m.byOwner[ownerID]), nil
}

func (m *listMockRepositoryRepo) UpdateVisibility(context.Context, uuid.UUID, string) error { return nil }

func (m *listMockRepositoryRepo) Delete(context.Context, uuid.UUID) error { return nil }

func paginateRepos(repos []*entity.Repository, page, perPage int) []*entity.Repository {
	if len(repos) == 0 {
		return []*entity.Repository{}
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	start := (page - 1) * perPage
	if start >= len(repos) {
		return []*entity.Repository{}
	}
	end := start + perPage
	if end > len(repos) {
		end = len(repos)
	}
	return repos[start:end]
}

type listMockMembershipRepo struct{}

func (m *listMockMembershipRepo) HasReadAccess(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}

func (m *listMockMembershipRepo) HasWriteAccess(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}

type listMockOrgRepo struct {
	byLogin map[string]*domain.Organization
	roles   map[int64]map[int64]string
}

func (m *listMockOrgRepo) GetByLogin(_ context.Context, login string) (*domain.Organization, error) {
	if m.byLogin == nil {
		return nil, nil
	}
	return m.byLogin[login], nil
}

func (m *listMockOrgRepo) ListByUserID(context.Context, int64) ([]*domain.Organization, error) {
	return nil, nil
}

func (m *listMockOrgRepo) GetMemberRole(_ context.Context, orgID, userID int64) (string, error) {
	if m.roles == nil {
		return "", nil
	}
	if userRoles, ok := m.roles[orgID]; ok {
		return userRoles[userID], nil
	}
	return "", nil
}

type listMockAuditLogRepo struct {
	records []auditLogRecord
}

type auditLogRecord struct {
	orgID      uuid.UUID
	actorID    uuid.UUID
	action     string
	targetType string
	targetID   uuid.UUID
	metadata   map[string]any
}

func (m *listMockAuditLogRepo) Record(_ context.Context, orgID, actorID uuid.UUID, action, targetType string, targetID uuid.UUID, metadata map[string]any) error {
	m.records = append(m.records, auditLogRecord{
		orgID:      orgID,
		actorID:    actorID,
		action:     action,
		targetType: targetType,
		targetID:   targetID,
		metadata:   metadata,
	})
	return nil
}

// listOwnerResolverStub resolves any owner id to a fixed login so the create
// usecase can compute a git path during handler tests.
type listOwnerResolverStub struct{}

func (listOwnerResolverStub) GetByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	return &entity.User{ID: id, Login: listTestOwnerLogin}, nil
}

func newRepositoryHandlerEcho(
	t *testing.T,
	repos *listMockRepositoryRepo,
	orgs *listMockOrgRepo,
	auditLog *listMockAuditLogRepo,
	auth echo.MiddlewareFunc,
) *echo.Echo {
	t.Helper()

	memberships := &listMockMembershipRepo{}
	listRepos := repoUC.NewListRepositoriesUsecase(repos, memberships, nil)
	create := repoUC.NewCreateRepositoryUsecase(
		repos,
		repoUC.WithGitDataRoot(t.TempDir()),
		repoUC.WithOwnerLoginResolver(listOwnerResolverStub{}),
	)
	get := repoUC.NewGetRepositoryUsecase(repos, nil, memberships)
	listAuditLogs := repoUC.NewListAuditLogsUsecase(&mockListAuditLogsUsecase{})

	h := handler.NewRepositoryHandler(create, get, listRepos, repos, orgs, nil, auditLog, listAuditLogs)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, auth)
	return e
}

func authMiddleware(userID int64) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user_id", userID)
			c.Set("scopes", []string{"repo"})
			return next(c)
		}
	}
}

func makeUserRepos(count int) []*entity.Repository {
	repos := make([]*entity.Repository, 0, count)
	for i := 0; i < count; i++ {
		repos = append(repos, &entity.Repository{
			ID:             uuid.New(),
			OrganizationID: listTestUserUUID,
			OwnerID:        listTestUserUUID,
			Name:           fmt.Sprintf("repo-%d", i),
			Visibility:     entity.VisibilityPublic,
			DefaultBranch:  "main",
		})
	}
	return repos
}

func TestListUserReposOK(t *testing.T) {
	repos := &listMockRepositoryRepo{
		byOwner: map[uuid.UUID][]*entity.Repository{
			listTestUserUUID: makeUserRepos(35),
		},
	}
	e := newRepositoryHandlerEcho(t, repos, &listMockOrgRepo{}, &listMockAuditLogRepo{}, authMiddleware(listTestUserID))

	req := httptest.NewRequest(http.MethodGet, "/user/repos?per_page=30&page=1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 30 {
		t.Fatalf("len(repos) = %d, want 30", len(resp))
	}

	link := rec.Header().Get("Link")
	if link == "" {
		t.Fatal("expected Link header")
	}
	if !strings.Contains(link, `rel="next"`) {
		t.Fatalf("Link header = %q, want rel=next", link)
	}
}

func TestListUserReposNegativePerPageClampsToOne(t *testing.T) {
	repos := &listMockRepositoryRepo{
		byOwner: map[uuid.UUID][]*entity.Repository{
			listTestUserUUID: makeUserRepos(1),
		},
	}
	e := newRepositoryHandlerEcho(t, repos, &listMockOrgRepo{}, &listMockAuditLogRepo{}, authMiddleware(listTestUserID))

	req := httptest.NewRequest(http.MethodGet, "/user/repos?per_page=-1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestListOrgReposNonMember(t *testing.T) {
	repos := &listMockRepositoryRepo{
		byOrg: map[uuid.UUID][]*entity.Repository{
			listTestOrgUUID: makeUserRepos(2),
		},
	}
	orgs := &listMockOrgRepo{
		byLogin: map[string]*domain.Organization{
			"acme": {ID: listTestOrgID, Login: "acme", Name: "Acme"},
		},
		roles: map[int64]map[int64]string{},
	}
	e := newRepositoryHandlerEcho(t, repos, orgs, &listMockAuditLogRepo{}, authMiddleware(listTestUserID))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/repos", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestListOrgReposOK(t *testing.T) {
	repos := &listMockRepositoryRepo{
		byOrg: map[uuid.UUID][]*entity.Repository{
			listTestOrgUUID: makeUserRepos(2),
		},
	}
	orgs := &listMockOrgRepo{
		byLogin: map[string]*domain.Organization{
			"acme": {ID: listTestOrgID, Login: "acme", Name: "Acme"},
		},
		roles: map[int64]map[int64]string{
			listTestOrgID: {listTestUserID: "member"},
		},
	}
	e := newRepositoryHandlerEcho(t, repos, orgs, &listMockAuditLogRepo{}, authMiddleware(listTestUserID))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/repos", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("len(repos) = %d, want 2", len(resp))
	}
}

func TestCreateRepoAuditLog(t *testing.T) {
	repos := &listMockRepositoryRepo{
		byOwner: map[uuid.UUID][]*entity.Repository{},
	}
	auditLog := &listMockAuditLogRepo{}
	e := newRepositoryHandlerEcho(t, repos, &listMockOrgRepo{}, auditLog, authMiddleware(listTestUserID))

	body := bytes.NewBufferString(`{"name":"new-repo","private":false}`)
	req := httptest.NewRequest(http.MethodPost, "/user/repos", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	if len(auditLog.records) != 1 {
		t.Fatalf("audit records = %d, want 1", len(auditLog.records))
	}
	if auditLog.records[0].action != "repo.create" {
		t.Fatalf("action = %q, want repo.create", auditLog.records[0].action)
	}
	if auditLog.records[0].targetType != "Repository" {
		t.Fatalf("targetType = %q, want Repository", auditLog.records[0].targetType)
	}
}

func TestDeleteRepoAuditLog(t *testing.T) {
	repoID := uuid.New()
	repos := &listMockRepositoryRepo{
		byOwner: map[uuid.UUID][]*entity.Repository{
			listTestUserUUID: {
				{
					ID:             repoID,
					OrganizationID: listTestUserUUID,
					OwnerID:        listTestUserUUID,
					Name:           "delete-me",
					Visibility:     entity.VisibilityPublic,
					DefaultBranch:  "main",
				},
			},
		},
		byLoginName: map[string]*entity.Repository{
			repoLoginKey(listTestOwnerLogin, "delete-me"): {
				ID:             repoID,
				OrganizationID: listTestUserUUID,
				OwnerID:        listTestUserUUID,
				Name:           "delete-me",
				Visibility:     entity.VisibilityPublic,
				DefaultBranch:  "main",
			},
		},
	}
	auditLog := &listMockAuditLogRepo{}
	e := newRepositoryHandlerEcho(t, repos, &listMockOrgRepo{}, auditLog, authMiddleware(listTestUserID))

	req := httptest.NewRequest(http.MethodDelete, "/repos/"+listTestOwnerLogin+"/delete-me", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	if len(auditLog.records) != 1 {
		t.Fatalf("audit records = %d, want 1", len(auditLog.records))
	}
	if auditLog.records[0].action != "repo.delete" {
		t.Fatalf("action = %q, want repo.delete", auditLog.records[0].action)
	}
}
