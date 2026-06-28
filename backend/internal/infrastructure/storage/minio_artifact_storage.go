package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/open-git/backend/internal/domain/service"
)

var _ service.ArtifactStorage = (*MinioArtifactStorage)(nil)

type MinioArtifactStorage struct {
	client *minio.Client
}

func NewMinioArtifactStorage(endpoint, accessKey, secretKey string, useTLS bool) (*MinioArtifactStorage, error) {
	if endpoint == "" {
		return nil, errors.New("minio endpoint is required")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &MinioArtifactStorage{client: client}, nil
}

func (s *MinioArtifactStorage) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket exists: %w", err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("make bucket: %w", err)
	}
	return nil
}

func (s *MinioArtifactStorage) PresignedPutURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedPutObject(ctx, bucket, key, expiry)
	if err != nil {
		return "", fmt.Errorf("presigned put url: %w", err)
	}
	return url.String(), nil
}

func (s *MinioArtifactStorage) PresignedGetURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("presigned get url: %w", err)
	}
	return url.String(), nil
}

func (s *MinioArtifactStorage) DeleteObject(ctx context.Context, bucket, key string) error {
	if err := s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object: %w", err)
	}
	return nil
}

func (s *MinioArtifactStorage) DeleteObjectsByPrefix(ctx context.Context, bucket, prefix string) error {
	for obj := range s.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if obj.Err != nil {
			return fmt.Errorf("list objects: %w", obj.Err)
		}
		if err := s.client.RemoveObject(ctx, bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
			return fmt.Errorf("remove object %q: %w", obj.Key, err)
		}
	}
	return nil
}
