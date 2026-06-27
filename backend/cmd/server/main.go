package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/config"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/infrastructure/database"
	infrarepo "github.com/open-git/backend/internal/infrastructure/repository"
	appmw "github.com/open-git/backend/internal/middleware"
	authUC "github.com/open-git/backend/internal/usecase/auth"
	issueUC "github.com/open-git/backend/internal/usecase/issue"
	prUC "github.com/open-git/backend/internal/usecase/pr"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
	repointer "github.com/open-git/backend/internal/repository"
)

var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	if err := database.Ping(context.Background(), db); err != nil {
		log.Fatalf("ping database: %v", err)
	}
	log.Printf("database connected (%s): %s", cfg.DBType, database.MaskDSN(cfg.DBDSN))

	if cfg.DBAutoMigrate {
		if err := database.RunMigrations(db, cfg.DBType, "./migrations"); err != nil {
			log.Fatalf("run migrations: %v", err)
		}
	}

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = newHTTPErrorHandler()

	e.Use(middleware.RequestID())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339_nano}","method":"${method}","path":"${path}","status":${status},"latency_ms":"${latency}","request_id":"${id}"}` + "\n",
	}))
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			c.Logger().Errorf("panic recovered: %v\n%s", err, stack)
			return nil
		},
	}))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: corsAllowedOrigins(),
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, echo.HeaderXRequestID},
	}))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{Timeout: 30 * time.Second}))
	e.Use(requestContextMiddleware())

	e.GET("/healthz", healthzHandler)
	e.GET("/readyz", readyzHandler(db))
	e.GET("/version", versionHandler)

	registerHandlers(e, cfg, db)

	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown server: %v", err)
	}
}

func corsAllowedOrigins() []string {
	origin := os.Getenv("CORS_ALLOWED_ORIGINS")
	if origin == "" {
		return []string{"*"}
	}
	parts := strings.Split(origin, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}

func requestContextMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := c.Response().Header().Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = c.Request().Header.Get(echo.HeaderXRequestID)
			}

			ctx := domain.WithRequestContext(c.Request().Context(), domain.RequestContext{
				RequestID: requestID,
			})
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func newHTTPErrorHandler() echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		requestID := c.Response().Header().Get(echo.HeaderXRequestID)
		if requestID == "" {
			requestID = c.Request().Header.Get(echo.HeaderXRequestID)
		}

		var he *echo.HTTPError
		if errors.As(err, &he) {
			message := httpErrorMessage(he)
			code := httpStatusToCode(he.Code)
			if writeErr := handler.RespondError(c, he.Code, code, message, requestID); writeErr != nil {
				c.Logger().Error(writeErr)
			}
			return
		}

		status, code := handler.DomainErrorToHTTP(err)
		if writeErr := handler.RespondError(c, status, code, err.Error(), requestID); writeErr != nil {
			c.Logger().Error(writeErr)
		}
	}
}

func httpErrorMessage(he *echo.HTTPError) string {
	switch msg := he.Message.(type) {
	case string:
		return msg
	case error:
		return msg.Error()
	default:
		return fmt.Sprintf("%v", msg)
	}
}

func httpStatusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return handler.CodeInvalidRequest
	case http.StatusUnauthorized:
		return handler.CodeUnauthorized
	case http.StatusForbidden:
		return handler.CodeForbidden
	case http.StatusNotFound:
		return handler.CodeNotFound
	case http.StatusConflict:
		return handler.CodeConflict
	case http.StatusUnsupportedMediaType:
		return handler.CodeUnsupportedMediaType
	case http.StatusUnprocessableEntity:
		return handler.CodeValidationFailed
	case http.StatusTooManyRequests:
		return handler.CodeRateLimited
	case http.StatusServiceUnavailable:
		return handler.CodeServiceUnavailable
	default:
		return handler.CodeInternal
	}
}

func healthzHandler(c echo.Context) error {
	return handler.RespondOK(c, map[string]string{"status": "ok"})
}

func readyzHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := db.PingContext(c.Request().Context()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]any{
				"data": map[string]string{"db": "down"},
			})
		}
		return handler.RespondOK(c, map[string]string{"db": "ok"})
	}
}

func versionHandler(c echo.Context) error {
	return handler.RespondOK(c, map[string]string{
		"version":   version,
		"commit":    commit,
		"buildTime": buildTime,
	})
}

