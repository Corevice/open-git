package graph

import "context"

const (
	ScopeRepo    = "repo"
	ScopeReadOrg = "read:org"
)

func RequireScope(ctx context.Context, scope string) error {
	for _, s := range ScopesFromContext(ctx) {
		if s == scope {
			return nil
		}
	}
	return ErrInsufficientScopes
}
