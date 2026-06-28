package handler

import (
	"context"
	"net/http"

	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/v3/disk"
)

const asynqDefaultPendingQueueKey = "asynq:{default}:pending"

type adminStatusRedisClient interface {
	Ping(ctx context.Context) *redis.StatusCmd
	LLen(ctx context.Context, key string) *redis.IntCmd
}

type asynqQueueInspector interface {
	GetQueueInfo(queue string) (*asynq.QueueInfo, error)
}

type APIV1AdminStatusHandler struct {
	db          *sqlx.DB
	minioClient minioHealthClient
	redisClient adminStatusRedisClient
	inspector   asynqQueueInspector
	storagePath string
}

func NewAPIV1AdminStatusHandler(
	db *sqlx.DB,
	minioClient minioHealthClient,
	redisClient adminStatusRedisClient,
	inspector asynqQueueInspector,
	storagePath string,
) *APIV1AdminStatusHandler {
	if storagePath == "" {
		storagePath = "/"
	}
	return &APIV1AdminStatusHandler{
		db:          db,
		minioClient: minioClient,
		redisClient: redisClient,
		inspector:   inspector,
		storagePath: storagePath,
	}
}

func (h *APIV1AdminStatusHandler) Handle(c echo.Context) error {
	if !isSiteAdmin(c) {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
	}

	ctx := c.Request().Context()

	dbStatus := "ok"
	if err := h.db.PingContext(ctx); err != nil {
		dbStatus = "error"
	}

	storageStatus := "ok"
	var storageFree, storageTotal uint64
	if usage, err := disk.Usage(h.storagePath); err != nil {
		storageStatus = "error"
	} else {
		storageFree = usage.Free
		storageTotal = usage.Total
	}
	if h.minioClient != nil {
		if _, err := h.minioClient.ListBuckets(ctx); err != nil {
			storageStatus = "error"
		}
	}

	queueStatus := "ok"
	var queueDepth int64
	if h.redisClient != nil {
		if err := h.redisClient.Ping(ctx).Err(); err != nil {
			queueStatus = "error"
		} else if depth, err := h.redisClient.LLen(ctx, asynqDefaultPendingQueueKey).Result(); err != nil {
			queueStatus = "error"
		} else {
			queueDepth = depth
		}
	} else if h.inspector != nil {
		info, err := h.inspector.GetQueueInfo("default")
		if err != nil {
			queueStatus = "error"
		} else {
			queueDepth = int64(info.Pending)
		}
	}

	dbConnections := 0
	if h.db != nil && h.db.DB != nil {
		dbConnections = h.db.Stats().OpenConnections
	}

	body := map[string]any{
		"db":                  dbStatus,
		"storage":             storageStatus,
		"queue":               queueStatus,
		"db_connections":      dbConnections,
		"queue_depth":         queueDepth,
		"storage_free_bytes":  storageFree,
		"storage_total_bytes": storageTotal,
	}

	return RespondOK(c, body)
}
