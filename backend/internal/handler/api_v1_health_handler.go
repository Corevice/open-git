package handler

import (
	"context"
	"net/http"

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
	return &APIV1HealthHandler{
		db:          db,
		minioClient: minioClient,
		redisClient: redisClient,
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
