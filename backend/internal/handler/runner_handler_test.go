package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	actionsusecase "github.com/open-git/backend/internal/usecase/actions"
)

var runnerTestOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000050")

type mockCreateRegistrationTokenUC struct {
	called bool
	token  *entity.RunnerRegistrationToken
	raw    string
	err    error
}

func (m *mockCreateRegistrationTokenUC) Execute(_ context.Context, _ uuid.UUID, _ string) (*entity.RunnerRegistrationToken, string, error) {
	m.called = true
	if m.err != nil {
		return nil, "", m.err
	}
	if m.token == nil {
		return &entity.RunnerRegistrationToken{
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		}, "raw-registration-token", nil
	}
	raw := m.raw
	if raw == "" {
		raw = "raw-registration-token"
	}
	return m.token, raw, nil
}

type mockRegisterRunnerUC struct {
	called bool
	runner *entity.Runner
	err    error
}

func (m *mockRegisterRunnerUC) Execute(_ context.Context, _ uuid.UUID, _ actionsusecase.RegisterRunnerRequest) (*entity.Runner, error) {
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	if m.runner != nil {
		return m.runner, nil
	}
	return &entity.Runner{
		ID:     uuid.New(),
		Name:   "runner-1",
		Status: entity.RunnerStatusOnline,
		Labels: []string{"self-hosted"},
	}, nil
}

type mockListRunnersUC struct {
	called  bool
	runners []*entity.Runner
	err     error
}

func (m *mockListRunnersUC) Execute(_ context.Context, _ uuid.UUID) ([]*entity.Runner, error) {
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	return m.runners, nil
}

type mockDeleteRunnerUC struct {
	called bool
	err    error
}

func (m *mockDeleteRunnerUC) Execute(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) error {
	m.called = true
	return m.err
}

type mockHeartbeatRunnerUC struct {
	called bool
	status string
	err    error
}

func (m *mockHeartbeatRunnerUC) Execute(_ context.Context, _ uuid.UUID, _ uuid.UUID, status string, _ *string) error {
	m.called = true
	m.status = status
	return m.err
}

func newRunnerHandlerEcho(
	t *testing.T,
	createToken *mockCreateRegistrationTokenUC,
	registerRunner *mockRegisterRunnerUC,
	listRunners *mockListRunnersUC,
	deleteRunner *mockDeleteRunnerUC,
	heartbeatRunner *mockHeartbeatRunnerUC,
	actorRole string,
) *echo.Echo {
	t.Helper()

	if createToken == nil {
		createToken = &mockCreateRegistrationTokenUC{}
	}
	if registerRunner == nil {
		registerRunner = &mockRegisterRunnerUC{}
	}
	if listRunners == nil {
		listRunners = &mockListRunnersUC{}
	}
	if deleteRunner == nil {
		deleteRunner = &mockDeleteRunnerUC{}
	}
	if heartbeatRunner == nil {
		heartbeatRunner = &mockHeartbeatRunnerUC{}
	}
	if actorRole == "" {
		actorRole = entity.RoleAdmin
	}

	h := handler.NewRunnerHandlerWithDeps(
		createToken,
		registerRunner,
		listRunners,
		deleteRunner,
		heartbeatRunner,
		func(_ echo.Context) (uuid.UUID, error) {
			return runnerTestOrgID, nil
		},
		func(_ echo.Context, _ uuid.UUID) (string, error) {
			return actorRole, nil
		},
	)

	e := echo.New()
	g := e.Group("/orgs/:org/actions")
	h.RegisterRoutes(g)
	return e
}

