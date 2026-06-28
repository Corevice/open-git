package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"

	"github.com/open-git/backend/internal/handler"
)

type stubMinioClient struct {
	err error
}

func (s stubMinioClient) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	if s.err != nil {
		return nil, s.err
	}
	return []minio.BucketInfo{}, nil
}

type stubRedisClient struct {
	err error
}

func (s stubRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "ping")
	if s.err != nil {
		cmd.SetErr(s.err)
	} else {
		cmd.SetVal("PONG")
	}
	return cmd
}

func newHealthTestDB(t *testing.T, pingErr error) *sqlx.DB {
	t.Helper()

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = mockDB.Close() })

	if pingErr != nil {
		mock.ExpectPing().WillReturnError(pingErr)
	} else {
		mock.ExpectPing()
	}

	return sqlx.NewDb(mockDB, "postgres")
}

func serveHealth(t *testing.T, db *sqlx.DB, minio stubMinioClient, redis stubRedisClient) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	h := handler.NewAPIV1HealthHandler(db, minio, redis)
	e.GET("/api/v1/health", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestAPIV1HealthHandler_Healthy(t *testing.T) {
	rec := serveHealth(t, newHealthTestDB(t, nil), stubMinioClient{}, stubRedisClient{})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"status", "db", "storage", "queue"} {
		if body.Data[key] != "ok" {
			t.Fatalf("data.%s = %q, want %q", key, body.Data[key], "ok")
		}
	}
}

func TestAPIV1HealthHandler_DBFail(t *testing.T) {
	rec := serveHealth(
		t,
		newHealthTestDB(t, errors.New("connection refused")),
		stubMinioClient{},
		stubRedisClient{},
	)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Data["db"] != "error" {
		t.Fatalf("data.db = %q, want %q", body.Data["db"], "error")
	}
	if body.Data["status"] != "error" {
		t.Fatalf("data.status = %q, want %q", body.Data["status"], "error")
	}
}

func TestAPIV1HealthHandler_MinIOFail(t *testing.T) {
	rec := serveHealth(
		t,
		newHealthTestDB(t, nil),
		stubMinioClient{err: errors.New("minio unavailable")},
		stubRedisClient{},
	)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Data["storage"] != "error" {
		t.Fatalf("data.storage = %q, want %q", body.Data["storage"], "error")
	}
	if body.Data["status"] != "error" {
		t.Fatalf("data.status = %q, want %q", body.Data["status"], "error")
	}
}

func TestAPIV1HealthHandler_RedisFail(t *testing.T) {
	rec := serveHealth(
		t,
		newHealthTestDB(t, nil),
		stubMinioClient{},
		stubRedisClient{err: errors.New("redis unavailable")},
	)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Data["queue"] != "error" {
		t.Fatalf("data.queue = %q, want %q", body.Data["queue"], "error")
	}
	if body.Data["status"] != "error" {
		t.Fatalf("data.status = %q, want %q", body.Data["status"], "error")
	}
}
