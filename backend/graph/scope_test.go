package graph

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequireScopePresent(t *testing.T) {
	ctx := WithScopes(context.Background(), []string{"read", ScopeRepo})

	err := RequireScope(ctx, ScopeRepo)
	require.NoError(t, err)
}

func TestRequireScopeAbsent(t *testing.T) {
	ctx := WithScopes(context.Background(), []string{"read"})

	err := RequireScope(ctx, ScopeRepo)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrInsufficientScopes))
}

func TestRequireScopeMissingContext(t *testing.T) {
	err := RequireScope(context.Background(), ScopeRepo)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrInsufficientScopes))
}
