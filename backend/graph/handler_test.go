package graph_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/open-git/backend/graph"
	"github.com/open-git/backend/internal/config"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	appmiddleware "github.com/open-git/backend/internal/middleware"
)

type mockAccessTokenRepo struct {
	byHash map[string]*domain.AccessToken
}

func (m *mockAccessTokenRepo) Create(_ context.Context, _ *domain.AccessToken) error {
	return nil
}

func (m *mockAccessTokenRepo) ListByUserID(_ context.Context, _ int64) ([]*domain.AccessToken, error) {
	return nil, nil
}

func (m *mockAccessTokenRepo) Revoke(_ context.Context, _, _ int64) error {
	return nil
}

func (m *mockAccessTokenRepo) FindByTokenHash(_ context.Context, tokenHash string) (*domain.AccessToken, error) {
	if m.byHash == nil {
		return nil, nil
	}
	return m.byHash[tokenHash], nil
}

type mockUserRepo struct {
	users map[uuid.UUID]*entity.User
}

func (m *mockUserRepo) Create(_ context.Context, _ *entity.User) error { return nil }
func (m *mockUserRepo) Update(_ context.Context, _ *entity.User) error { return nil }
func (m *mockUserRepo) GetByLogin(_ context.Context, _ string) (*entity.User, error) {
	return nil, nil
}
func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*entity.User, error) {
	return nil, nil
}
func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	if m.users == nil {
		return nil, nil
	}
	return m.users[id], nil
}

type stubLabelRepo struct{}

func (stubLabelRepo) Create(context.Context, *entity.Label) error { return nil }
func (stubLabelRepo) GetByName(context.Context, uuid.UUID, string) (*entity.Label, error) {
	return nil, nil
}
func (stubLabelRepo) ListByRepo(context.Context, uuid.UUID, int, int) ([]*entity.Label, int, error) {
	return nil, 0, nil
}
func (stubLabelRepo) Update(context.Context, *entity.Label) error { return nil }
func (stubLabelRepo) Delete(context.Context, uuid.UUID) error     { return nil }
func (stubLabelRepo) AddToIssue(context.Context, uuid.UUID, int, uuid.UUID) error {
	return nil
}
func (stubLabelRepo) RemoveFromIssue(context.Context, uuid.UUID, int, uuid.UUID) error {
	return nil
}

type stubMilestoneRepo struct{}

func (stubMilestoneRepo) Create(context.Context, *entity.Milestone) error { return nil }
func (stubMilestoneRepo) GetByNumber(context.Context, uuid.UUID, int) (*entity.Milestone, error) {
	return nil, nil
}
func (stubMilestoneRepo) ListByRepo(context.Context, uuid.UUID, string, int, int) ([]*entity.Milestone, int, error) {
	return nil, 0, nil
}
func (stubMilestoneRepo) Update(context.Context, *entity.Milestone) error { return nil }
func (stubMilestoneRepo) Delete(context.Context, uuid.UUID) error         { return nil }
func (stubMilestoneRepo) NextNumber(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}
func (stubMilestoneRepo) IncrOpenCount(context.Context, uuid.UUID) error { return nil }
func (stubMilestoneRepo) DecrOpenCount(context.Context, uuid.UUID) error { return nil }

type stubRepositoryRepo struct{}

func (stubRepositoryRepo) Create(context.Context, *entity.Repository) error { return nil }
func (stubRepositoryRepo) GetByOwnerAndName(context.Context, uuid.UUID, string) (*entity.Repository, error) {
	return nil, nil
}
func (stubRepositoryRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}
func (stubRepositoryRepo) UpdateVisibility(context.Context, uuid.UUID, string) error { return nil }
func (stubRepositoryRepo) Delete(context.Context, uuid.UUID) error                   { return nil }

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func newTestResolver(userRepo domainrepo.IUserRepository) *graph.Resolver {
	return &graph.Resolver{
		UserRepo:       userRepo,
		LabelRepo:      stubLabelRepo{},
		MilestoneRepo:  stubMilestoneRepo{},
		RepositoryRepo: stubRepositoryRepo{},
	}
}

func newGraphQLEcho(resolver *graph.Resolver, cfg *config.Config, tokens *mockAccessTokenRepo) *echo.Echo {
	e := echo.New()
	authMiddleware := appmiddleware.AuthMiddleware(tokens)
	gqlHandler := graph.NewHandler(resolver, cfg)
	e.POST("/api/graphql", gqlHandler, authMiddleware)
	e.GET("/api/graphql", gqlHandler)
	return e
}

