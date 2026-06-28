package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
	importUC "github.com/open-git/backend/internal/usecase/import"
)

const (
	importTestOrgLogin = "myorg"
	importTestOrgID    = int64(1)
	importTestUserID   = int64(7)
)

var (
	importOrgUUID  = middleware.Int64ToUUID(importTestOrgID)
	importUserUUID = middleware.Int64ToUUID(importTestUserID)
)

type importMockOrgRepo struct {
	byLogin map[string]*domain.Organization
}

func (m *importMockOrgRepo) GetByLogin(_ context.Context, login string) (*domain.Organization, error) {
	if m.byLogin == nil {
		return nil, nil
	}
	return m.byLogin[login], nil
}

func (m *importMockOrgRepo) ListByUserID(context.Context, int64) ([]*domain.Organization, error) {
	return nil, nil
}

func (m *importMockOrgRepo) GetMemberRole(context.Context, int64, int64) (string, error) {
	return "", nil
}

type importMockMembershipRepo struct {
	readAccess map[string]bool
}

func importMembershipKey(orgID, userID uuid.UUID) string {
	return fmt.Sprintf("%s:%s", orgID, userID)
}

func (m *importMockMembershipRepo) HasReadAccess(_ context.Context, userID, orgID uuid.UUID) (bool, error) {
	if m.readAccess == nil {
		return false, nil
	}
	return m.readAccess[importMembershipKey(orgID, userID)], nil
}

