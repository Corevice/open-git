package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
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

	gossh "github.com/gliderlabs/ssh"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/open-git/backend/internal/config"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/handler"
	appmiddleware "github.com/open-git/backend/internal/middleware"
	infraDB "github.com/open-git/backend/internal/infrastructure/database"
	sshinfra "github.com/open-git/backend/internal/infrastructure/ssh"
	infrarepo "github.com/open-git/backend/internal/infrastructure/repository"
	authUC "github.com/open-git/backend/internal/usecase/auth"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
	orgUC "github.com/open-git/backend/internal/usecase/org"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
	userUC "github.com/open-git/backend/internal/usecase/user"
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

	db, err := infraDB.Connect(cfg)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	if err := infraDB.Ping(context.Background(), db); err != nil {
		log.Fatalf("ping database: %v", err)
	}
	log.Printf("database connected (%s): %s", cfg.DBType, infraDB.MaskDSN(cfg.DBDSN))

	if cfg.DBAutoMigrate {
		if err := infraDB.RunMigrations(db, cfg.DBType, "./migrations"); err != nil {
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

	sshServer, err := registerHandlers(e, cfg, db)
	if err != nil {
		log.Fatalf("register handlers: %v", err)
	}

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
	if sshServer != nil {
		if err := sshServer.Close(); err != nil {
			log.Printf("shutdown ssh server: %v", err)
		}
	}
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
	if strings.Contains(ownerLogin, "..") || strings.Contains(repoName, "..") {
		return nil, domain.ErrNotFound
	}
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
		OwnerID:        appmiddleware.UUIDToInt64(repository.OwnerID),
		Name:           repository.Name,
		Visibility:     repository.Visibility,
		DiskPath:       filepath.Join(r.gitRoot, ownerLogin, repoName+".git"),
	}, nil
}

type gitMembershipAdapter struct {
	memberships membershipRoleLookup
}

type membershipRoleLookup interface {
	GetRole(ctx context.Context, orgID, userID uuid.UUID) (string, error)
}