func registerHandlers(e *echo.Echo, cfg config.Config, db *sql.DB) {
	sqlxDB := sqlx.NewDb(db, cfg.DBType)

	userRepo := infrarepo.NewUserRepository(sqlxDB)
	repoRepo := infrarepo.NewRepositoryRepository(sqlxDB)
	tokenRepo := infrarepo.NewTokenRepository(sqlxDB)
	membershipRepo := infrarepo.NewMembershipRepository(sqlxDB)

	registerUC := authUC.NewRegisterUserUsecase(userRepo)
	loginUC := authUC.NewLoginUsecase(userRepo, cfg.JWTSecret)
	issuePATUC := authUC.NewIssuePATUsecase(tokenRepo)
	revokePATUC := authUC.NewRevokePATUsecase(tokenRepo)

	oauthCodes := &memoryOAuthCodeStore{data: make(map[string]oauthCodeEntry)}
	oauthAuthorizeUC := authUC.NewOAuthAuthorizeUsecase(&noopOAuthAppRepo{}, oauthCodes)
	oauthTokenUC := authUC.NewOAuthTokenUsecase(oauthCodes, issuePATUC)

	createRepoUC := repoUC.NewCreateRepositoryUsecase(repoRepo)
	getRepoUC := repoUC.NewGetRepositoryUsecase(repoRepo, userRepo, membershipRepo)

	issueRepo := infrarepo.NewIssueRepository(sqlxDB)
	auditLogRepo := infrarepo.NewAuditLogRepository(sqlxDB)
	txManager := &sqlxTxManager{db: sqlxDB}

	createIssueUC := issueUC.NewCreateIssueUsecase(issueRepo, auditLogRepo, txManager)
	listIssuesUC := issueUC.NewListIssuesUsecase(issueRepo)
	createCommentUC := issueUC.NewCreateCommentUsecase(issueRepo, &memoryCommentRepo{}, auditLogRepo)

	prRepo := &stubPullRequestRepo{}
	gitService := &stubGitService{}
	createPRUC := prUC.NewCreatePRUsecase(prRepo, auditLogRepo, gitService, txManager)
	mergePRUC := prUC.NewMergePRUsecase(
		prRepo,
		&stubBranchProtectionRepo{},
		&stubReviewRepo{},
		&stubWorkflowRunRepo{},
		auditLogRepo,
		gitService,
		txManager,
	)

	resolver := &repoResolver{repos: repoRepo, gitDataRoot: cfg.GitDataRoot}
	authMiddleware := appmw.AuthMiddleware(tokenRepo)

	resolveRepo := func(c echo.Context, owner, repoName string) (*entity.Repository, error) {
		return getRepoUC.Execute(c.Request().Context(), repoUC.GetRepositoryInput{
			RequestUserID: appmw.UserUUIDFromContext(c),
			OwnerLogin:    owner,
			Name:          repoName,
		})
	}

	authHandler := handler.NewAuthHandler(registerUC, loginUC)
	repositoryHandler := handler.NewRepositoryHandler(createRepoUC, getRepoUC, repoRepo)
	contentHandler := handler.NewContentHandler(resolver)
	tokenHandler := handler.NewTokenHandler(tokenRepo, issuePATUC, revokePATUC)
	issueHandler := handler.NewIssueHandler(createIssueUC, listIssuesUC, createCommentUC, resolveRepo)
	pullRequestHandler := handler.NewPullRequestHandler(createPRUC, mergePRUC, prRepo, resolveRepo)
	oauthHandler := handler.NewOAuthHandler(oauthAuthorizeUC, oauthTokenUC)
	gitMembership := &gitMembershipAdapter{memberships: membershipRepo}
	gitHTTPHandler := handler.NewGitHTTPHandler(
		cfg.GitDataRoot,
		resolver,
		gitMembership,
		&stubGitBranchProtectionStore{},
		authMiddleware,
	)
	sshKeyRepo := infrarepo.NewSSHKeyRepository(sqlxDB)
	sshKeyHandler := handler.NewSSHKeyHandler(sshKeyRepo)

	api := e.Group("")
	e.POST("/register", authHandler.Register)
	e.POST("/login", authHandler.Login)

	tokens := api.Group("/user/tokens", authMiddleware)
	tokens.GET("", tokenHandler.List)
	tokens.POST("", tokenHandler.Create)
	tokens.DELETE("/:id", tokenHandler.Revoke)

	keys := api.Group("/user/keys", authMiddleware)
	keys.GET("", sshKeyHandler.List)
	keys.POST("", sshKeyHandler.Add)
	keys.DELETE("/:key_id", sshKeyHandler.Delete)

	repositoryHandler.RegisterRoutes(api, authMiddleware)
	contentHandler.RegisterRoutes(api)
	issueHandler.RegisterRoutes(api, authMiddleware)
	pullRequestHandler.RegisterRoutes(api, authMiddleware)
	oauthHandler.RegisterRoutes(api, authMiddleware)
	gitHTTPHandler.RegisterRoutes(e)
}

type repoResolver struct {
	repos       repointer.IRepositoryRepository
	gitDataRoot string
}