func (m *importMockMembershipRepo) HasWriteAccess(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

type importMockJobRepo struct {
	jobs map[uuid.UUID]*entity.ImportJob
}

func (m *importMockJobRepo) Create(_ context.Context, job *entity.ImportJob) error {
	if m.jobs == nil {
		m.jobs = map[uuid.UUID]*entity.ImportJob{}
	}
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	m.jobs[job.ID] = job
	return nil
}

func (m *importMockJobRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.ImportJob, error) {
	return m.jobs[id], nil
}

func (m *importMockJobRepo) GetByIDAndOrg(_ context.Context, id, orgID uuid.UUID) (*entity.ImportJob, error) {
	job, ok := m.jobs[id]
	if !ok || job.OrganizationID != orgID {
		return nil, fmt.Errorf("import job not found: %w", sql.ErrNoRows)
	}
	return job, nil
}

func (m *importMockJobRepo) ListByOrg(_ context.Context, orgID uuid.UUID, _, _ int) ([]*entity.ImportJob, int, error) {
	var jobs []*entity.ImportJob
	for _, job := range m.jobs {
		if job.OrganizationID == orgID {
			jobs = append(jobs, job)
		}
	}
	return jobs, len(jobs), nil
}

func (m *importMockJobRepo) UpdateStatus(_ context.Context, id uuid.UUID, status entity.ImportJobStatus) error {
	job, ok := m.jobs[id]
	if !ok {
		return errors.New("not found")
	}
	job.Status = status
	return nil
}

func (m *importMockJobRepo) UpdatePhase(_ context.Context, id uuid.UUID, phase entity.ImportJobPhase) error {
	job, ok := m.jobs[id]
	if !ok {
		return errors.New("not found")
	}
	job.Phase = phase
	return nil
}

func (m *importMockJobRepo) UpdateProgress(context.Context, uuid.UUID, entity.ImportProgress) error {
	return nil
}

func (m *importMockJobRepo) SetError(context.Context, uuid.UUID, string) error {
	return nil
}

func (m *importMockJobRepo) SetTargetRepository(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (m *importMockJobRepo) SaveCheckpoint(context.Context, *entity.ImportPhaseCheckpoint) error {
	return nil
}

func (m *importMockJobRepo) GetCheckpoint(context.Context, uuid.UUID, entity.ImportJobPhase) (*entity.ImportPhaseCheckpoint, error) {
	return nil, sql.ErrNoRows
}

func (m *importMockJobRepo) MarkPhaseComplete(context.Context, uuid.UUID, entity.ImportJobPhase) error {
	return nil
}

type importDomainMembershipRepo struct {
	roles map[string]string
}

func (m *importDomainMembershipRepo) Add(context.Context, *entity.Membership) error { return nil }

func (m *importDomainMembershipRepo) GetRole(_ context.Context, orgID, userID uuid.UUID) (string, error) {
	if m.roles == nil {
		return "", domain.ErrNotFound
	}
	role, ok := m.roles[importMembershipKey(orgID, userID)]
	if !ok {
		return "", domain.ErrNotFound
	}
	return role, nil
}

func (m *importDomainMembershipRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Membership, error) {
	return nil, nil
}

func (m *importDomainMembershipRepo) UpdateRole(context.Context, uuid.UUID, uuid.UUID, string) error { return nil }

func (m *importDomainMembershipRepo) Remove(context.Context, uuid.UUID, uuid.UUID) error { return nil }

type importMockRepositoryRepo struct {
	byLoginAndName map[string]*entity.Repository
}

func importRepoLoginKey(ownerLogin, name string) string {
	return fmt.Sprintf("%s:%s", ownerLogin, name)
}

func (m *importMockRepositoryRepo) Create(context.Context, *entity.Repository) error { return nil }

func (m *importMockRepositoryRepo) GetByOwnerAndName(context.Context, uuid.UUID, string) (*entity.Repository, error) {
	return nil, errors.New("not found")
}

func (m *importMockRepositoryRepo) GetByOwnerLoginAndName(_ context.Context, ownerLogin, name string) (*entity.Repository, error) {
	if m.byLoginAndName == nil {
		return nil, errors.New("not found")
	}
	if repo, ok := m.byLoginAndName[importRepoLoginKey(ownerLogin, name)]; ok {
		return repo, nil
	}
	return nil, errors.New("not found")
}

func (m *importMockRepositoryRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *importMockRepositoryRepo) CountByOrg(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (m *importMockRepositoryRepo) ListByOwner(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *importMockRepositoryRepo) CountByOwner(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (m *importMockRepositoryRepo) UpdateVisibility(context.Context, uuid.UUID, string) error { return nil }

func (m *importMockRepositoryRepo) UpdateName(context.Context, uuid.UUID, string) error { return nil }

func (m *importMockRepositoryRepo) Delete(context.Context, uuid.UUID) error { return nil }

type importMockEntityOrgRepo struct {
	org *entity.Organization
}

func (m *importMockEntityOrgRepo) Create(context.Context, *entity.Organization) error { return nil }

func (m *importMockEntityOrgRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.Organization, error) {
	if m.org != nil && m.org.ID == id {
		return m.org, nil
	}
	return nil, nil
}

func (m *importMockEntityOrgRepo) GetByLogin(context.Context, string) (*entity.Organization, error) {
	return nil, nil
}

func (m *importMockEntityOrgRepo) List(context.Context, int, int) ([]*entity.Organization, error) {
	return nil, nil
}

func (m *importMockEntityOrgRepo) Update(context.Context, *entity.Organization) error { return nil }

func (m *importMockEntityOrgRepo) Delete(context.Context, uuid.UUID) error { return nil }

type importMockEnqueuer struct{}

func (importMockEnqueuer) EnqueueGitHubImport(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

type importHandlerFixture struct {
	jobs *importMockJobRepo
}

func newImportHandlerEcho(t *testing.T, fixture *importHandlerFixture, repos *importMockRepositoryRepo, auth echo.MiddlewareFunc) *echo.Echo {
	t.Helper()

	if fixture == nil {
		fixture = &importHandlerFixture{jobs: &importMockJobRepo{jobs: map[uuid.UUID]*entity.ImportJob{}}}
	}
	if fixture.jobs == nil {
		fixture.jobs = &importMockJobRepo{jobs: map[uuid.UUID]*entity.ImportJob{}}
	}

	memberships := &importDomainMembershipRepo{
		roles: map[string]string{
			importMembershipKey(importOrgUUID, importUserUUID): entity.RoleAdmin,
		},
	}
	entityOrgs := &importMockEntityOrgRepo{
		org: &entity.Organization{ID: importOrgUUID, Login: importTestOrgLogin},
	}
	enqueuer := importMockEnqueuer{}

	createUC := importUC.NewCreateImportJobUsecaseWithDeps(
		fixture.jobs,
		memberships,
		repos,
		entityOrgs,
		nil,
		enqueuer,
	)

	h := handler.NewImportHandler(
		createUC,
		importUC.NewGetImportJobUsecase(fixture.jobs),
		importUC.NewListImportJobsUsecase(fixture.jobs),
		importUC.NewCancelImportJobUsecase(fixture.jobs, memberships),
		importUC.NewRetryImportJobUsecaseWithEnqueuer(fixture.jobs, fixture.jobs, memberships, enqueuer),
		&importMockOrgRepo{
			byLogin: map[string]*domain.Organization{
				importTestOrgLogin: {ID: importTestOrgID, Login: importTestOrgLogin},
			},
		},
		&importMockMembershipRepo{
			readAccess: map[string]bool{
				importMembershipKey(importOrgUUID, importUserUUID): true,
			},
		},
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, auth)
	return e
}

func importAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set("user_id", importTestUserID)
		return next(c)
	}
}

func TestCreateImportAccepted(t *testing.T) {
	repos := &importMockRepositoryRepo{byLoginAndName: map[string]*entity.Repository{}}
	e := newImportHandlerEcho(t, nil, repos, importAuthMiddleware)

	body := map[string]any{
		"source_url":  "https://github.com/acme/source-repo",
		"target_name": "source-repo",
		"include":     []string{"code"},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/orgs/myorg/imports", bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["job_id"] == "" {
		t.Fatal("expected job_id in response")
	}
	if resp["status"] != "queued" {
		t.Fatalf("status = %q, want queued", resp["status"])
	}
}

func TestCreateImportTargetNameConflict(t *testing.T) {
	repos := &importMockRepositoryRepo{
		byLoginAndName: map[string]*entity.Repository{
			importRepoLoginKey(importTestOrgLogin, "taken"): {Name: "taken"},
		},
	}
	e := newImportHandlerEcho(t, nil, repos, importAuthMiddleware)

	body := map[string]any{
		"source_url":  "https://github.com/acme/source-repo",
		"target_name": "taken",
		"include":     []string{"code"},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/orgs/myorg/imports", bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestGetImportJobOK(t *testing.T) {
	jobID := uuid.New()
	now := time.Now().UTC()
	fixture := &importHandlerFixture{
		jobs: &importMockJobRepo{
			jobs: map[uuid.UUID]*entity.ImportJob{
				jobID: {
					ID:             jobID,
					OrganizationID: importOrgUUID,
					Status:         entity.ImportJobStatusRunning,
					Phase:          entity.ImportJobPhaseIssues,
					Progress:       entity.ImportProgress{"issues": {Done: 1, Total: 3}},
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
		},
	}
	repos := &importMockRepositoryRepo{byLoginAndName: map[string]*entity.Repository{}}
	e := newImportHandlerEcho(t, fixture, repos, importAuthMiddleware)

	req := httptest.NewRequest(http.MethodGet, "/orgs/myorg/imports/"+jobID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["status"] != "running" {
		t.Fatalf("status = %v, want running", resp["status"])
	}
	if resp["phase"] != "issues" {
		t.Fatalf("phase = %v, want issues", resp["phase"])
	}
}

func TestGetImportJobWrongOrgNotFound(t *testing.T) {
	jobID := uuid.New()
	otherOrgID := uuid.New()
	fixture := &importHandlerFixture{
		jobs: &importMockJobRepo{
			jobs: map[uuid.UUID]*entity.ImportJob{
				jobID: {
					ID:             jobID,
					OrganizationID: otherOrgID,
					Status:         entity.ImportJobStatusQueued,
					Phase:          entity.ImportJobPhaseClone,
				},
			},
		},
	}
	repos := &importMockRepositoryRepo{byLoginAndName: map[string]*entity.Repository{}}
	e := newImportHandlerEcho(t, fixture, repos, importAuthMiddleware)

	req := httptest.NewRequest(http.MethodGet, "/orgs/myorg/imports/"+jobID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestCancelImportJobOK(t *testing.T) {
	jobID := uuid.New()
	fixture := &importHandlerFixture{
		jobs: &importMockJobRepo{
			jobs: map[uuid.UUID]*entity.ImportJob{
				jobID: {
					ID:             jobID,
					OrganizationID: importOrgUUID,
					Status:         entity.ImportJobStatusRunning,
					Phase:          entity.ImportJobPhaseClone,
				},
			},
		},
	}
	repos := &importMockRepositoryRepo{byLoginAndName: map[string]*entity.Repository{}}
	e := newImportHandlerEcho(t, fixture, repos, importAuthMiddleware)

	req := httptest.NewRequest(http.MethodPost, "/orgs/myorg/imports/"+jobID.String()+"/cancel", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["status"] != "cancelled" {
		t.Fatalf("status = %q, want cancelled", resp["status"])
	}
}

func TestImportUnauthenticated(t *testing.T) {
	repos := &importMockRepositoryRepo{byLoginAndName: map[string]*entity.Repository{}}
	e := newImportHandlerEcho(t, nil, repos, func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "Bad credentials"})
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/orgs/myorg/imports", bytes.NewReader([]byte(`{}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
