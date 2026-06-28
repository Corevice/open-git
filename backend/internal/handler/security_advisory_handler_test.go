package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
	orgUC "github.com/open-git/backend/internal/usecase/org"
	securityusecase "github.com/open-git/backend/internal/usecase/security"
)

type advisoryMembershipRepo struct {
	roles map[int64]map[int64]string
}

func (m *advisoryMembershipRepo) Add(context.Context, *entity.Membership) error {
	return nil
}

func (m *advisoryMembershipRepo) GetRole(_ context.Context, orgID, userID uuid.UUID) (string, error) {
	if m.roles == nil {
		return "", nil
	}
	orgKey := middleware.UUIDToInt64(orgID)
	userKey := middleware.UUIDToInt64(userID)
	if userRoles, ok := m.roles[orgKey]; ok {
		return userRoles[userKey], nil
	}
	return "", nil
}

func (m *advisoryMembershipRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Membership, error) {
	return nil, nil
}

func (m *advisoryMembershipRepo) UpdateRole(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}

func (m *advisoryMembershipRepo) Remove(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

type mockListAdvisoriesUC struct {
	output *securityusecase.ListAdvisoriesOutput
	err    error
}

func (m *mockListAdvisoriesUC) Execute(context.Context, securityusecase.ListAdvisoriesInput) (*securityusecase.ListAdvisoriesOutput, error) {
	return m.output, m.err
}

type mockUpdateAdvisoryStateUC struct {
	result *entity.SecurityAdvisory
	err    error
}

func (m *mockUpdateAdvisoryStateUC) Execute(context.Context, securityusecase.UpdateAdvisoryStateInput) (*entity.SecurityAdvisory, error) {
	return m.result, m.err
}

var testAdvisoryRepo = &entity.Repository{
	ID:             uuid.MustParse("00000000-0000-0000-0000-000000000099"),
	OrganizationID: listTestOrgUUID,
	Name:           "demo",
}

func newSecurityAdvisoryHandlerEcho(
	t *testing.T,
	orgs *mockOrgRepo,
	memberships *advisoryMembershipRepo,
	listUC handler.ListAdvisoriesExecutor,
	updateUC handler.UpdateAdvisoryStateExecutor,
	auth echo.MiddlewareFunc,
) *echo.Echo {
	t.Helper()

	h := handler.NewSecurityAdvisoryHandler(
		orgUC.NewGetOrgUsecase(orgs),
		memberships,
		listUC,
		nil,
		updateUC,
		func(_ echo.Context, _, _ string) (*entity.Repository, error) {
			return testAdvisoryRepo, nil
		},
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, auth)
	return e
}

func TestPatchAdvisory_Forbidden(t *testing.T) {
	memberships := &advisoryMembershipRepo{
		roles: map[int64]map[int64]string{
			listTestOrgID: {listTestUserID: entity.RoleMember},
		},
	}
	e := newSecurityAdvisoryHandlerEcho(
		t,
		&mockOrgRepo{},
		memberships,
		nil,
		&mockUpdateAdvisoryStateUC{},
		authMiddleware(listTestUserID),
	)

	body := bytes.NewBufferString(`{"state":"acknowledged"}`)
	req := httptest.NewRequest(http.MethodPatch, "/repos/acme/demo/security-advisories/GHSA-test-0001", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestPatchAdvisory_InvalidTransition(t *testing.T) {
	memberships := &advisoryMembershipRepo{
		roles: map[int64]map[int64]string{
			listTestOrgID: {listTestUserID: entity.RoleAdmin},
		},
	}
	e := newSecurityAdvisoryHandlerEcho(
		t,
		&mockOrgRepo{},
		memberships,
		nil,
		&mockUpdateAdvisoryStateUC{
			err: fmt.Errorf("wrap: %w", securityusecase.ErrInvalidTransition),
		},
		authMiddleware(listTestUserID),
	)

	body := bytes.NewBufferString(`{"state":"open"}`)
	req := httptest.NewRequest(http.MethodPatch, "/repos/acme/demo/security-advisories/GHSA-test-0001", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestPatchAdvisory_MissingReason(t *testing.T) {
	memberships := &advisoryMembershipRepo{
		roles: map[int64]map[int64]string{
			listTestOrgID: {listTestUserID: entity.RoleAdmin},
		},
	}
	e := newSecurityAdvisoryHandlerEcho(
		t,
		&mockOrgRepo{},
		memberships,
		nil,
		&mockUpdateAdvisoryStateUC{err: apperror.ErrValidation},
		authMiddleware(listTestUserID),
	)

	body := bytes.NewBufferString(`{"state":"dismissed"}`)
	req := httptest.NewRequest(http.MethodPatch, "/repos/acme/demo/security-advisories/GHSA-test-0001", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestListOrgAdvisories_OK(t *testing.T) {
	orgs := &mockOrgRepo{
		byLogin: map[string]*domain.Organization{
			"acme": {ID: listTestOrgID, Login: "acme"},
		},
	}
	listUC := &mockListAdvisoriesUC{
		output: &securityusecase.ListAdvisoriesOutput{
			Advisories: []*entity.SecurityAdvisory{
				{GHSAPID: "GHSA-1", State: entity.AdvisoryStateOpen},
				{GHSAPID: "GHSA-2", State: entity.AdvisoryStateOpen},
			},
			Total: 2,
		},
	}
	e := newSecurityAdvisoryHandlerEcho(
		t,
		orgs,
		&advisoryMembershipRepo{},
		listUC,
		nil,
		authMiddleware(listTestUserID),
	)

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/security-advisories", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("len(advisories) = %d, want 2", len(resp))
	}
}

var _ domainrepo.IMembershipRepository = (*advisoryMembershipRepo)(nil)