func (r *repoResolver) Resolve(ctx context.Context, ownerLogin, repoName string) (*handler.ResolvedGitRepository, error) {
	repository, err := r.repos.GetByOwnerLoginAndName(ctx, ownerLogin, repoName)
	if err != nil || repository == nil {
		return nil, err
	}

	return &handler.ResolvedGitRepository{
		ID:             repository.ID,
		OrganizationID: repository.OrganizationID,
		OwnerID:        uuidToInt64(repository.OwnerID),
		Name:           repository.Name,
		DiskPath:       filepath.Join(r.gitDataRoot, ownerLogin, repoName+".git"),
	}, nil
}

type gitMembershipAdapter struct {
	memberships repointer.IMembershipRepository
}

func (a *gitMembershipAdapter) HasWriteAccess(ctx context.Context, userID int64, organizationID uuid.UUID) (bool, error) {
	return a.memberships.HasWriteAccess(ctx, appmw.Int64ToUUID(userID), organizationID)
}

type txContextKey struct{}

type sqlxTxManager struct {
	db *sqlx.DB
}

func (m *sqlxTxManager) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := context.WithValue(ctx, txContextKey{}, tx)
	if err := fn(txCtx); err != nil {
		return err
	}
	return tx.Commit()
}

type oauthCodeEntry struct {
	value   string
	expires time.Time
}

type memoryOAuthCodeStore struct {
	mu   sync.Mutex
	data map[string]oauthCodeEntry
}

func (s *memoryOAuthCodeStore) Set(_ context.Context, key, value string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = oauthCodeEntry{
		value:   value,
		expires: time.Now().Add(ttl),
	}
	return nil
}

func (s *memoryOAuthCodeStore) GetDel(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.data[key]
	if !ok {
		return "", nil
	}
	delete(s.data, key)
	if time.Now().After(entry.expires) {
		return "", nil
	}
	return entry.value, nil
}

type memoryCommentRepo struct {
	mu       sync.Mutex
	comments []*entity.Comment
}

func (r *memoryCommentRepo) Create(_ context.Context, comment *entity.Comment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.comments = append(r.comments, comment)
	return nil
}

// Stub implementations for features not yet backed by infrastructure repositories.

type noopOAuthAppRepo struct{}

func (noopOAuthAppRepo) GetByClientID(_ context.Context, _ string) (*domain.OAuthApp, error) {
	return nil, errors.New("not found")
}

type stubPullRequestRepo struct{}

func (stubPullRequestRepo) Create(_ context.Context, _ *entity.PullRequest) error {
	return errors.New("pull request repository not configured")
}

func (stubPullRequestRepo) GetByNumber(_ context.Context, _ uuid.UUID, _ int) (*entity.PullRequest, error) {
	return nil, errors.New("not found")
}

func (stubPullRequestRepo) ListByRepo(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*entity.PullRequest, error) {
	return nil, nil
}

func (stubPullRequestRepo) UpdateState(_ context.Context, _ uuid.UUID, _ string) error {
	return errors.New("pull request repository not configured")
}

func (stubPullRequestRepo) SetMerged(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return errors.New("pull request repository not configured")
}

func (stubPullRequestRepo) Update(_ context.Context, _ *entity.PullRequest) error {
	return errors.New("pull request repository not configured")
}

type stubGitService struct{}

func (stubGitService) BranchExists(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

func (stubGitService) ResolveRef(_ context.Context, _ uuid.UUID, _ string) (string, error) {
	return "", errors.New("not found")
}

func (stubGitService) Merge(_ context.Context, _ uuid.UUID, _, _, _ string) error {
	return errors.New("git service not configured")
}

type stubBranchProtectionRepo struct{}

func (stubBranchProtectionRepo) GetForRef(_ context.Context, _ uuid.UUID, _ string) (*entity.BranchProtection, error) {
	return nil, errors.New("not found")
}

type stubReviewRepo struct{}

func (stubReviewRepo) CountSatisfiedReviews(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

type stubWorkflowRunRepo struct{}

func (stubWorkflowRunRepo) ListByHeadSHA(_ context.Context, _ uuid.UUID, _ string) ([]*entity.WorkflowRun, error) {
	return nil, nil
}

type stubGitBranchProtectionStore struct{}

func (stubGitBranchProtectionStore) IsBranchProtected(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

// uuidToInt64 extracts the int64 user ID only when the UUID uses the
// Int64ToUUID encoding (first 8 bytes zero). Full UUIDs return 0 to avoid
// collision-based privilege escalation in permission checks.
func uuidToInt64(id uuid.UUID) int64 {
	for i := 0; i < 8; i++ {
		if id[i] != 0 {
			return 0
		}
	}
	var n int64
	for i := 8; i < 16; i++ {
		n = n<<8 | int64(id[i])
	}
	return n
}
