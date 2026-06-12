package domain

import "context"

type RequestContext struct {
	RequestID      string
	ActorUserID    *int64
	OrganizationID *int64
}

type ctxKey struct{}

func WithRequestContext(ctx context.Context, rc RequestContext) context.Context {
	return context.WithValue(ctx, ctxKey{}, rc)
}

func GetRequestContext(ctx context.Context) (RequestContext, bool) {
	rc, ok := ctx.Value(ctxKey{}).(RequestContext)
	return rc, ok
}
