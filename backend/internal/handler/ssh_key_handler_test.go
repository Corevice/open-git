package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
)

type stubSSHKeyStore struct {
	keys []*entity.SSHKey
}

func (s *stubSSHKeyStore) FindByFingerprint(_ context.Context, _ string) (*entity.SSHKey, error) {
	return nil, nil
}

func (s *stubSSHKeyStore) ListByUserID(_ context.Context, _ uuid.UUID) ([]*entity.SSHKey, error) {
	return s.keys, nil
}

func (s *stubSSHKeyStore) Create(_ context.Context, key *entity.SSHKey) error {
	if key.ID == uuid.Nil {
		key.ID = uuid.New()
	}
	if key.CreatedAt.IsZero() {
		key.CreatedAt = time.Now().UTC()
	}
	if key.Fingerprint == "" {
		key.Fingerprint = "SHA256:stub"
	}
	s.keys = append(s.keys, key)
	return nil
}

func (s *stubSSHKeyStore) Delete(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

const validSSHPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOzRANdrmNo46uGr2ky5ETd7ObwPSeqqxgc/K27LwS1P test@example.com"

func newSSHKeyEcho(t *testing.T, store *stubSSHKeyStore, userID int64) *echo.Echo {
	t.Helper()

	e := echo.New()
	h := handler.NewSSHKeyHandler(store)

	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user_id", userID)
			return next(c)
		}
	}

	keys := e.Group("/user/keys", auth)
	keys.GET("", h.List)
	keys.POST("", h.Add)
	return e
}

func TestAddSSHKey_InvalidKeyFormat(t *testing.T) {
	store := &stubSSHKeyStore{}
	e := newSSHKeyEcho(t, store, int64(7))

	body := `{"title":"bad","key":"not-a-valid-key"}`
	req := httptest.NewRequest(http.MethodPost, "/user/keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestAddSSHKey_Valid(t *testing.T) {
	store := &stubSSHKeyStore{}
	e := newSSHKeyEcho(t, store, int64(7))

	payload, err := json.Marshal(map[string]string{
		"title": "laptop",
		"key":   validSSHPublicKey,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/user/keys", bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["fingerprint"] == "" {
		t.Fatal("expected fingerprint in response")
	}
	if !strings.HasPrefix(resp["fingerprint"], "SHA256:") {
		t.Fatalf("fingerprint = %q, want SHA256 prefix", resp["fingerprint"])
	}
}

func TestListSSHKeys_Empty(t *testing.T) {
	store := &stubSSHKeyStore{keys: []*entity.SSHKey{}}
	e := newSSHKeyEcho(t, store, int64(7))

	req := httptest.NewRequest(http.MethodGet, "/user/keys", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp))
	}
}