func postGraphQL(t *testing.T, e *echo.Echo, token string, query string) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(map[string]string{"query": query})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/graphql", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

type graphQLError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type graphQLResponse struct {
	Errors []graphQLError `json:"errors"`
	Data   map[string]any `json:"data"`
}

func decodeGraphQLResponse(t *testing.T, rec *httptest.ResponseRecorder) graphQLResponse {
	t.Helper()

	var resp graphQLResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp
}

func TestGraphQL_Unauthenticated(t *testing.T) {
	tokens := &mockAccessTokenRepo{}
	userRepo := &mockUserRepo{}
	e := newGraphQLEcho(newTestResolver(userRepo), &config.Config{}, tokens)

	rec := postGraphQL(t, e, "", `{ viewer { login } }`)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGraphQL_IntrospectionDisabled(t *testing.T) {
	t.Setenv("GRAPHQL_INTROSPECTION_ENABLED", "false")

	rawToken := "introspection-token"
	tokens := &mockAccessTokenRepo{
		byHash: map[string]*domain.AccessToken{
			tokenHash(rawToken): {
				UserID: 1,
				Scopes: []string{"repo"},
			},
		},
	}
	userRepo := &mockUserRepo{
		users: map[uuid.UUID]*entity.User{
			appmiddleware.Int64ToUUID(1): {ID: appmiddleware.Int64ToUUID(1), Login: "octocat"},
		},
	}
	cfg := &config.Config{GraphQLIntrospectionEnabled: false}
	e := newGraphQLEcho(newTestResolver(userRepo), cfg, tokens)

	rec := postGraphQL(t, e, rawToken, `{ __schema { queryType { name } } }`)
	require.Equal(t, http.StatusOK, rec.Code)

	resp := decodeGraphQLResponse(t, rec)
	require.NotEmpty(t, resp.Errors)
	require.Contains(t, resp.Errors[0].Message, "introspection")
}

func TestGraphQL_ComplexityLimitExceeded(t *testing.T) {
	rawToken := "complexity-token"
	tokens := &mockAccessTokenRepo{
		byHash: map[string]*domain.AccessToken{
			tokenHash(rawToken): {
				UserID: 1,
				Scopes: []string{"repo"},
			},
		},
	}
	userRepo := &mockUserRepo{
		users: map[uuid.UUID]*entity.User{
			appmiddleware.Int64ToUUID(1): {ID: appmiddleware.Int64ToUUID(1), Login: "octocat"},
		},
	}
	e := newGraphQLEcho(newTestResolver(userRepo), &config.Config{}, tokens)

	query := `{ repository(owner: "acme", name: "demo") { issues(first: 2000) { nodes { id } } } }`
	rec := postGraphQL(t, e, rawToken, query)
	require.Equal(t, http.StatusOK, rec.Code)

	resp := decodeGraphQLResponse(t, rec)
	require.NotEmpty(t, resp.Errors)
	require.Equal(t, "MAX_NODE_LIMIT_EXCEEDED", resp.Errors[0].Type)
}

func TestGraphQL_AuthenticatedViewerReachesResolver(t *testing.T) {
	rawToken := "viewer-token"
	tokens := &mockAccessTokenRepo{
		byHash: map[string]*domain.AccessToken{
			tokenHash(rawToken): {
				UserID: 1,
				Scopes: []string{"read:user"},
				ExpiresAt: func() *time.Time {
					t := time.Now().UTC().Add(time.Hour)
					return &t
				}(),
			},
		},
	}
	userRepo := &mockUserRepo{
		users: map[uuid.UUID]*entity.User{
			appmiddleware.Int64ToUUID(1): {ID: appmiddleware.Int64ToUUID(1), Login: "octocat"},
		},
	}
	e := newGraphQLEcho(newTestResolver(userRepo), &config.Config{}, tokens)

	rec := postGraphQL(t, e, rawToken, `{ viewer { login } }`)
	require.Equal(t, http.StatusOK, rec.Code)

	resp := decodeGraphQLResponse(t, rec)
	require.NotEmpty(t, resp.Errors)
	require.Contains(t, resp.Errors[0].Message, "not implemented")
}

func TestMain(m *testing.M) {
	os.Unsetenv("GRAPHQL_INTROSPECTION_ENABLED")
	os.Exit(m.Run())
}