func (a *gitMembershipAdapter) HasReadAccess(ctx context.Context, userID int64, organizationID uuid.UUID) (bool, error) {
	_, err := a.memberships.GetRole(ctx, organizationID, appmiddleware.Int64ToUUID(userID))
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (a *gitMembershipAdapter) HasWriteAccess(ctx context.Context, userID int64, organizationID uuid.UUID) (bool, error) {
	role, err := a.memberships.GetRole(ctx, organizationID, appmiddleware.Int64ToUUID(userID))
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return role == entity.RoleOwner || role == entity.RoleAdmin, nil
}

type legacyMembershipRepoAdapter struct {
	inner membershipRoleLookup
}

func (a *legacyMembershipRepoAdapter) HasReadAccess(ctx context.Context, userID, organizationID uuid.UUID) (bool, error) {
	_, err := a.inner.GetRole(ctx, organizationID, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (a *legacyMembershipRepoAdapter) HasWriteAccess(ctx context.Context, userID, organizationID uuid.UUID) (bool, error) {
	role, err := a.inner.GetRole(ctx, organizationID, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return role == entity.RoleOwner || role == entity.RoleAdmin, nil
}

type legacyUserRepoAdapter struct {
	users domainrepo.IUserRepository
}

func (a *legacyUserRepoAdapter) Create(ctx context.Context, user *domain.User) error {
	entityUser := domainUserToEntity(user)
	if err := a.users.Create(ctx, entityUser); err != nil {
		return err
	}
	user.ID = uuidToInt64(entityUser.ID)
	return nil
}

func (a *legacyUserRepoAdapter) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	entityUser, err := a.users.GetByID(ctx, appmiddleware.Int64ToUUID(id))
	if err != nil {
		return nil, err
	}
	return entityUserToDomain(entityUser), nil
}

func (a *legacyUserRepoAdapter) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	entityUser, err := a.users.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	return entityUserToDomain(entityUser), nil
}

func (a *legacyUserRepoAdapter) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	entityUser, err := a.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return entityUserToDomain(entityUser), nil
}

type gitSSHInfraResolver struct {
	resolver handler.GitRepositoryResolver
}

func (r *gitSSHInfraResolver) Resolve(ctx context.Context, ownerLogin, repoName string) (string, uuid.UUID, error) {
	resolved, err := r.resolver.Resolve(ctx, ownerLogin, repoName)
	if err != nil {
		return "", uuid.Nil, err
	}
	return resolved.DiskPath, appmiddleware.Int64ToUUID(resolved.OwnerID), nil
}

func entityUserToDomain(user *entity.User) *domain.User {
	if user == nil {
		return nil
	}
	return &domain.User{
		ID:           appmiddleware.UUIDToInt64(user.ID),
		Login:        user.Login,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
	}
}

func domainUserToEntity(user *domain.User) *entity.User {
	entityUser := &entity.User{
		Login:        user.Login,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
	}
	if user.ID != 0 {
		entityUser.ID = appmiddleware.Int64ToUUID(user.ID)
	}
	return entityUser
}

func loadOrGenerateHostKey(path string) (gossh.Signer, error) {
	if data, err := os.ReadFile(path); err == nil {
		signer, err := gossh.ParsePrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("parse host key: %w", err)
		}
		return signer, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("generate host key: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create host key directory: %w", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write host key: %w", err)
	}

	return gossh.NewSignerFromKey(privateKey)
}

func registerHandlers(e *echo.Echo, cfg config.Config, db *sql.DB) (*sshinfra.SSHServer, error) {
	sqlxDB := sqlx.NewDb(db, cfg.DBType)

	tokenRepo := infrarepo.NewAccessTokenRepository(sqlxDB)
	entityUserRepo := infrarepo.NewUserRepository(sqlxDB)
	userRepo := &legacyUserRepoAdapter{users: entityUserRepo}
	repoRepo := infrarepo.NewRepositoryRepository(sqlxDB)
	membershipRepo := infrarepo.NewMembershipRepository(sqlxDB)
	legacyMembershipRepo := &legacyMembershipRepoAdapter{inner: membershipRepo}
	orgRepo := infraDB.NewOrganizationRepository(db)
	auditLogRepo := infraDB.NewAuditLogRepository(db)
	sshKeyRepo := infrarepo.NewSSHKeyRepository(sqlxDB)
	issueRepo := infrarepo.NewIssueRepository(sqlxDB)

	authMiddleware := appmiddleware.AuthMiddleware(tokenRepo)
	realGitBasicAuth := appmiddleware.GitBasicAuthMiddleware(tokenRepo)
	realOptionalGitAuth := appmiddleware.OptionalGitAuth(tokenRepo)

	repoGitResolver := &gitResolver{repos: repoRepo, gitRoot: cfg.GitDataRoot}
	membershipAdapter := &gitMembershipAdapter{memberships: membershipRepo}

	registerUC := authUC.NewRegisterUserUsecase(userRepo)
	loginUC := authUC.NewLoginUsecase(userRepo, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(registerUC, loginUC)

	createRepoUC := repoUC.NewCreateRepositoryUsecase(
		repoRepo,
		repoUC.WithGitDataRoot(cfg.GitDataRoot),
		repoUC.WithOwnerLoginResolver(entityUserRepo),
	)
	getRepoUC := repoUC.NewGetRepositoryUsecase(repoRepo, userRepo, legacyMembershipRepo)
	listReposUC := repoUC.NewListRepositoriesUsecase(repoRepo, legacyMembershipRepo, userRepo)
	repositoryHandler := handler.NewRepositoryHandler(createRepoUC, getRepoUC, listReposUC, repoRepo, orgRepo, auditLogRepo)

	getCurrentUserUC := userUC.NewGetCurrentUserUsecase(userRepo)
	getUserByLoginUC := userUC.NewGetUserByLoginUsecase(userRepo)
	updateUserUC := userUC.NewUpdateUserUsecase(entityUserRepo)
	userHandler := handler.NewUserHandler(getCurrentUserUC, getUserByLoginUC, updateUserUC)

	getOrgUC := orgUC.NewGetOrgUsecase(orgRepo)
	listUserOrgsUC := orgUC.NewListUserOrgsUsecase(orgRepo)
	entityOrgRepo := infrarepo.NewOrganizationRepository(sqlxDB)
	createOrgUC := orgUC.NewCreateOrgUsecase(entityOrgRepo, membershipRepo)
	updateOrgUC := orgUC.NewUpdateOrgUsecase(entityOrgRepo, membershipRepo)
	deleteOrgUC := orgUC.NewDeleteOrgUsecase(entityOrgRepo, membershipRepo, auditLogRepo)
	inviteMemberUC := orgUC.NewInviteMemberUsecase(membershipRepo)
	removeMemberUC := orgUC.NewRemoveMemberUsecase(membershipRepo, auditLogRepo)
	orgHandler := handler.NewOrgHandler(
		getOrgUC,
		listUserOrgsUC,
		createOrgUC,
		updateOrgUC,
		deleteOrgUC,
		inviteMemberUC,
		removeMemberUC,
		membershipRepo,
		entityUserRepo,
	)

	contentHandler := handler.NewContentHandler(repoGitResolver)

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
		repoGitResolver,
		membershipAdapter,
		nil,
		realGitBasicAuth,
	)
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
	e.GET("/:owner/:repo.git/info/refs", gitHTTPHandler.InfoRefs, realOptionalGitAuth)
	e.POST("/:owner/:repo.git/git-upload-pack", gitHTTPHandler.UploadPack, realOptionalGitAuth)
	e.POST("/:owner/:repo.git/git-receive-pack", gitHTTPHandler.ReceivePack, realGitBasicAuth)

	v3 := e.Group("/api/v3")
	v3.Use(appmiddleware.GitHubCompatHeaders())
	v3.Use(appmiddleware.RateLimitMiddleware(5000))
	v3.Use(appmiddleware.GitHubCommonHeadersMiddleware())

	userHandler.RegisterRoutes(v3, authMiddleware)
	orgHandler.RegisterRoutes(v3, authMiddleware)
	repositoryHandler.RegisterRoutes(v3, authMiddleware)
	contentHandler.RegisterRoutes(v3)
	issueHandler.RegisterRoutes(v3, authMiddleware)
	pullRequestHandler.RegisterRoutes(v3, authMiddleware)

	v3Tokens := v3.Group("/user/tokens", authMiddleware)
	v3Tokens.GET("", tokenHandler.List)
	v3Tokens.POST("", tokenHandler.Create)
	v3Tokens.DELETE("/:id", tokenHandler.Revoke)

	v3Keys := v3.Group("/user/keys", authMiddleware)
	v3Keys.GET("", sshKeyHandler.List)
	v3Keys.POST("", sshKeyHandler.Add)
	v3Keys.DELETE("/:key_id", sshKeyHandler.Delete)

	var sshServer *sshinfra.SSHServer
	if cfg.SSHEnabled {
		hostKey, err := loadOrGenerateHostKey(cfg.SSHHostKeyPath)
		if err != nil {
			return nil, fmt.Errorf("load ssh host key: %w", err)
		}

		sshListenAddr := cfg.SSHPort
		if !strings.HasPrefix(sshListenAddr, ":") {
			sshListenAddr = ":" + sshListenAddr
		}

		sshServer = sshinfra.NewSSHServer(
			cfg.GitDataRoot,
			sshKeyRepo,
			&gitSSHInfraResolver{resolver: repoGitResolver},
			hostKey,
		)
		go func() {
			log.Printf("ssh server listening on %s", sshListenAddr)
			if err := sshServer.Start(sshListenAddr); err != nil && !errors.Is(err, gossh.ErrServerClosed) {
				log.Printf("ssh server stopped: %v", err)
			}
		}()
	}
	return sshServer, nil
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
