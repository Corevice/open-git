package artifact

import (
	"context"
	"time"
)

type ArtifactStorage interface {
	PresignedPutURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
	PresignedGetURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
	DeleteObject(ctx context.Context, bucket, key string) error
}
