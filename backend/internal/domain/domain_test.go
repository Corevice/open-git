package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/open-git/backend/internal/domain"
)

func TestRequestContextRoundTrip(t *testing.T) {
	actorID := int64(42)
	orgID := int64(7)
	rc := domain.RequestContext{
		RequestID:      "req-123",
		ActorUserID:    &actorID,
		OrganizationID: &orgID,
	}

	ctx := domain.WithRequestContext(context.Background(), rc)
	got, ok := domain.GetRequestContext(ctx)
	if !ok {
		t.Fatal("expected request context to be present")
	}
	if got.RequestID != rc.RequestID {
		t.Fatalf("expected request id %q, got %q", rc.RequestID, got.RequestID)
	}
	if got.ActorUserID == nil || *got.ActorUserID != actorID {
		t.Fatalf("expected actor user id %d, got %v", actorID, got.ActorUserID)
	}
	if got.OrganizationID == nil || *got.OrganizationID != orgID {
		t.Fatalf("expected organization id %d, got %v", orgID, got.OrganizationID)
	}
}

func TestDomainErrorErrorFormat(t *testing.T) {
	err := &domain.DomainError{
		Code:    "not_found",
		Message: "user not found",
		Err:     domain.ErrNotFound,
	}

	msg := err.Error()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
	if msg != "not_found: user not found" {
		t.Fatalf("unexpected error message: %q", msg)
	}
}

func TestDomainErrorUnwrap(t *testing.T) {
	inner := domain.ErrConflict
	err := &domain.DomainError{
		Code:    "conflict",
		Message: "duplicate resource",
		Err:     inner,
	}

	if !errors.Is(err, inner) {
		t.Fatal("expected DomainError to unwrap to inner error")
	}
	if err.Unwrap() != inner {
		t.Fatal("Unwrap() should return inner Err")
	}
}

func TestNoopUnitOfWorkDoHappyPath(t *testing.T) {
	uow := domain.NoopUnitOfWork{}
	called := false

	err := uow.Do(context.Background(), func(ctx context.Context) error {
		called = true
		if ctx == nil {
			t.Fatal("expected non-nil context")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected callback to be executed")
	}
}

func TestNoopUnitOfWorkDoErrorPropagation(t *testing.T) {
	uow := domain.NoopUnitOfWork{}
	expected := errors.New("boom")

	err := uow.Do(context.Background(), func(context.Context) error {
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}
