package handler_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
)

type adminStatusMinioClient struct{}

func (adminStatusMinioClient) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	return []minio.BucketInfo{}, nil
}

type adminStatusRedisClient struct {
	queueDepth int64
}

func (s adminStatusRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "ping")
	cmd.SetVal("PONG")
	return cmd
}

func (s adminStatusRedisClient) LLen(ctx context.Context, key string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "llen", key)
	cmd.SetVal(s.queueDepth)
	return cmd
}

type stubAsynqInspector struct {
	pending int
}

func (s stubAsynqInspector) GetQueueInfo(string) (*asynq.QueueInfo, error) {
	return &asynq.QueueInfo{Pending: s.pending}, nil
}

type adminStatusTokenRepo struct {
	byHash map[string]*domain.AccessToken
}

func (m adminStatusTokenRepo) Create(context.Context, *domain.AccessToken) error { return nil }
func (m adminStatusTokenRepo) ListByUserID(context.Context, int64) ([]*domain.AccessToken, error) {
	return nil, nil
}
func (m adminStatusTokenRepo) Revoke(context.Context, int64, int64) error { return nil }
func (m adminStatusTokenRepo) FindByTokenHash(_ context.Context, tokenHash string) (*domain.AccessToken, error) {
	return m.byHash[tokenHash], nil
}

func adminStatusTokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func newAdminStatusTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = mockDB.Close() })
	mock.ExpectPing()

	return sqlx.NewDb(mockDB, "postgres")
}

func newAdminStatusEcho(t *testing.T, siteAdmin bool) *echo.Echo {
	t.Helper()

	repo := adminStatusTokenRepo{
		byHash: map[string]*domain.AccessToken{
			adminStatusTokenHash("valid-token"): {
				UserID: 42,
				Scopes: []string{"read"},
			},
		},
	}

	e := echo.New()
	e.Use(middleware.AuthMiddleware(repo))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if siteAdmin {
				c.Set("is_admin", true)
			}
			return next(c)
		}
	})

	h := handler.NewAPIV1AdminStatusHandler(
		newAdminStatusTestDB(t),
		adminStatusMinioClient{},
		adminStatusRedisClient{queueDepth: 2},
		stubAsynqInspector{pending: 2},
		"/",
	)
	e.GET("/api/v1/admin/status", h.Handle)
	return e
}

func serveAdminStatus(t *testing.T, e *echo.Echo, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/status", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestAPIV1AdminStatusHandler_Unauthenticated(t *testing.T) {
	e := newAdminStatusEcho(t, false)
	rec := serveAdminStatus(t, e, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestAPIV1AdminStatusHandler_NonAdmin(t *testing.T) {
	e := newAdminStatusEcho(t, false)
	rec := serveAdminStatus(t, e, "valid-token")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestAPIV1AdminStatusHandler_SiteAdmin(t *testing.T) {
	e := newAdminStatusEcho(t, true)
	rec := serveAdminStatus(t, e, "valid-token")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"db", "storage", "queue", "db_connections", "queue_depth"} {
		if _, ok := body.Data[key]; !ok {
			t.Fatalf("data missing key %q: %#v", key, body.Data)
		}
	}

	if body.Data["db"] != "ok" {
		t.Fatalf("data.db = %v, want %q", body.Data["db"], "ok")
	}
	if body.Data["storage"] != "ok" {
		t.Fatalf("data.storage = %v, want %q", body.Data["storage"], "ok")
	}
	if body.Data["queue"] != "ok" {
		t.Fatalf("data.queue = %v, want %q", body.Data["queue"], "ok")
	}
	if body.Data["queue_depth"] != float64(2) {
		t.Fatalf("data.queue_depth = %v, want 2", body.Data["queue_depth"])
	}
}
