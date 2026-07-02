package handler_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
	secretusecase "github.com/open-git/backend/internal/usecase/secret"
)

var (
	secretHandlerOrgID  = uuid.MustParse("00000000-0000-0000-0000-000000000040")
	secretHandlerRepoID = uuid.MustParse("00000000-0000-0000-0000-000000000041")
	secretHandlerUserID = int64(41)
)

type mockListRepoSecretsUC struct {
	secrets []*entity.ActionSecret
	err     error
}

func (m *mockListRepoSecretsUC) Execute(_ context.Context, _, _ uuid.UUID) ([]*entity.ActionSecret, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.secrets, nil
}

type mockUpsertSecretUC struct {
	created bool
	err     error
	last    secretusecase.UpsertActionSecretInput
}

func (m *mockUpsertSecretUC) Execute(_ context.Context, _ uuid.UUID, _ *uuid.UUID, input secretusecase.UpsertActionSecretInput) (bool, error) {
	m.last = input
	if m.err != nil {
		return false, m.err
	}
	return m.created, nil
}

type mockDeleteSecretUC struct {
	err error
}

func (m *mockDeleteSecretUC) Execute(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ uuid.UUID, _ string) error {
	return m.err
}

type mockGetPublicKeyUC struct {
	keyID string
	key   string
}

func (m *mockGetPublicKeyUC) Execute() (string, string) {
	return m.keyID, m.key
}

type mockSecretCryptor struct {
	plaintext []byte
	err       error
}

func (m *mockSecretCryptor) DecryptSealedBox(_ []byte) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.plaintext, nil
}

type noopSecretRepo struct{}

func (noopSecretRepo) Upsert(context.Context, *entity.ActionSecret) (bool, error) {
	return false, nil
}
func (noopSecretRepo) GetByName(context.Context, uuid.UUID, *uuid.UUID, string) (*entity.ActionSecret, error) {
	return nil, apperror.ErrNotFound
}
func (noopSecretRepo) ListByRepo(context.Context, uuid.UUID, uuid.UUID) ([]*entity.ActionSecret, error) {
	return nil, nil
}
func (noopSecretRepo) ListByOrg(context.Context, uuid.UUID) ([]*entity.ActionSecret, error) {
	return nil, nil
}
func (noopSecretRepo) Delete(context.Context, uuid.UUID, *uuid.UUID, string) error {
	return nil
}
func (noopSecretRepo) ListForWorkflow(context.Context, uuid.UUID, uuid.UUID) ([]*entity.ActionSecret, error) {
	return nil, nil
}
func (noopSecretRepo) SetSelectedRepositories(context.Context, uuid.UUID, uuid.UUID, []uuid.UUID) error {
	return nil
}
func (noopSecretRepo) GetSelectedRepositories(context.Context, uuid.UUID, uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func secretHandlerAuth(scopes ...string) echo.MiddlewareFunc {
	if len(scopes) == 0 {
		scopes = []string{"read", "write", "admin"}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, secretHandlerUserID, scopes)
			return next(c)
		}
	}
}

func secretHandlerTestRepo() *entity.Repository {
	return &entity.Repository{
		ID:             secretHandlerRepoID,
		OrganizationID: secretHandlerOrgID,
		OwnerLogin:     "alice",
		Name:           "myrepo",
	}
}

// secretTestAccess, when set, is wired into the handler so a test can exercise
// authorization; nil leaves it permissive for happy-path tests.
var secretTestAccess *handler.RepoAccess

type denySecretMembership struct{}

func (denySecretMembership) GetRole(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", domain.ErrNotFound
}

type denySecretCollab struct{}

func (denySecretCollab) GetPermission(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", nil
}

func newSecretHandlerEcho(
	t *testing.T,
	listUC *mockListRepoSecretsUC,
	upsertUC *mockUpsertSecretUC,
	deleteUC *mockDeleteSecretUC,
	publicKeyUC *mockGetPublicKeyUC,
	cryptor *mockSecretCryptor,
	auth echo.MiddlewareFunc,
	orgMiddleware echo.MiddlewareFunc,
) *echo.Echo {
	t.Helper()

	if listUC == nil {
		listUC = &mockListRepoSecretsUC{}
	}
	if upsertUC == nil {
		upsertUC = &mockUpsertSecretUC{created: true}
	}
	if deleteUC == nil {
		deleteUC = &mockDeleteSecretUC{}
	}
	if publicKeyUC == nil {
		publicKeyUC = &mockGetPublicKeyUC{keyID: "test-key-id", key: "dGVzdC1wdWJsaWNLZXk="}
	}
	if cryptor == nil {
		cryptor = &mockSecretCryptor{plaintext: []byte("plain-secret")}
	}
	if auth == nil {
		auth = secretHandlerAuth()
	}

	h := handler.NewSecretHandler(
		listUC,
		nil,
		nil,
		upsertUC,
		deleteUC,
		publicKeyUC,
		noopSecretRepo{},
		nil,
		cryptor,
		func(_ echo.Context, _, _ string) (*entity.Repository, error) {
			return secretHandlerTestRepo(), nil
		},
		func(_ echo.Context, _ string) (uuid.UUID, error) {
			return secretHandlerOrgID, nil
		},
	)
	h.SetAccess(secretTestAccess)

	e := echo.New()
	g := e.Group("")
	if orgMiddleware != nil {
		g.Use(orgMiddleware)
	}
	h.RegisterRoutes(g, auth)
	return e
}

