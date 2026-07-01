package handler

import (
	"context"
	"net/http"
	"reflect"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
)

type minioHealthClient interface {
	ListBuckets(ctx context.Context) ([]minio.BucketInfo, error)
}

type redisHealthClient interface {
	Ping(ctx context.Context) *redis.StatusCmd
}

type APIV1HealthHandler struct {
	db          *sqlx.DB
	minioClient minioHealthClient
	redisClient redisHealthClient
}

func NewAPIV1HealthHandler(db *sqlx.DB, minioClient minioHealthClient, redisClient redisHealthClient) *APIV1HealthHandler {
	h := &APIV1HealthHandler{db: db}
	// Guard against the typed-nil interface gotcha: callers pass concrete
	// *minio.Client / *redis.Client that may be nil when those dependencies are
	// not configured. Assigned directly, the interface would be non-nil (it
	// carries a type), so the `!= nil` checks in Handle would pass and then
	// panic dereferencing the nil pointer. Normalize typed-nil to a real nil.
	if !isNilValue(minioClient) {
		h.minioClient = minioClient
	}
	if !isNilValue(redisClient) {
		h.redisClient = redisClient
	}
	return h
}

func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return rv.IsNil()
	default:
		return false
	}
}

func (h *APIV1HealthHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	dbStatus := "ok"
	if err := h.db.PingContext(ctx); err != nil {
		dbStatus = "error"
	}

	storageStatus := "ok"
	if h.minioClient != nil {
		if _, err := h.minioClient.ListBuckets(ctx); err != nil {
			storageStatus = "error"
		}
	}

	queueStatus := "ok"
	if h.redisClient != nil {
		if err := h.redisClient.Ping(ctx).Err(); err != nil {
			queueStatus = "error"
		}
	}

	overallStatus := "ok"
	if dbStatus != "ok" || storageStatus != "ok" || queueStatus != "ok" {
		overallStatus = "error"
	}

	body := map[string]string{
		"status":  overallStatus,
		"db":      dbStatus,
		"storage": storageStatus,
		"queue":   queueStatus,
	}

	if overallStatus == "ok" {
		return RespondOK(c, body)
	}
	return c.JSON(http.StatusServiceUnavailable, apiResponse{Data: body})
}
