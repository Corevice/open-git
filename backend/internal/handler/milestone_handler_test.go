package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/infrastructure/database"
	infrarepo "github.com/open-git/backend/internal/infrastructure/repository"
	"github.com/open-git/backend/internal/middleware"
	milestoneusecase "github.com/open-git/backend/internal/usecase/milestone"
)

var milestoneTestUserID = uuid.MustParse("00000000-0000-0000-0000-000000000008")

type milestoneTestEnv struct {
	echo   *echo.Echo
	repo   *entity.Repository
	userID uuid.UUID
	orgID  uuid.UUID
}

func openMilestoneTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.RunMigrations(db, "sqlite", "../../migrations"); err != nil {
		_ = db.Close()
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func createMilestoneTestUser(t *testing.T, db *sqlx.DB, login string) uuid.UUID {
	t.Helper()

	userRepo := infrarepo.NewUserRepository(db)
	user := &entity.User{
		ID:           milestoneTestUserID,
		Login:        login,
		Email:        login + "@example.com",
		PasswordHash: "hashed",
	}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user %s: %v", login, err)
	}
	return user.ID
}

func createMilestoneTestOrganization(t *testing.T, db *sqlx.DB, login string) uuid.UUID {
	t.Helper()

	orgRepo := infrarepo.NewOrganizationRepository(db)
	org := &entity.Organization{
		Login: login,
		Name:  login,
	}
	if err := orgRepo.Create(context.Background(), org); err != nil {
		t.Fatalf("create org %s: %v", login, err)
	}
	return org.ID
}

func createMilestoneTestRepository(t *testing.T, db *sqlx.DB, orgID, ownerID uuid.UUID, name string) *entity.Repository {
	t.Helper()

	repoRepo := infrarepo.NewRepositoryRepository(db)
	repo := &entity.Repository{
		OrganizationID: orgID,
		OwnerID:        ownerID,
		Name:           name,
		OwnerLogin:     "alice",
		Visibility:     entity.VisibilityPrivate,
		DefaultBranch:  "main",
		GitPath:        "/tmp/" + name + ".git",
	}
	if err := repoRepo.Create(context.Background(), repo); err != nil {
		t.Fatalf("create repository: %v", err)
	}
	return repo
}

func newMilestoneTestEnv(t *testing.T) milestoneTestEnv {
	t.Helper()

	db := openMilestoneTestDB(t)
	orgID := createMilestoneTestOrganization(t, db, "milestone-org")
	userID := createMilestoneTestUser(t, db, "alice")
	testRepo := createMilestoneTestRepository(t, db, orgID, userID, "demo")

	milestoneRepo := infrarepo.NewMilestoneRepository(db)
	auditRepo := infrarepo.NewAuditLogRepository(db)

	h := handler.NewMilestoneHandler(
		milestoneusecase.NewListMilestonesUsecase(milestoneRepo),
		milestoneusecase.NewCreateMilestoneUsecase(milestoneRepo, auditRepo),
		milestoneusecase.NewUpdateMilestoneUsecase(milestoneRepo),
		milestoneusecase.NewDeleteMilestoneUsecase(milestoneRepo, auditRepo),
		func(_ echo.Context, _, _ string) (*entity.Repository, error) {
			return testRepo, nil
		},
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, milestoneTestAuth(orgID, userID))

	return milestoneTestEnv{
		echo:   e,
		repo:   testRepo,
		userID: userID,
		orgID:  orgID,
	}
}

func milestoneTestAuth(_ uuid.UUID, userID uuid.UUID) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, middleware.UUIDToInt64(userID), []string{"repo"})
			return next(c)
		}
	}
}

func TestListMilestonesEmpty(t *testing.T) {
	env := newMilestoneTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/milestones", nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp []any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("expected empty array, got %d items", len(resp))
	}
}

func TestCreateMilestoneSuccess(t *testing.T) {
	env := newMilestoneTestEnv(t)

	body := `{"title":"v1.0","description":"First release","due_on":"2026-12-31T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/milestones", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["title"] != "v1.0" {
		t.Fatalf("title = %v, want v1.0", resp["title"])
	}
	if resp["description"] != "First release" {
		t.Fatalf("description = %v, want First release", resp["description"])
	}
	if resp["state"] != "open" {
		t.Fatalf("state = %v, want open", resp["state"])
	}
	if resp["number"].(float64) != 1 {
		t.Fatalf("number = %v, want 1", resp["number"])
	}
}

func TestCreateMilestoneEmptyTitle(t *testing.T) {
	env := newMilestoneTestEnv(t)

	body := `{"title":"","description":"No title"}`
	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/milestones", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestUpdateMilestone(t *testing.T) {
	env := newMilestoneTestEnv(t)

	createBody := `{"title":"v1.0","description":"First release"}`
	createReq := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/milestones", bytes.NewBufferString(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRec := httptest.NewRecorder()
	env.echo.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("setup create status = %d, want %d, body = %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}

	updateBody := `{"title":"v1.1","description":"Updated release"}`
	updateReq := httptest.NewRequest(http.MethodPatch, "/repos/alice/demo/milestones/1", bytes.NewBufferString(updateBody))
	updateReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	updateRec := httptest.NewRecorder()
	env.echo.ServeHTTP(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", updateRec.Code, http.StatusOK, updateRec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(updateRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["title"] != "v1.1" {
		t.Fatalf("title = %v, want v1.1", resp["title"])
	}
	if resp["description"] != "Updated release" {
		t.Fatalf("description = %v, want Updated release", resp["description"])
	}
}

func TestDeleteMilestone(t *testing.T) {
	env := newMilestoneTestEnv(t)

	createBody := `{"title":"v1.0","description":"First release"}`
	createReq := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/milestones", bytes.NewBufferString(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRec := httptest.NewRecorder()
	env.echo.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("setup create status = %d, want %d, body = %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/repos/alice/demo/milestones/1", nil)
	deleteRec := httptest.NewRecorder()
	env.echo.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", deleteRec.Code, http.StatusNoContent, deleteRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/milestones?state=all", nil)
	listRec := httptest.NewRecorder()
	env.echo.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body = %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var resp []any
	if err := json.Unmarshal(listRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("expected empty array after delete, got %d items", len(resp))
	}
}
