package main

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/open-git/backend/internal/config"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	appmiddleware "github.com/open-git/backend/internal/middleware"
	"github.com/open-git/backend/internal/infrastructure/database"
	infrarepo "github.com/open-git/backend/internal/infrastructure/repository"
	appssh "github.com/open-git/backend/internal/ssh"
	repo "github.com/open-git/backend/internal/repository"
	authUC "github.com/open-git/backend/internal/usecase/auth"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
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

type gitResolver struct {
	repos   repo.IRepositoryRepository
	gitRoot string
}

func (r *gitResolver) Resolve(ctx context.Context, ownerLogin, repoName string) (*handler.ResolvedGitRepository, error) {
	repoName = strings.TrimSuffix(repoName, ".git")
	repository, err := r.repos.GetByOwnerLoginAndName(ctx, ownerLogin, repoName)
	if err != nil {
		return nil, err
	}
	if repository == nil {
		return nil, domain.ErrNotFound
	}

	return &handler.ResolvedGitRepository{
		ID:             repository.ID,
		OrganizationID: repository.OrganizationID,
		OwnerID:        uuidToInt64(repository.OwnerID),
		Name:           repository.Name,
		DiskPath:       filepath.Join(r.gitRoot, ownerLogin, repoName+".git"),
	}, nil
}

type gitMembershipAdapter struct {
	memberships repo.IMembershipRepository
}

func (a *gitMembershipAdapter) HasWriteAccess(ctx context.Context, userID int64, organizationID uuid.UUID) (bool, error) {
	return a.memberships.HasWriteAccess(ctx, appmiddleware.Int64ToUUID(userID), organizationID)
}

func uuidToInt64(id uuid.UUID) int64 {
	return int64(binary.BigEndian.Uint64(id[8:]))
}

func registerHandlers(e *echo.Echo, cfg config.Config, db *sql.DB) {
	sqlxDB := sqlx.NewDb(db, cfg.DBType)

	userRepo := infrarepo.NewUserRepository(sqlxDB)
	tokenRepo := infrarepo.NewAccessTokenRepository(sqlxDB)
	repoRepo := infrarepo.NewRepositoryRepository(sqlxDB)
	sshKeyRepo := infrarepo.NewSSHKeyRepository(sqlxDB)
	hostKeyRepo := infrarepo.NewHostKeyRepository(sqlxDB)
	membershipRepo := infrarepo.NewMembershipRepository(sqlxDB)
	issueRepo := infrarepo.NewIssueRepository(sqlxDB)

	realAuthMiddleware := appmiddleware.AuthMiddleware(tokenRepo)
	realGitBasicAuth := appmiddleware.GitBasicAuthMiddleware(tokenRepo)
	realOptionalGitAuth := appmiddleware.OptionalGitAuth(tokenRepo)

	gitResolver := &gitResolver{repos: repoRepo, gitRoot: cfg.GitDataRoot}
	membershipAdapter := &gitMembershipAdapter{memberships: membershipRepo}

	registerUC := authUC.NewRegisterUserUsecase(userRepo)
	loginUC := authUC.NewLoginUsecase(userRepo, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(registerUC, loginUC)

	createRepoUC := repoUC.NewCreateRepositoryUsecase(repoRepo)
	getRepoUC := repoUC.NewGetRepositoryUsecase(repoRepo, userRepo, membershipRepo)
	repositoryHandler := handler.NewRepositoryHandler(createRepoUC, getRepoUC, repoRepo)

	contentHandler := handler.NewContentHandler(gitResolver)

	issuePATUC := authUC.NewIssuePATUsecase(tokenRepo)
	revokePATUC := authUC.NewRevokePATUsecase(tokenRepo)
	tokenHandler := handler.NewTokenHandler(tokenRepo, issuePATUC, revokePATUC)

	resolveRepo := func(c echo.Context, owner, name string) (*entity.Repository, error) {
		return getRepoUC.Execute(c.Request().Context(), repoUC.GetRepositoryInput{
			RequestUserID: appmiddleware.UserUUIDFromContext(c),
			OwnerLogin:    owner,
			Name:          name,
		})
	}

	listIssuesUC := issueusecase.NewListIssuesUsecase(issueRepo)
	issueHandler := handler.NewIssueHandler(nil, listIssuesUC, nil, resolveRepo)
	pullRequestHandler := handler.NewPullRequestHandler(nil, nil, nil, resolveRepo)
	oauthHandler := handler.NewOAuthHandler(nil, nil)

	gitHTTPHandler := handler.NewGitHTTPHandler(
		cfg.GitDataRoot,
		gitResolver,
		membershipAdapter,
		nil,
		realGitBasicAuth,
	)
	_ = realOptionalGitAuth
	sshKeyHandler := handler.NewSSHKeyHandler(sshKeyRepo)

	api := e.Group("")
	e.POST("/register", authHandler.Register)
	e.POST("/login", authHandler.Login)

	tokens := api.Group("/user/tokens", realAuthMiddleware)
	tokens.GET("", tokenHandler.List)
	tokens.POST("", tokenHandler.Create)
	tokens.DELETE("/:id", tokenHandler.Revoke)

	keys := api.Group("/user/keys", realAuthMiddleware)
	keys.GET("", sshKeyHandler.List)
	keys.POST("", sshKeyHandler.Add)
	keys.DELETE("/:key_id", sshKeyHandler.Delete)

	repositoryHandler.RegisterRoutes(api, realAuthMiddleware)
	contentHandler.RegisterRoutes(api)
	issueHandler.RegisterRoutes(api, realAuthMiddleware)
	pullRequestHandler.RegisterRoutes(api, realAuthMiddleware)
	oauthHandler.RegisterRoutes(api, realAuthMiddleware)
	gitHTTPHandler.RegisterRoutes(e)

	v3 := e.Group("/api/v3")
	v3.Use(appmiddleware.GitHubCompatHeaders())
	v3.Use(appmiddleware.RateLimitMiddleware(5000))

	repositoryHandler.RegisterRoutes(v3, realAuthMiddleware)
	contentHandler.RegisterRoutes(v3)
	issueHandler.RegisterRoutes(v3, realAuthMiddleware)
	pullRequestHandler.RegisterRoutes(v3, realAuthMiddleware)

	v3Tokens := v3.Group("/user/tokens", realAuthMiddleware)
	v3Tokens.GET("", tokenHandler.List)
	v3Tokens.POST("", tokenHandler.Create)
	v3Tokens.DELETE("/:id", tokenHandler.Revoke)

	v3Keys := v3.Group("/user/keys", realAuthMiddleware)
	v3Keys.GET("", sshKeyHandler.List)
	v3Keys.POST("", sshKeyHandler.Add)
	v3Keys.DELETE("/:key_id", sshKeyHandler.Delete)

	if cfg.SSHEnabled {
		go func() {
			ctx := context.Background()
			signer, err := appssh.LoadOrGenerateHostKey(ctx, hostKeyRepo, appssh.AlgorithmEd25519)
			if err != nil {
				log.Fatalf("load ssh host key: %v", err)
			}

			sshServer := appssh.NewSSHServer(cfg.GitDataRoot, gitResolver, sshKeyRepo, membershipAdapter, nil)
			log.Printf("ssh server listening on :%s", cfg.SSHPort)
			if err := sshServer.Start(":"+cfg.SSHPort, signer); err != nil {
				log.Fatalf("start ssh server: %v", err)
			}
		}()
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