func TestCreateRegistrationToken_HappyPath(t *testing.T) {
	createToken := &mockCreateRegistrationTokenUC{
		token: &entity.RunnerRegistrationToken{
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		},
	}
	e := newRunnerHandlerEcho(t, createToken, nil, nil, nil, nil, entity.RoleAdmin)

	req := httptest.NewRequest(http.MethodPost, "/orgs/test/actions/runners/registration-token", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !createToken.called {
		t.Fatal("expected create token use case to be called")
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["token"] == "" || resp["expires_at"] == "" {
		t.Fatalf("expected token and expires_at, got %#v", resp)
	}
}

func TestRegisterRunner_HappyPath(t *testing.T) {
	runnerID := uuid.New()
	registerRunner := &mockRegisterRunnerUC{
		runner: &entity.Runner{
			ID:     runnerID,
			Name:   "ci-runner",
			Status: entity.RunnerStatusOnline,
			Labels: []string{"linux", "x64"},
		},
	}
	e := newRunnerHandlerEcho(t, nil, registerRunner, nil, nil, nil, entity.RoleAdmin)

	body := bytes.NewBufferString(`{
		"registration_token":"tok",
		"name":"ci-runner",
		"labels":["linux","x64"],
		"os":"linux",
		"arch":"x64",
		"runner_type":"act"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/orgs/test/actions/runners", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !registerRunner.called {
		t.Fatal("expected register runner use case to be called")
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["id"] != runnerID.String() || resp["status"] != entity.RunnerStatusOnline {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestRegisterRunner_ExpiredToken_Returns401(t *testing.T) {
	registerRunner := &mockRegisterRunnerUC{err: domain.ErrUnauthorized}
	e := newRunnerHandlerEcho(t, nil, registerRunner, nil, nil, nil, entity.RoleAdmin)

	body := bytes.NewBufferString(`{
		"registration_token":"expired",
		"name":"ci-runner",
		"labels":["linux"],
		"os":"linux",
		"arch":"x64",
		"runner_type":"act"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/orgs/test/actions/runners", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRegisterRunner_InvalidLabel_Returns400(t *testing.T) {
	registerRunner := &mockRegisterRunnerUC{
		err: fmt.Errorf("%w: invalid label %q", apperror.ErrValidation, "bad label"),
	}
	e := newRunnerHandlerEcho(t, nil, registerRunner, nil, nil, nil, entity.RoleAdmin)

	body := bytes.NewBufferString(`{
		"registration_token":"tok",
		"name":"ci-runner",
		"labels":["bad label"],
		"os":"linux",
		"arch":"x64",
		"runner_type":"act"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/orgs/test/actions/runners", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListRunners_HappyPath(t *testing.T) {
	runnerID := uuid.New()
	listRunners := &mockListRunnersUC{
		runners: []*entity.Runner{
			{
				ID:         runnerID,
				Name:       "runner-1",
				Status:     entity.RunnerStatusOnline,
				Labels:     []string{"self-hosted"},
				RunnerType: entity.RunnerTypeAct,
			},
		},
	}
	e := newRunnerHandlerEcho(t, nil, nil, listRunners, nil, nil, entity.RoleAdmin)

	req := httptest.NewRequest(http.MethodGet, "/orgs/test/actions/runners", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !listRunners.called {
		t.Fatal("expected list runners use case to be called")
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	runners, ok := resp["runners"].([]any)
	if !ok || len(runners) != 1 {
		t.Fatalf("expected 1 runner, got %#v", resp["runners"])
	}
}

func TestDeleteRunner_HappyPath(t *testing.T) {
	deleteRunner := &mockDeleteRunnerUC{}
	e := newRunnerHandlerEcho(t, nil, nil, nil, deleteRunner, nil, entity.RoleAdmin)

	runnerID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/orgs/test/actions/runners/"+runnerID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !deleteRunner.called {
		t.Fatal("expected delete runner use case to be called")
	}
}

func TestDeleteRunner_NonAdmin_Returns403(t *testing.T) {
	deleteRunner := &mockDeleteRunnerUC{err: domain.ErrForbidden}
	e := newRunnerHandlerEcho(t, nil, nil, nil, deleteRunner, nil, entity.RoleMember)

	runnerID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/orgs/test/actions/runners/"+runnerID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHeartbeatRunner_UpdatesStatus(t *testing.T) {
	heartbeatRunner := &mockHeartbeatRunnerUC{}
	e := newRunnerHandlerEcho(t, nil, nil, nil, nil, heartbeatRunner, entity.RoleAdmin)

	runnerID := uuid.New()
	body := bytes.NewBufferString(`{"status":"busy"}`)
	req := httptest.NewRequest(http.MethodPost, "/orgs/test/actions/runners/"+runnerID.String()+"/heartbeat", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !heartbeatRunner.called {
		t.Fatal("expected heartbeat use case to be called")
	}
	if heartbeatRunner.status != "busy" {
		t.Fatalf("expected status busy, got %q", heartbeatRunner.status)
	}
}
