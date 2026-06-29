package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/labstack/echo/v4"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/open-git/backend/graph/dataloader"
	"github.com/open-git/backend/graph/generated"
	"github.com/open-git/backend/internal/config"
	appmiddleware "github.com/open-git/backend/internal/middleware"
)

func NewHandler(resolver *Resolver, cfg *config.Config) echo.HandlerFunc {
	srv := handler.New(generated.NewExecutableSchema(generated.Config{
		Resolvers:  resolver,
		Complexity: NewComplexityRoot(),
	}))

	srv.SetErrorPresenter(GitHubErrorPresenter)
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})
	srv.Use(NewDepthLimiter())
	srv.Use(extension.FixedComplexityLimit(ComplexityLimit))

	if cfg != nil && cfg.GraphQLIntrospectionEnabled {
		srv.Use(extension.Introspection{})
	} else {
		srv.AroundOperations(blockIntrospectionOperations)
	}

	loaderMiddleware := dataloader.Middleware(
		resolver.UserRepo,
		resolver.LabelRepo,
		resolver.MilestoneRepo,
		resolver.RepositoryRepo,
	)
	playgroundHandler := playground.Handler("GraphQL playground", "/api/graphql")

	return func(c echo.Context) error {
		if isDevMode() && c.Request().Method == http.MethodGet && isGraphQLPlaygroundRequest(c) {
			playgroundHandler.ServeHTTP(c.Response(), c.Request())
			return nil
		}

		if err := runLoaderMiddleware(loaderMiddleware, c); err != nil {
			return err
		}

		ctx := c.Request().Context()
		userID := appmiddleware.UserIDFromContext(c)
		if userID != 0 {
			if user, err := resolver.UserRepo.GetByID(ctx, appmiddleware.Int64ToUUID(userID)); err == nil && user != nil {
				ctx = WithViewer(ctx, user)
			}
			ctx = WithScopes(ctx, appmiddleware.GetScopes(c))
		}
		if loaders := dataloader.FromEcho(c); loaders != nil {
			ctx = WithLoaders(ctx, loaders)
		}
		c.SetRequest(c.Request().WithContext(ctx))

		recorder := &responseRecorder{ResponseWriter: c.Response().Writer}
		c.Response().Writer = recorder
		srv.ServeHTTP(recorder, c.Request())

		body, err := formatGitHubGraphQLResponse(recorder.body.Bytes())
		if err != nil {
			return err
		}
		if recorder.statusCode != 0 {
			c.Response().Writer = recorder.ResponseWriter
			c.Response().WriteHeader(recorder.statusCode)
		}
		c.Response().Writer = recorder.ResponseWriter
		_, err = c.Response().Writer.Write(body)
		return err
	}
}

func runLoaderMiddleware(mw echo.MiddlewareFunc, c echo.Context) error {
	h := mw(func(c echo.Context) error { return nil })
	return h(c)
}

func blockIntrospectionOperations(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	opCtx := graphql.GetOperationContext(ctx)
	if opCtx != nil && isIntrospectionOperation(opCtx.OperationName, opCtx.RawQuery) {
		return func(_ context.Context) *graphql.Response {
			return &graphql.Response{
				Errors: gqlerror.List{gqlerror.Errorf("GraphQL introspection is disabled")},
			}
		}
	}
	return next(ctx)
}

func isIntrospectionOperation(name, rawQuery string) bool {
	if strings.EqualFold(name, "IntrospectionQuery") {
		return true
	}
	lower := strings.ToLower(rawQuery)
	return strings.Contains(lower, "__schema") || strings.Contains(lower, "__type")
}

func isDevMode() bool {
	switch strings.ToLower(os.Getenv("APP_ENV")) {
	case "development", "dev", "local":
		return true
	default:
		return strings.EqualFold(os.Getenv("DEV"), "true")
	}
}

func isGraphQLPlaygroundRequest(c echo.Context) bool {
	if c.QueryParam("query") != "" {
		return false
	}
	accept := c.Request().Header.Get("Accept")
	return strings.Contains(accept, "text/html")
}

type responseRecorder struct {
	http.ResponseWriter
	body       bytes.Buffer
	statusCode int
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

type githubGraphQLError struct {
	Type      string              `json:"type,omitempty"`
	Message   string              `json:"message"`
	Path      any                 `json:"path,omitempty"`
	Locations []gqlerror.Location `json:"locations,omitempty"`
}

type githubGraphQLResponse struct {
	Data   json.RawMessage      `json:"data,omitempty"`
	Errors []githubGraphQLError `json:"errors,omitempty"`
}

func formatGitHubGraphQLResponse(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return raw, nil
	}

	var payload struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message    string              `json:"message"`
			Path       any                 `json:"path"`
			Locations  []gqlerror.Location `json:"locations"`
			Extensions map[string]any      `json:"extensions"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return raw, nil
	}
	if len(payload.Errors) == 0 {
		return raw, nil
	}

	out := githubGraphQLResponse{Data: payload.Data}
	for _, errItem := range payload.Errors {
		errType := ""
		if errItem.Extensions != nil {
			if value, ok := errItem.Extensions["type"].(string); ok {
				errType = value
			}
		}
		if errType == "" {
			errType = inferErrorType(errItem.Message)
		}
		out.Errors = append(out.Errors, githubGraphQLError{
			Type:      errType,
			Message:   sanitizeErrorMessage(errItem.Message),
			Path:      errItem.Path,
			Locations: errItem.Locations,
		})
	}

	return json.Marshal(out)
}