func TestSecretHandler_ListRepoSecrets_ReturnsTotalCountAndSecrets(t *testing.T) {
	now := time.Now().UTC()
	listUC := &mockListRepoSecretsUC{
		secrets: []*entity.ActionSecret{
			{
				Name:           "MY_SECRET",
				EncryptedValue: "must-not-leak",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
		},
	}

	e := newSecretHandlerEcho(t, listUC, nil, nil, nil, nil, secretHandlerAuth("write"), nil)
	req := httptest.NewRequest(http.MethodGet, "/repos/alice/myrepo/actions/secrets", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["total_count"]; !ok {
		t.Fatalf("expected total_count in response, got %#v", resp)
	}
	secrets, ok := resp["secrets"].([]any)
	if !ok || len(secrets) != 1 {
		t.Fatalf("expected secrets array with 1 item, got %#v", resp["secrets"])
	}
	if strings.Contains(rec.Body.String(), "encrypted_value") {
		t.Fatalf("response must not contain encrypted_value, got %s", rec.Body.String())
	}
}

func TestSecretHandler_UpsertRepoSecret_ValidBody_Returns201(t *testing.T) {
	upsertUC := &mockUpsertSecretUC{created: true}
	cryptor := &mockSecretCryptor{plaintext: []byte("plain-secret")}
	e := newSecretHandlerEcho(t, nil, upsertUC, nil, nil, cryptor, secretHandlerAuth("admin"), nil)

	encoded := base64.StdEncoding.EncodeToString([]byte("sealed-box"))
	body := bytes.NewBufferString(`{"encrypted_value":"` + encoded + `","key_id":"test-key-id"}`)
	req := httptest.NewRequest(http.MethodPut, "/repos/alice/myrepo/actions/secrets/MY_SECRET", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if upsertUC.last.PlaintextValue != "plain-secret" {
		t.Fatalf("expected decrypted plaintext passed to use case, got %q", upsertUC.last.PlaintextValue)
	}
}

func TestSecretHandler_UpsertRepoSecret_ValidationFailure_Returns422(t *testing.T) {
	upsertUC := &mockUpsertSecretUC{
		err: apperror.ErrValidation,
	}
	e := newSecretHandlerEcho(t, nil, upsertUC, nil, nil, nil, secretHandlerAuth("admin"), nil)

	encoded := base64.StdEncoding.EncodeToString([]byte("sealed-box"))
	body := bytes.NewBufferString(`{"encrypted_value":"` + encoded + `","key_id":"test-key-id"}`)
	req := httptest.NewRequest(http.MethodPut, "/repos/alice/myrepo/actions/secrets/MY_SECRET", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSecretHandler_DeleteRepoSecret_NotFound_Returns404(t *testing.T) {
	deleteUC := &mockDeleteSecretUC{err: apperror.ErrNotFound}
	e := newSecretHandlerEcho(t, nil, nil, deleteUC, nil, nil, secretHandlerAuth("admin"), nil)

	req := httptest.NewRequest(http.MethodDelete, "/repos/alice/myrepo/actions/secrets/MY_SECRET", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSecretHandler_GetRepoPublicKey_ReturnsKeyIDAndKey(t *testing.T) {
	publicKeyUC := &mockGetPublicKeyUC{keyID: "repo-key-id", key: "cHVibGljS2V5"}
	e := newSecretHandlerEcho(t, nil, nil, nil, publicKeyUC, nil, secretHandlerAuth("write"), nil)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/myrepo/actions/secrets/public-key", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["key_id"] != "repo-key-id" {
		t.Fatalf("expected key_id repo-key-id, got %q", resp["key_id"])
	}
	if resp["key"] != "cHVibGljS2V5" {
		t.Fatalf("expected key cHVibGljS2V5, got %q", resp["key"])
	}
}

func TestSecretHandler_CrossOrgRequest_Returns404(t *testing.T) {
	secretTestAccess = handler.NewRepoAccess(denySecretMembership{}, denySecretCollab{})
	t.Cleanup(func() { secretTestAccess = nil })
	e := newSecretHandlerEcho(
		t,
		&mockListRepoSecretsUC{},
		nil,
		nil,
		nil,
		nil,
		secretHandlerAuth("write"),
		func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Set("org_id", uuid.New())
				return next(c)
			}
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/myrepo/actions/secrets", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound && rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 or 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}
