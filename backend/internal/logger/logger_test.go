package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"

	"github.com/open-git/backend/internal/domain"
)

func initTestLogger(buf *bytes.Buffer) {
	glog = zerolog.New(buf).With().Timestamp().Logger()
}

func TestFromContext_AuthenticatedRequest(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf)

	userID := int64(42)
	orgID := int64(99)
	ctx := domain.WithRequestContext(context.Background(), domain.RequestContext{
		RequestID:      "req-auth-123",
		ActorUserID:    &userID,
		OrganizationID: &orgID,
	})

	l := FromContext(ctx)
	l.Info().Msg("authenticated")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log output: %v; raw=%q", err, buf.String())
	}

	if entry["request_id"] != "req-auth-123" {
		t.Fatalf("expected request_id req-auth-123, got %v", entry["request_id"])
	}
	if entry["user_id"] != float64(42) {
		t.Fatalf("expected user_id 42, got %v", entry["user_id"])
	}
	if entry["organization_id"] != float64(99) {
		t.Fatalf("expected organization_id 99, got %v", entry["organization_id"])
	}
}

func TestFromContext_UnauthenticatedRequest(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf)

	ctx := domain.WithRequestContext(context.Background(), domain.RequestContext{
		RequestID:      "req-unauth-456",
		ActorUserID:    nil,
		OrganizationID: nil,
	})

	l := FromContext(ctx)
	l.Info().Msg("unauthenticated")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log output: %v; raw=%q", err, buf.String())
	}

	if entry["request_id"] != "req-unauth-456" {
		t.Fatalf("expected request_id req-unauth-456, got %v", entry["request_id"])
	}
	if _, ok := entry["user_id"]; ok {
		t.Fatalf("expected user_id to be absent, got %v", entry["user_id"])
	}
	if _, ok := entry["organization_id"]; ok {
		t.Fatalf("expected organization_id to be absent, got %v", entry["organization_id"])
	}
}

func TestFromContext_NoContextKey(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf)

	ctx := context.Background()

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("FromContext panicked: %v", r)
			}
		}()
		l := FromContext(ctx)
		l.Info().Msg("no context")
	}()

	if buf.Len() == 0 {
		t.Fatal("expected log output from default logger")
	}
}
