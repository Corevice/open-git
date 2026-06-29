package service

import (
	"context"
	"time"
)

type ArtifactStorage interface {
	EnsureBucket(ctx context.Context, bucket string) error
	PresignedPutURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
	PresignedGetURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
	DeleteObject(ctx context.Context, bucket, key string) error
	DeleteObjectsByPrefix(ctx context.Context, bucket, prefix string) error
}
