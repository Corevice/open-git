package graph

import (
	"context"

	"github.com/open-git/backend/graph/dataloader"
	"github.com/open-git/backend/internal/domain/entity"
)

type contextKey int

const (
	viewerContextKey contextKey = iota + 1
	scopesContextKey
	loadersContextKey
)

func WithViewer(ctx context.Context, viewer *entity.User) context.Context {
	return context.WithValue(ctx, viewerContextKey, viewer)
}

func WithScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, scopesContextKey, scopes)
}

func WithLoaders(ctx context.Context, loaders *dataloader.Loaders) context.Context {
	return context.WithValue(ctx, loadersContextKey, loaders)
}

func ViewerFromContext(ctx context.Context) (*entity.User, bool) {
	viewer, ok := ctx.Value(viewerContextKey).(*entity.User)
	if !ok || viewer == nil {
		return nil, false
	}
	return viewer, true
}

func ScopesFromContext(ctx context.Context) []string {
	scopes, ok := ctx.Value(scopesContextKey).([]string)
	if !ok {
		return nil
	}
	return scopes
}

func LoadersFromContext(ctx context.Context) *dataloader.Loaders {
	loaders, ok := ctx.Value(loadersContextKey).(*dataloader.Loaders)
	if !ok {
		return nil
	}
	return loaders
}
