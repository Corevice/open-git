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
	labelusecase "github.com/open-git/backend/internal/usecase/label"
)

type labelTestEnv struct {
	echo   *echo.Echo
	repo   *entity.Repository
	userID uuid.UUID
	orgID  uuid.UUID
}

func openLabelTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.RunMigrations(db, "sqlite", "../../../migrations"); err != nil {
		_ = db.Close()
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func createLabelTestUser(t *testing.T, db *sqlx.DB, login string) uuid.UUID {
	t.Helper()

	userRepo := infrarepo.NewUserRepository(db)
	user := &entity.User{
		Login:        login,
		Email:        login + "@example.com",
		PasswordHash: "hashed",
	}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user %s: %v", login, err)
	}
	return user.ID
}

func createLabelTestOrganization(t *testing.T, db *sqlx.DB, login string) uuid.UUID {
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

func createLabelTestRepository(t *testing.T, db *sqlx.DB, orgID, ownerID uuid.UUID, name string) *entity.Repository {
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

func newLabelTestEnv(t *testing.T) labelTestEnv {
	t.Helper()

	db := openLabelTestDB(t)
	orgID := createLabelTestOrganization(t, db, "label-org")
	userID := createLabelTestUser(t, db, "alice")
	testRepo := createLabelTestRepository(t, db, orgID, userID, "demo")

	labelRepo := infrarepo.NewLabelRepository(db)
	auditRepo := infrarepo.NewAuditLogRepository(db)

	h := handler.NewLabelHandler(
		labelusecase.NewListLabelsUsecase(labelRepo),
		labelusecase.NewCreateLabelUsecase(labelRepo),
		labelusecase.NewUpdateLabelUsecase(labelRepo),
		labelusecase.NewDeleteLabelUsecase(labelRepo, auditRepo),
		func(_ echo.Context, _, _ string) (*entity.Repository, error) {
			return testRepo, nil
		},
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, labelTestAuth(orgID, userID))

	return labelTestEnv{
		echo:   e,
		repo:   testRepo,
		userID: userID,
		orgID:  orgID,
	}
}

func labelTestAuth(_ uuid.UUID, userID uuid.UUID) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, middleware.UUIDToInt64(userID), []string{"repo"})
			return next(c)
		}
	}
}

func TestListLabelsEmpty(t *testing.T) {
	env := newLabelTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/labels", nil)
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

func TestCreateLabelSuccess(t *testing.T) {
	env := newLabelTestEnv(t)

	body := `{"name":"bug","color":"ff0000","description":"Bug reports"}`
	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/labels", bytes.NewBufferString(body))
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
	if resp["name"] != "bug" {
		t.Fatalf("name = %v, want bug", resp["name"])
	}
	if resp["color"] != "ff0000" {
		t.Fatalf("color = %v, want ff0000", resp["color"])
	}
	if resp["description"] != "Bug reports" {
		t.Fatalf("description = %v, want Bug reports", resp["description"])
	}
}

func TestCreateLabelDuplicateName(t *testing.T) {
	env := newLabelTestEnv(t)

	body := `{"name":"bug","color":"ff0000","description":"Bug reports"}`
	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/labels", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup create status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	dupReq := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/labels", bytes.NewBufferString(body))
	dupReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	dupRec := httptest.NewRecorder()
	env.echo.ServeHTTP(dupRec, dupReq)

	if dupRec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", dupRec.Code, http.StatusUnprocessableEntity, dupRec.Body.String())
	}
}

func TestCreateLabelInvalidColor(t *testing.T) {
	env := newLabelTestEnv(t)

	body := `{"name":"bug","color":"gg0000","description":"Bug reports"}`
	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/labels", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestUpdateLabel(t *testing.T) {
	env := newLabelTestEnv(t)

	createBody := `{"name":"bug","color":"ff0000","description":"Bug reports"}`
	createReq := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/labels", bytes.NewBufferString(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRec := httptest.NewRecorder()
	env.echo.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("setup create status = %d, want %d, body = %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}

	updateBody := `{"color":"00ff00","description":"Updated description"}`
	updateReq := httptest.NewRequest(http.MethodPatch, "/repos/alice/demo/labels/bug", bytes.NewBufferString(updateBody))
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
	if resp["name"] != "bug" {
		t.Fatalf("name = %v, want bug", resp["name"])
	}
	if resp["color"] != "00ff00" {
		t.Fatalf("color = %v, want 00ff00", resp["color"])
	}
	if resp["description"] != "Updated description" {
		t.Fatalf("description = %v, want Updated description", resp["description"])
	}
}

func TestDeleteLabel(t *testing.T) {
	env := newLabelTestEnv(t)

	createBody := `{"name":"bug","color":"ff0000","description":"Bug reports"}`
	createReq := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/labels", bytes.NewBufferString(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRec := httptest.NewRecorder()
	env.echo.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("setup create status = %d, want %d, body = %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/repos/alice/demo/labels/bug", nil)
	deleteRec := httptest.NewRecorder()
	env.echo.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", deleteRec.Code, http.StatusNoContent, deleteRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/labels", nil)
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
