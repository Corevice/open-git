package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestCheckRESTPass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"login":"octocat"}`))
	}))
	t.Cleanup(server.Close)

	worker := NewMCPVerificationWorker(nil, server.URL).WithHTTPClient(server.Client())
	check := worker.performHTTPCheck(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"rest.user",
		entity.CheckCategoryREST,
		"/api/v3/user",
		"",
		http.StatusOK,
	)

	if check.Status != entity.CheckStatusPass {
		t.Fatalf("expected pass, got %s", check.Status)
	}
}

func TestCheckRESTFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	worker := NewMCPVerificationWorker(nil, server.URL).WithHTTPClient(server.Client())
	check := worker.performHTTPCheck(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"rest.user",
		entity.CheckCategoryREST,
		"/api/v3/user",
		"",
		http.StatusOK,
	)

	if check.Status != entity.CheckStatusFail {
		t.Fatalf("expected fail, got %s", check.Status)
	}
}

func TestCheckRESTTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	worker := NewMCPVerificationWorker(nil, server.URL).WithHTTPClient(server.Client())
	check := worker.performHTTPCheck(
		ctx,
		uuid.New(),
		uuid.New(),
		"rest.user",
		entity.CheckCategoryREST,
		"/api/v3/user",
		"",
		http.StatusOK,
	)

	if check.Status != entity.CheckStatusFail {
		t.Fatalf("expected fail, got %s", check.Status)
	}
	if check.Error == nil || *check.Error != "timeout" {
		t.Fatalf("expected timeout error, got %v", check.Error)
	}
}

func TestCheck429Retry(t *testing.T) {
	originalDelays := mcpRetryDelays
	mcpRetryDelays = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	t.Cleanup(func() { mcpRetryDelays = originalDelays })

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"login":"octocat"}`))
	}))
	t.Cleanup(server.Close)

	worker := NewMCPVerificationWorker(nil, server.URL).WithHTTPClient(server.Client())
	check := worker.performHTTPCheck(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"rest.user",
		entity.CheckCategoryREST,
		"/api/v3/user",
		"",
		http.StatusOK,
	)

	if check.Status != entity.CheckStatusPass {
		t.Fatalf("expected pass after retry, got %s", check.Status)
	}
	if attempts.Load() < 2 {
		t.Fatalf("expected at least 2 attempts, got %d", attempts.Load())
	}
}
