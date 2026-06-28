package storage_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/open-git/backend/internal/infrastructure/storage"
)

type fakeMinioOps struct {
	bucketExists     bool
	makeBucketCalled bool
	makeBucketName   string
}

func (f *fakeMinioOps) handleBucket(w http.ResponseWriter, r *http.Request) {
	bucket := strings.TrimPrefix(r.URL.Path, "/")
	if bucket == "" {
		http.Error(w, "missing bucket", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodHead:
		if f.bucketExists {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	case http.MethodPut:
		f.makeBucketCalled = true
		f.makeBucketName = bucket
		f.bucketExists = true
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func TestNewMinioArtifactStorage_ValidEndpoint(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	host, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	client, err := storage.NewMinioArtifactStorage(host.Host, "minioadmin", "minioadmin", false)
	if err != nil {
		t.Fatalf("NewMinioArtifactStorage: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewMinioArtifactStorage_EmptyEndpoint(t *testing.T) {
	t.Parallel()

	_, err := storage.NewMinioArtifactStorage("", "minioadmin", "minioadmin", false)
	if err == nil {
		t.Fatal("expected error for empty endpoint")
	}
}

func TestEnsureBucket_CallsMakeBucketWhenNotExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping minio integration test in short mode")
	}

	ops := &fakeMinioOps{bucketExists: false}
	srv := httptest.NewServer(http.HandlerFunc(ops.handleBucket))
	t.Cleanup(srv.Close)

	host, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	client, err := storage.NewMinioArtifactStorage(host.Host, "minioadmin", "minioadmin", false)
	if err != nil {
		t.Fatalf("NewMinioArtifactStorage: %v", err)
	}

	const bucket = "artifacts"
	if err := client.EnsureBucket(context.Background(), bucket); err != nil {
		t.Fatalf("EnsureBucket: %v", err)
	}
	if !ops.makeBucketCalled {
		t.Fatal("expected MakeBucket to be called")
	}
	if ops.makeBucketName != bucket {
		t.Fatalf("MakeBucket bucket = %q, want %q", ops.makeBucketName, bucket)
	}
}
