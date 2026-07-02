package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	gossh "github.com/gliderlabs/ssh"
	"github.com/google/uuid"
	cryptossh "golang.org/x/crypto/ssh"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/compat"
	"github.com/open-git/backend/internal/config"
	"github.com/open-git/backend/graph"
	obs "github.com/open-git/backend/observability"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/infrastructure/crypto"
	infraDB "github.com/open-git/backend/internal/infrastructure/database"
	"github.com/open-git/backend/internal/infrastructure/ci"
	infragit "github.com/open-git/backend/internal/infrastructure/git"
	"github.com/open-git/backend/internal/infrastructure/kvstore"
	"github.com/open-git/backend/internal/infrastructure/queue"
	infrarepo "github.com/open-git/backend/internal/infrastructure/repository"
	sshinfra "github.com/open-git/backend/internal/infrastructure/ssh"
	"github.com/open-git/backend/internal/logger"
	"github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
	actionsusecase "github.com/open-git/backend/internal/usecase/actions"
	authUC "github.com/open-git/backend/internal/usecase/auth"
	compatusecase "github.com/open-git/backend/internal/usecase/compat"
	docsuc "github.com/open-git/backend/internal/usecase/docs"
	importUC "github.com/open-git/backend/internal/usecase/import"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
	labelusecase "github.com/open-git/backend/internal/usecase/label"
	mcpusecase "github.com/open-git/backend/internal/usecase/mcp"
	milestoneusecase "github.com/open-git/backend/internal/usecase/milestone"
	orgUC "github.com/open-git/backend/internal/usecase/org"
	prusecase "github.com/open-git/backend/internal/usecase/pr"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
	securityUC "github.com/open-git/backend/internal/usecase/security"
	userUC "github.com/open-git/backend/internal/usecase/user"
	userpreferencesUC "github.com/open-git/backend/internal/usecase/user_preferences"
	webhookusecase "github.com/open-git/backend/internal/usecase/webhook"
	workflowusecase "github.com/open-git/backend/internal/usecase/workflow"
	"github.com/open-git/backend/internal/worker"
	artifactusecase "github.com/open-git/backend/internal/usecase/artifact"
	secretusecase "github.com/open-git/backend/internal/usecase/secret"
)

var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func validateRequiredEnv(vars []string) error {
	for _, v := range vars {
		if os.Getenv(v) == "" {
			return fmt.Errorf("missing required environment variable: %s", v)
		}
	}
	return nil
}

func main() {
	if err := validateRequiredEnv([]string{"JWT_SECRET"}); err != nil {
		log.Fatalf("%v", err)
	}

	cfg := config.Load()
	// DB_DSN is only required for postgres; sqlite falls back to a default file
	// path (see config.Validate and the database package), so requiring it
	// unconditionally here made the documented sqlite default impossible to run.
	if cfg.DBType == "postgres" && os.Getenv("DB_DSN") == "" {
		log.Fatalf("missing required environment variable: DB_DSN")
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}
	middleware.InitLogging(os.Getenv("LOG_LEVEL"))

	db, err := infraDB.Connect(cfg)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	if err := infraDB.Ping(context.Background(), db); err != nil {
		log.Fatalf("ping database: %v", err)
	}
	middleware.Log().Info("database connected", "db_type", cfg.DBType, "dsn", infraDB.MaskDSN(cfg.DBDSN))

	if cfg.DBAutoMigrate {
		if err := infraDB.RunMigrations(db, cfg.DBType, "./migrations"); err != nil {
			log.Fatalf("run migrations: %v", err)
		}
	}

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = newHTTPErrorHandler()

	if err := middleware.SetupProxyTrust(e, cfg.TrustedProxyCIDRs); err != nil {
		log.Printf("warning: setup proxy trust: %v", err)
	}

	e.Use(echoMiddleware.RequestID())
	e.Use(middleware.SecurityHeadersMiddleware())
	e.Use(middleware.RequestLogger())
	e.Use(middleware.StructuredRecover())
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins: corsAllowedOrigins(),
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, echo.HeaderXRequestID},
	}))
	e.Use(echoMiddleware.RateLimiter(echoMiddleware.NewRateLimiterMemoryStore(20)))
	e.Use(echoMiddleware.TimeoutWithConfig(echoMiddleware.TimeoutConfig{Timeout: 30 * time.Second}))
	e.Use(requestContextMiddleware())
	if cfg.MetricsEnabled {
		e.Use(obs.EchoPrometheusMiddleware)
		obs.RegisterMetricsRoute(e, cfg.MetricsPath, cfg.MetricsAuthToken)
	}

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
			middleware.Log().Info("shutdown ssh server", "error", err)
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
		OwnerID:        middleware.UUIDToInt64(repository.OwnerID),
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
	_, err := a.memberships.GetRole(ctx, organizationID, middleware.Int64ToUUID(userID))
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (a *gitMembershipAdapter) HasWriteAccess(ctx context.Context, userID int64, organizationID uuid.UUID) (bool, error) {
	role, err := a.memberships.GetRole(ctx, organizationID, middleware.Int64ToUUID(userID))
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
	user.ID = middleware.UUIDToInt64(entityUser.ID)
	return nil
}

func (a *legacyUserRepoAdapter) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	entityUser, err := a.users.GetByID(ctx, middleware.Int64ToUUID(id))
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
	return resolved.DiskPath, middleware.Int64ToUUID(resolved.OwnerID), nil
}

// sshRepoAuthorizer enforces the same read/write permissions for git-over-SSH
// as the HTTP git handler (owner, org membership, and collaborator).
type sshCollaboratorLookup interface {
	GetPermission(ctx context.Context, repoID, userID uuid.UUID) (string, error)
}

type sshRepoAuthorizer struct {
	resolver      handler.GitRepositoryResolver
	memberships   *gitMembershipAdapter
	collaborators sshCollaboratorLookup
}

func (a *sshRepoAuthorizer) CanRead(ctx context.Context, userID uuid.UUID, ownerLogin, repoName string) (bool, error) {
	repo, err := a.resolver.Resolve(ctx, ownerLogin, repoName)
	if err != nil {
		return false, err
	}
	if repo.Visibility != entity.VisibilityPrivate {
		return true, nil
	}
	uid := middleware.UUIDToInt64(userID)
	if repo.OwnerID != 0 && repo.OwnerID == uid {
		return true, nil
	}
	if a.memberships != nil {
		if ok, err := a.memberships.HasReadAccess(ctx, uid, repo.OrganizationID); err == nil && ok {
			return true, nil
		}
	}
	if a.collaborators != nil {
		if perm, err := a.collaborators.GetPermission(ctx, repo.ID, userID); err == nil && perm != "" {
			return true, nil
		}
	}
	return false, nil
}

func (a *sshRepoAuthorizer) CanWrite(ctx context.Context, userID uuid.UUID, ownerLogin, repoName string) (bool, error) {
	repo, err := a.resolver.Resolve(ctx, ownerLogin, repoName)
	if err != nil {
		return false, err
	}
	uid := middleware.UUIDToInt64(userID)
	if repo.OwnerID != 0 && repo.OwnerID == uid {
		return true, nil
	}
	if a.memberships != nil {
		if ok, err := a.memberships.HasWriteAccess(ctx, uid, repo.OrganizationID); err == nil && ok {
			return true, nil
		}
	}
	if a.collaborators != nil {
		if perm, err := a.collaborators.GetPermission(ctx, repo.ID, userID); err == nil &&
			(perm == entity.CollaboratorPermWrite || perm == entity.CollaboratorPermAdmin) {
			return true, nil
		}
	}
	return false, nil
}

func entityUserToDomain(user *entity.User) *domain.User {
	if user == nil {
		return nil
	}
	return &domain.User{
		ID:           middleware.UUIDToInt64(user.ID),
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
		entityUser.ID = middleware.Int64ToUUID(user.ID)
	}
	return entityUser
}

func loadOrGenerateHostKey(path string) (gossh.Signer, error) {
	if data, err := os.ReadFile(path); err == nil {
		signer, err := cryptossh.ParsePrivateKey(data)
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

	return cryptossh.NewSignerFromKey(privateKey)
}

func registerHandlers(e *echo.Echo, cfg config.Config, db *sql.DB) (*sshinfra.SSHServer, error) {
	sqlxDB := sqlx.NewDb(db, cfg.DBType)

	tokenRepo := infrarepo.NewAccessTokenRepository(sqlxDB)
	entityUserRepo := infrarepo.NewUserRepository(sqlxDB)
	userRepo := &legacyUserRepoAdapter{users: entityUserRepo}
	repoRepo := infrarepo.NewRepositoryRepository(sqlxDB)
	collaboratorRepo := infrarepo.NewRepositoryCollaboratorRepository(sqlxDB)
	membershipRepo := infrarepo.NewMembershipRepository(sqlxDB)
	// Shared repository/organization authorization used by all handlers to
	// enforce read/write/admin on the target (not just authentication).
	repoAccess := handler.NewRepoAccess(membershipRepo, collaboratorRepo)
	legacyMembershipRepo := &legacyMembershipRepoAdapter{inner: membershipRepo}
	orgRepo := infraDB.NewOrganizationRepository(db)
	auditLogRepo := infraDB.NewAuditLogRepository(db)
	sshKeyRepo := infrarepo.NewSSHKeyRepository(sqlxDB)
	issueRepo := infrarepo.NewIssueRepository(sqlxDB)
	issueAuditRepo := infrarepo.NewAuditLogRepository(sqlxDB)
	commentRepo := infrarepo.NewCommentRepository(sqlxDB)
	labelRepo := infrarepo.NewLabelRepository(sqlxDB)
	milestoneRepo := infrarepo.NewMilestoneRepository(sqlxDB)
	txManager := infraDB.NewDomainTxManager(sqlxDB)

	authMiddleware := middleware.AuthMiddleware(tokenRepo)
	realGitBasicAuth := middleware.GitBasicAuthMiddleware(tokenRepo)
	realOptionalGitAuth := middleware.OptionalGitAuth(tokenRepo)

	repoGitResolver := &gitResolver{repos: repoRepo, gitRoot: cfg.GitDataRoot}
	membershipAdapter := &gitMembershipAdapter{memberships: membershipRepo}

	registerUC := authUC.NewRegisterUserUsecase(userRepo, &personalOrgCreator{orgs: infrarepo.NewOrganizationRepository(sqlxDB)})
	loginUC := authUC.NewLoginUsecase(userRepo, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(registerUC, loginUC)

	createRepoUC := repoUC.NewCreateRepositoryUsecase(
		repoRepo,
		repoUC.WithGitDataRoot(cfg.GitDataRoot),
		repoUC.WithOwnerLoginResolver(entityUserRepo),
	)
	getRepoUC := repoUC.NewGetRepositoryUsecase(repoRepo, userRepo, legacyMembershipRepo)
	listReposUC := repoUC.NewListRepositoriesUsecase(repoRepo, legacyMembershipRepo, userRepo)
	auditListRepo := infrarepo.NewAuditLogRepository(sqlxDB)
	listAuditLogsUC := repoUC.NewListAuditLogsUsecase(auditListRepo)
	searchAuditLogsUC := securityUC.NewSearchAuditLogsUsecase(auditListRepo)
	repositoryHandler := handler.NewRepositoryHandler(createRepoUC, getRepoUC, listReposUC, repoRepo, orgRepo, userRepo, auditLogRepo, listAuditLogsUC)

	getCurrentUserUC := userUC.NewGetCurrentUserUsecase(userRepo)
	getUserByLoginUC := userUC.NewGetUserByLoginUsecase(userRepo)
	updateUserUC := userUC.NewUpdateUserUsecase(entityUserRepo)
	userHandler := handler.NewUserHandler(getCurrentUserUC, getUserByLoginUC, updateUserUC)

	userPrefsRepo := infrarepo.NewUserPreferencesRepository(sqlxDB)
	getUserPrefsUC := userpreferencesUC.NewGetUserPreferencesUsecase(userPrefsRepo)
	updateUserPrefsUC := userpreferencesUC.NewUpdateUserPreferencesUsecase(userPrefsRepo)
	userPreferencesHandler := handler.NewUserPreferencesHandler(getUserPrefsUC, updateUserPrefsUC)

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
	branchHandler := handler.NewBranchHandler(repoGitResolver, repoRepo, membershipAdapter)

	issuePATUC := authUC.NewIssuePATUsecase(tokenRepo)
	revokePATUC := authUC.NewRevokePATUsecase(tokenRepo)
	tokenAuditRepo := infrarepo.NewAuditLogRepository(sqlxDB)
	tokenHandler := handler.NewTokenHandler(tokenRepo, issuePATUC, revokePATUC, tokenAuditRepo, entityUserRepo)

	resolveRepo := func(c echo.Context, owner, name string) (*entity.Repository, error) {
		return getRepoUC.Execute(c.Request().Context(), repoUC.GetRepositoryInput{
			RequestUserID: middleware.UserUUIDFromContext(c),
			OwnerLogin:    owner,
			Name:          name,
		})
	}

	listLabelsUC := labelusecase.NewListLabelsUsecase(labelRepo)
	createLabelUC := labelusecase.NewCreateLabelUsecase(labelRepo)
	updateLabelUC := labelusecase.NewUpdateLabelUsecase(labelRepo)
	deleteLabelUC := labelusecase.NewDeleteLabelUsecase(labelRepo, issueAuditRepo)
	addIssueLabelsUC := labelusecase.NewAddIssueLabelsUsecase(labelRepo)
	removeIssueLabelUC := labelusecase.NewRemoveIssueLabelUsecase(labelRepo)
	labelHandler := handler.NewLabelHandler(
		createLabelUC,
		listLabelsUC,
		updateLabelUC,
		deleteLabelUC,
		addIssueLabelsUC,
		removeIssueLabelUC,
		resolveRepo,
	)

	listMilestonesUC := milestoneusecase.NewListMilestonesUsecase(milestoneRepo)
	createMilestoneUC := milestoneusecase.NewCreateMilestoneUsecase(milestoneRepo, issueAuditRepo)
	updateMilestoneUC := milestoneusecase.NewUpdateMilestoneUsecase(milestoneRepo)
	deleteMilestoneUC := milestoneusecase.NewDeleteMilestoneUsecase(milestoneRepo, issueAuditRepo)
	milestoneHandler := handler.NewMilestoneHandler(
		listMilestonesUC,
		createMilestoneUC,
		updateMilestoneUC,
		deleteMilestoneUC,
		resolveRepo,
	)

	createIssueUC := issueusecase.NewCreateIssueUsecase(issueRepo, issueAuditRepo, txManager)
	updateIssueUC := issueusecase.NewUpdateIssueUsecase(issueRepo, labelRepo, milestoneRepo, issueAuditRepo)
	createCommentUC := issueusecase.NewCreateCommentUsecase(issueRepo, commentRepo, issueAuditRepo)
	listIssuesUC := issueusecase.NewListIssuesUsecase(issueRepo)
	getIssueUC := issueusecase.NewGetIssueUsecase(issueRepo)
	issueHandler := handler.NewIssueHandler(
		createIssueUC,
		listIssuesUC,
		getIssueUC,
		updateIssueUC,
		createCommentUC,
		resolveRepo,
	)

	gitSvc := infragit.NewGitServiceAdapter()
	prRepo := infrarepo.NewPullRequestRepository(sqlxDB)
	prReviewRepo := infrarepo.NewReviewRepository(sqlxDB)
	prReviewCommentRepo := infrarepo.NewReviewCommentRepository(sqlxDB)
	bpRepo := infrarepo.NewBranchProtectionRepository(sqlxDB)
	bpAuditRepo := infrarepo.NewAuditLogRepository(sqlxDB)
	wfRepo := infrarepo.NewWorkflowRunRepository(sqlxDB)
	prAuditRepo := infrarepo.NewAuditLogRepository(sqlxDB)
	prTxManager := infrarepo.NewTransactionManager(sqlxDB)
	mergePRUC := prusecase.NewMergePRUsecase(prRepo, bpRepo, prReviewRepo, wfRepo, prAuditRepo, gitSvc, prTxManager, membershipRepo)
	createReviewUC := prusecase.NewCreateReviewUsecase(prRepo, prReviewRepo, prAuditRepo, membershipRepo)
	listReviewsUC := prusecase.NewListReviewsUsecase(prRepo, prReviewRepo)

	bpReadRepo := &branchProtectionReadRepo{db: sqlxDB}
	bpWriteRepo := &branchProtectionWriteRepo{db: sqlxDB}
	bpUpsertUC := repoUC.NewUpsertBranchProtectionUsecase(bpWriteRepo, auditLogRepo)
	bpDeleteUC := repoUC.NewDeleteBranchProtectionUsecase(bpWriteRepo, auditLogRepo)
	checkRepoAdmin := func(c echo.Context, repo *entity.Repository) error {
		userID := middleware.UserUUIDFromContext(c)
		if repo.OwnerID == userID {
			return nil
		}
		role, err := membershipRepo.GetRole(c.Request().Context(), repo.OrganizationID, userID)
		if err != nil || (role != "admin" && role != "owner") {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
		}
		return nil
	}
	branchProtectionHandler := handler.NewBranchProtectionHandler(
		bpReadRepo,
		bpUpsertUC,
		bpDeleteUC,
		resolveRepo,
		checkRepoAdmin,
		bpAuditRepo,
	)

	webhookEncryptor := crypto.NewSecretEncryptorFromEnv()
	webhookRepo := infrarepo.NewWebhookRepository(sqlxDB, webhookEncryptor)
	webhookAuditRepo := infrarepo.NewAuditLogRepository(sqlxDB)
	createWebhookUC := webhookusecase.NewCreateWebhookUsecase(webhookRepo, webhookAuditRepo, webhookEncryptor)
	listWebhooksUC := webhookusecase.NewListWebhooksUsecase(webhookRepo)
	getWebhookUC := webhookusecase.NewGetWebhookUsecase(webhookRepo)
	updateWebhookUC := webhookusecase.NewUpdateWebhookUsecase(webhookRepo, webhookAuditRepo, webhookEncryptor)
	deleteWebhookUC := webhookusecase.NewDeleteWebhookUsecase(webhookRepo, webhookAuditRepo)
	webhookHandler := handler.NewWebhookHandler(
		createWebhookUC,
		listWebhooksUC,
		getWebhookUC,
		updateWebhookUC,
		deleteWebhookUC,
		resolveRepo,
	)

	actionSecretEnc := newActionSecretEncryptorFromEnv()
	actionSecretRepo := infrarepo.NewActionSecretRepository(sqlxDB, actionSecretEnc.SecretEncryptor)
	actionSecretAuditRepo := infrarepo.NewAuditLogRepository(sqlxDB)
	listRepoSecretsUC := secretusecase.NewListRepoSecretsUsecase(actionSecretRepo)
	listOrgSecretsUC := secretusecase.NewListOrgSecretsUsecase(actionSecretRepo)
	getActionSecretUC := secretusecase.NewGetActionSecretUsecase(actionSecretRepo)
	upsertActionSecretUC := secretusecase.NewUpsertActionSecretUsecase(actionSecretRepo, actionSecretAuditRepo, actionSecretEnc)
	deleteActionSecretUC := secretusecase.NewDeleteActionSecretUsecase(actionSecretRepo, actionSecretAuditRepo)
	getActionSecretPublicKeyUC := secretusecase.NewGetPublicKeyUsecase(actionSecretEnc)
	resolveOrg := func(c echo.Context, orgLogin string) (uuid.UUID, error) {
		org, err := entityOrgRepo.GetByLogin(c.Request().Context(), orgLogin)
		if err != nil {
			return uuid.Nil, err
		}
		if org == nil {
			return uuid.Nil, apperror.ErrNotFound
		}
		return org.ID, nil
	}
	secretHandler := handler.NewSecretHandler(
		listRepoSecretsUC,
		listOrgSecretsUC,
		getActionSecretUC,
		upsertActionSecretUC,
		deleteActionSecretUC,
		getActionSecretPublicKeyUC,
		actionSecretRepo,
		repoRepo,
		actionSecretEnc,
		resolveRepo,
		resolveOrg,
	)

	compatRepo := infrarepo.NewCompatRepository(sqlxDB)
	compatRunner := &compat.Runner{}
	getReportUC := compatusecase.NewGetReportUsecase(compatRepo)
	triggerRunUC := compatusecase.NewTriggerRunUsecase(compatRepo, compatRunner)
	compatHandler := handler.NewCompatHandler(getReportUC, triggerRunUC, compatRepo)

	minioBucket := getenv("MINIO_BUCKET", "artifacts")
	var artifactStorage artifactusecase.ArtifactStorage
	if cfg.MinioEndpoint != "" {
		minioStorage, err := newArtifactMinioStorage(
			cfg.MinioEndpoint,
			os.Getenv("MINIO_ACCESS_KEY"),
			os.Getenv("MINIO_SECRET_KEY"),
			getenvBool("MINIO_USE_TLS", false),
		)
		if err != nil {
			return nil, fmt.Errorf("init artifact storage: %w", err)
		}
		if err := minioStorage.EnsureBucket(context.Background(), minioBucket); err != nil {
			return nil, fmt.Errorf("ensure artifact bucket: %w", err)
		}
		artifactStorage = minioStorage
	}

	artifactRepo := infrarepo.NewArtifactRepository(sqlxDB)
	createArtifactUC := artifactusecase.NewCreateArtifactUsecase(artifactRepo, artifactStorage, minioBucket)
	completeArtifactUC := artifactusecase.NewCompleteArtifactUsecase(artifactRepo)
	listArtifactsUC := artifactusecase.NewListArtifactsUsecase(artifactRepo)
	getArtifactDownloadURLUC := artifactusecase.NewGetArtifactDownloadURLUsecase(artifactRepo, artifactStorage, minioBucket)
	deleteArtifactUC := artifactusecase.NewDeleteArtifactUsecase(artifactRepo, artifactStorage, minioBucket)
	artifactHandler := handler.NewArtifactHandler(
		createArtifactUC,
		completeArtifactUC,
		listArtifactsUC,
		getArtifactDownloadURLUC,
		deleteArtifactUC,
		resolveRepo,
	)

	mcpVerificationRepo := infrarepo.NewSQLxMCPVerificationRepository(sqlxDB)
	mcpAuditRepo := infrarepo.NewAuditLogRepository(sqlxDB)
	getLatestVerificationUC := mcpusecase.NewGetLatestVerificationUsecase(mcpVerificationRepo)
	listHistoryVerificationUC := mcpusecase.NewListVerificationHistoryUsecase(mcpVerificationRepo)
	getJobStatusUC := mcpusecase.NewGetJobStatusUsecase(mcpVerificationRepo)
	deleteVerificationUC := mcpusecase.NewDeleteVerificationUsecase(mcpVerificationRepo, mcpAuditRepo)

	var runVerificationUC *mcpusecase.RunVerificationUsecase
	var asynqClient *asynq.Client
	if cfg.RedisAddr != "" {
		asynqClient = queue.NewAsynqClient(cfg.RedisAddr)
		runVerificationUC = mcpusecase.NewRunVerificationUsecase(mcpVerificationRepo, mcpAuditRepo, asynqClient)

		asynqServer := queue.NewAsynqServer(cfg.RedisAddr, 10)
		mux := asynq.NewServeMux()
		mcpWorker := worker.NewMCPVerificationWorker(mcpVerificationRepo, cfg.WebBaseURL)
		mux.HandleFunc(queue.TypeMCPVerification, mcpWorker.HandleMCPVerification)
		artifactCleanupWorker := worker.NewArtifactCleanupWorker(artifactRepo, artifactStorage, minioBucket)
		mux.HandleFunc(queue.TypeArtifactCleanup, artifactCleanupWorker.HandleCleanup)
		scheduler := asynq.NewScheduler(asynq.RedisClientOpt{Addr: cfg.RedisAddr}, nil)
		if _, err := scheduler.Register("@hourly", asynq.NewTask(queue.TypeArtifactCleanup, nil)); err != nil {
			return nil, fmt.Errorf("register artifact cleanup scheduler: %w", err)
		}
		go func() {
			if err := scheduler.Run(); err != nil {
				middleware.Log().Info("asynq scheduler stopped", "error", err)
			}
		}()
		prMergeableWorker := queue.NewPRMergeableWorker(prRepo, repoRepo, gitSvc)
		mux.HandleFunc(queue.TypePRMergeableCheck, prMergeableWorker.HandlePRMergeableCheck)
		go func() {
			middleware.Log().Info("asynq server starting")
			if err := asynqServer.Run(mux); err != nil {
				middleware.Log().Info("asynq server stopped", "error", err)
			}
		}()
	} else {
		runVerificationUC = mcpusecase.NewRunVerificationUsecaseWithDeps(
			mcpVerificationRepo,
			mcpAuditRepo,
			noopMCPEnqueuer{},
		)
	}

	var prMergeableEnqueuer prusecase.PRMergeableEnqueuer = prusecase.NoopPRMergeableEnqueuer{}
	if asynqClient != nil {
		prMergeableEnqueuer = asynqPRMergeableEnqueuer{client: asynqClient}
	}
	createPRUC := prusecase.NewCreatePRUsecase(prRepo, prAuditRepo, gitSvc, prTxManager, membershipRepo, prMergeableEnqueuer)
	pullRequestHandler := handler.NewPullRequestHandler(
		createPRUC,
		mergePRUC,
		createReviewUC,
		listReviewsUC,
		prRepo,
		prReviewRepo,
		prReviewCommentRepo,
		gitSvc,
		resolveRepo,
	)

	mcpVerificationHandler := handler.NewMCPVerificationHandler(
		runVerificationUC,
		getLatestVerificationUC,
		listHistoryVerificationUC,
		getJobStatusUC,
		deleteVerificationUC,
	)

	importJobRepo := infrarepo.NewImportJobRepository(sqlxDB)
	getImportUC := importUC.NewGetImportJobUsecase(importJobRepo)
	listImportUC := importUC.NewListImportJobsUsecase(importJobRepo)
	cancelImportUC := importUC.NewCancelImportJobUsecase(importJobRepo, membershipRepo)

	var createImportUC *importUC.CreateImportJobUsecase
	var retryImportUC *importUC.RetryImportJobUsecase
	if cfg.RedisAddr != "" {
		importAsynqClient := queue.NewAsynqClient(cfg.RedisAddr)
		createImportUC = importUC.NewCreateImportJobUsecase(importJobRepo, membershipRepo, repoRepo, entityOrgRepo, importAsynqClient)
		retryImportUC = importUC.NewRetryImportJobUsecase(importJobRepo, importJobRepo, membershipRepo, importAsynqClient)
	} else {
		noopImport := noopImportEnqueuer{}
		createImportUC = importUC.NewCreateImportJobUsecaseWithDeps(importJobRepo, membershipRepo, repoRepo, entityOrgRepo, nil, noopImport)
		retryImportUC = importUC.NewRetryImportJobUsecaseWithEnqueuer(importJobRepo, importJobRepo, membershipRepo, noopImport)
	}
	importHandler := handler.NewImportHandler(
		createImportUC,
		getImportUC,
		listImportUC,
		cancelImportUC,
		retryImportUC,
		orgRepo,
		legacyMembershipRepo,
	)

	exportAuditLogsUC := securityUC.NewExportAuditLogsUsecase(asynqClient)
	orgAuditLogHandler := handler.NewOrgAuditLogHandler(
		getOrgUC,
		membershipRepo,
		searchAuditLogsUC,
		exportAuditLogsUC,
	)

	securityAdvisoryRepo := infrarepo.NewSecurityAdvisoryRepository(sqlxDB)
	dependabotAlertRepo := infrarepo.NewDependabotAlertRepository(sqlxDB)
	listAdvisoriesUC := securityUC.NewListAdvisoriesUsecase(securityAdvisoryRepo)
	getAdvisoryUC := securityUC.NewGetAdvisoryUsecase(securityAdvisoryRepo)
	updateAdvisoryStateUC := securityUC.NewUpdateAdvisoryStateUsecase(securityAdvisoryRepo)
	listDependabotAlertsUC := securityUC.NewListDependabotAlertsUsecase(dependabotAlertRepo)
	updateDependabotAlertUC := securityUC.NewUpdateDependabotAlertUsecase(dependabotAlertRepo)
	securityAdvisoryHandler := handler.NewSecurityAdvisoryHandler(
		getOrgUC,
		membershipRepo,
		listAdvisoriesUC,
		getAdvisoryUC,
		updateAdvisoryStateUC,
		resolveRepo,
	)
	dependabotAlertHandler := handler.NewDependabotAlertHandler(
		membershipRepo,
		listDependabotAlertsUC,
		updateDependabotAlertUC,
		dependabotAlertRepo,
		resolveRepo,
	)

	// OAuth application management + the GitHub-compatible authorize/token
	// server flow. Authorization codes live in Redis when configured, else in
	// a process-local TTL store (fine for single-process deployments; codes
	// are short-lived and consumed once).
	oauthAppRepo := infrarepo.NewOAuthAppRepository(sqlxDB)
	oauthAccessTokenRepo := infrarepo.NewOAuthAccessTokenRepository(sqlxDB)
	oauthAuthorizationRepo := infrarepo.NewOAuthAuthorizationRepository(sqlxDB)
	var oauthCodeStore authUC.OAuthCodeStore
	if cfg.RedisAddr != "" {
		oauthCodeStore = kvstore.NewRedisTTLStore(redis.NewClient(&redis.Options{Addr: cfg.RedisAddr}))
	} else {
		oauthCodeStore = kvstore.NewInMemoryTTLStore()
	}
	oauthAuthorizeUC := authUC.NewOAuthAuthorizeUsecase(oauthAppRepo, oauthCodeStore)
	oauthTokenUC := authUC.NewOAuthTokenUsecase(oauthCodeStore, issuePATUC, oauthAppRepo, oauthAuthorizationRepo)
	oauthHandler := handler.NewOAuthHandler(oauthAuthorizeUC, oauthTokenUC)
	oauthAppHandler := handler.NewOAuthAppHandler(oauthAppRepo, oauthAccessTokenRepo, oauthAuthorizationRepo, tokenRepo)
	rateLimitHandler := handler.NewRateLimitHandler()
	rootHandler := handler.NewRootHandler()

	thirdPartyLicenses := loadLicensesFromFile(cfg.LicensesFilePath)
	metaHandler := handler.NewMetaHandler(handler.BuildInfo{
		AppName:     cfg.AppName,
		Version:     version,
		GitCommit:   commit,
		BuildDate:   buildTime,
		LicenseName: cfg.LicenseName,
		SourceURL:   cfg.SourceURL,
	}, thirdPartyLicenses)
	metaHandler.RegisterRoutes(e)

	gitHTTPHandler := handler.NewGitHTTPHandler(
		cfg.GitDataRoot,
		repoGitResolver,
		membershipAdapter,
		nil,
		collaboratorRepo,
		realGitBasicAuth,
	)
	sshKeyHandler := handler.NewSSHKeyHandler(sshKeyRepo)
	collaboratorHandler := handler.NewCollaboratorHandler(repoGitResolver, repoRepo, collaboratorRepo, entityUserRepo)

	api := e.Group("")
	e.POST("/register", authHandler.Register, middleware.AuthRateLimitMiddleware(10, 15*time.Minute))
	e.POST("/login", authHandler.Login, middleware.AuthRateLimitMiddleware(10, 15*time.Minute))
	// The web frontend posts to /api/v1/auth/{login,register}; register the same
	// handlers there so UI sign-in/sign-up work (the top-level routes above are
	// kept for API clients and backward compatibility).
	e.POST("/api/v1/auth/register", authHandler.Register, middleware.AuthRateLimitMiddleware(10, 15*time.Minute))
	e.POST("/api/v1/auth/login", authHandler.Login, middleware.AuthRateLimitMiddleware(10, 15*time.Minute))

	tokens := api.Group("/user/tokens", authMiddleware)
	tokens.GET("", tokenHandler.List)
	tokens.DELETE("/:id", tokenHandler.Revoke)
	api.Group("", authMiddleware, middleware.RequireScope("repo")).POST("/user/tokens", tokenHandler.Create)

	keys := api.Group("/user/keys", authMiddleware)
	keys.GET("", sshKeyHandler.List)
	keys.POST("", sshKeyHandler.Add)
	keys.DELETE("/:key_id", sshKeyHandler.Delete)

	repositoryHandler.RegisterRoutes(api, authMiddleware)
	branchHandler.RegisterRoutes(api, authMiddleware)
	collaboratorHandler.RegisterRoutes(api, authMiddleware)
	contentHandler.RegisterRoutes(api)
	issueHandler.RegisterRoutes(api, authMiddleware)
	labelHandler.RegisterRoutes(api, authMiddleware)
	milestoneHandler.RegisterRoutes(api, authMiddleware)
	pullRequestHandler.RegisterRoutes(api, authMiddleware)
	oauthHandler.RegisterRoutes(api, authMiddleware)
	api.GET("/rate_limit", rateLimitHandler.Get)
	api.GET("/", rootHandler.Get)
	// Capture the whole final segment as :repo and strip ".git" in the handler
	// (Echo cannot mix a param with a literal ".git" in one segment).
	e.GET("/:owner/:repo/info/refs", gitHTTPHandler.InfoRefs, realOptionalGitAuth)
	e.POST("/:owner/:repo/git-upload-pack", gitHTTPHandler.UploadPack, realOptionalGitAuth)
	e.POST("/:owner/:repo/git-receive-pack", gitHTTPHandler.ReceivePack, realGitBasicAuth)

	v3 := e.Group("/api/v3")
	v3.Use(middleware.GitHubCompatHeaders())
	v3.Use(middleware.RateLimitMiddleware(5000, 60))
	v3.Use(middleware.GitHubCommonHeadersMiddleware())

	userHandler.RegisterRoutes(v3, authMiddleware)
	userPreferencesHandler.RegisterRoutes(v3, authMiddleware)
	v3.Group("", authMiddleware, middleware.RequireScope("admin:org")).PUT("/orgs/:org/memberships/:username", orgHandler.UpdateMembership)
	orgHandler.RegisterRoutes(v3, authMiddleware)
	orgAuditLogHandler.RegisterRoutes(v3, authMiddleware)
	securityAdvisoryHandler.RegisterRoutes(v3, authMiddleware)
	dependabotAlertHandler.RegisterRoutes(v3, authMiddleware)
	// Wire authorization into every handler that must enforce it.
	contentHandler.SetAccess(repoAccess)
	branchHandler.SetAccess(repoAccess)
	labelHandler.SetAccess(repoAccess)
	milestoneHandler.SetAccess(repoAccess)
	issueHandler.SetAccess(repoAccess)
	webhookHandler.SetAccess(repoAccess)
	secretHandler.SetAccess(repoAccess)
	artifactHandler.SetAccess(repoAccess)
	pullRequestHandler.SetAccess(repoAccess)
	mcpVerificationHandler.SetAccess(repoAccess)
	securityAdvisoryHandler.SetAccess(repoAccess)

	repositoryHandler.RegisterRoutes(v3, authMiddleware)
	branchHandler.RegisterRoutes(v3, authMiddleware)
	collaboratorHandler.RegisterRoutes(v3, authMiddleware)
	contentHandler.RegisterRoutes(v3)
	issueHandler.RegisterRoutes(v3, authMiddleware)
	labelHandler.RegisterRoutes(v3, authMiddleware)
	milestoneHandler.RegisterRoutes(v3, authMiddleware)
	pullRequestHandler.RegisterRoutes(v3, authMiddleware)
	branchProtectionHandler.RegisterRoutes(v3, authMiddleware)
	webhookHandler.RegisterRoutes(v3, authMiddleware)
	secretHandler.RegisterRoutes(v3, authMiddleware)
	artifactHandler.RegisterRoutes(v3, authMiddleware)
	oauthAppHandler.RegisterRoutes(v3, authMiddleware)
	v3.GET("/rate_limit", rateLimitHandler.Get)
	v3.GET("", rootHandler.Get)

	v3Tokens := v3.Group("/user/tokens", authMiddleware)
	v3Tokens.GET("", tokenHandler.List)
	v3Tokens.DELETE("/:id", tokenHandler.Revoke)
	v3.Group("", authMiddleware, middleware.RequireScope("repo")).POST("/user/tokens", tokenHandler.Create)

	v3Keys := v3.Group("/user/keys", authMiddleware)
	v3Keys.GET("", sshKeyHandler.List)
	v3Keys.POST("", sshKeyHandler.Add)
	v3Keys.DELETE("/:key_id", sshKeyHandler.Delete)

	v1 := e.Group("/api/v1")

	docsRoot := getenv("DOCS_ROOT", ".")
	editBaseURL := getenv("DOCS_EDIT_BASE_URL", "")
	docsTreeUC := docsuc.NewGetDocTreeUsecase(docsRoot)
	docsSectionUC := docsuc.NewGetDocSectionUsecase(docsTreeUC)
	docsHandler := handler.NewDocsHandler(docsTreeUC, docsSectionUC, editBaseURL)
	docsHandler.RegisterRoutes(v1)

	contributorsHandler := handler.NewContributorsHandler(repoGitResolver, membershipAdapter)
	contributorsHandler.RegisterRoutes(v1)

	var healthMinioClient *minio.Client
	if cfg.MinioEndpoint != "" {
		client, minioErr := minio.New(cfg.MinioEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_SECRET_KEY"), ""),
			Secure: getenvBool("MINIO_USE_TLS", false),
		})
		if minioErr != nil {
			return nil, fmt.Errorf("init health minio client: %w", minioErr)
		}
		healthMinioClient = client
	}

	var healthRedisClient *redis.Client
	if cfg.RedisAddr != "" {
		healthRedisClient = redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	}

	apiV1HealthHandler := handler.NewAPIV1HealthHandler(sqlxDB, healthMinioClient, healthRedisClient)
	apiV1VersionHandler := handler.NewAPIV1VersionHandler()
	v1.GET("/health", apiV1HealthHandler.Handle)
	v1.GET("/version", apiV1VersionHandler.Handle)

	var asynqInspector *asynq.Inspector
	if cfg.RedisAddr != "" {
		asynqInspector = asynq.NewInspector(asynq.RedisClientOpt{Addr: cfg.RedisAddr})
	}
	adminStatusHandler := handler.NewAPIV1AdminStatusHandler(
		sqlxDB,
		healthMinioClient,
		healthRedisClient,
		asynqInspector,
		getenv("MINIO_DATA_PATH", "/"),
	)
	v1Ops := v1.Group("", authMiddleware)
	v1Ops.GET("/admin/status", adminStatusHandler.Handle)

	compatHandler.RegisterRoutes(v1, authMiddleware)
	mcpVerificationHandler.RegisterRoutes(v1, authMiddleware)
	importHandler.RegisterRoutes(v1, authMiddleware)
	branchProtectionHandler.RegisterInternalRoutes(e.Group("/api/internal"), authMiddleware)

	gqlResolver := &graph.Resolver{
		UserRepo:         entityUserRepo,
		LabelRepo:        labelRepo,
		MilestoneRepo:    milestoneRepo,
		RepositoryRepo:   repoRepo,
		GetCurrentUserUC: getCurrentUserUC,
		GetUserByLoginUC: getUserByLoginUC,
		GetRepositoryUC:  getRepoUC,
		GetOrgUC:         getOrgUC,
		CreateIssueUC:    createIssueUC,
		UpdateIssueUC:    updateIssueUC,
		CreateCommentUC:  createCommentUC,
		ListIssuesUC:     listIssuesUC,
		CreatePRUC:       createPRUC,
		MergePRUC:        mergePRUC,
	}
	gqlHandler := graph.NewHandler(gqlResolver, &cfg)
	e.POST("/api/graphql", gqlHandler, authMiddleware)
	e.GET("/api/graphql", gqlHandler)

	workflowJobRepo := infrarepo.NewWorkflowJobRepository(sqlxDB)
	// Job logs are persisted in job_log_lines / job_logs_meta so the actions
	// log read + SSE endpoints return real CI output.
	jobLogRepo := infrarepo.NewJobLogRepository(sqlxDB)
	var jobLogSub *queue.JobLogSubscriber
	var jobLogPublisher *queue.JobLogPublisher
	if cfg.RedisAddr != "" {
		jobLogSub = queue.NewJobLogSubscriber(cfg.RedisAddr)
		jobLogPublisher = queue.NewJobLogPublisher(cfg.RedisAddr)
	}
	actionsLogHandler := handler.NewActionsLogHandler(jobLogRepo, workflowJobRepo, jobLogSub, repoRepo)
	actionsLogHandler.SetAccess(repoAccess)
	actionsLogHandler.RegisterRoutes(v1, authMiddleware)

	apiActions := e.Group("/api")
	actionsLogHandler.RegisterRoutes(apiActions, authMiddleware)

	// ---- Actions / CI runtime ------------------------------------------------
	// The CI worker executes workflow YAML (jobs + steps), streaming masked
	// logs to job_log_lines. The dispatcher runs it via asynq when Redis is
	// configured, else in-process. The trigger creates workflow_runs from git
	// pushes and manual dispatches; the workflow-run + runner HTTP handlers
	// expose them to the UI and self-hosted runners.
	ciDecrypter := func(_ context.Context, encrypted string) (string, error) {
		plain, err := actionSecretEnc.SecretEncryptor.Decrypt([]byte(encrypted))
		if err != nil {
			return "", err
		}
		return string(plain), nil
	}
	ciWorker := worker.NewCIWorker(db).
		WithDecrypter(ciDecrypter).
		WithLogRepository(jobLogRepo).
		WithJobRepository(workflowJobRepo).
		WithSandbox(cfg.CISandboxMode, cfg.CISandboxImage)
	if jobLogPublisher != nil {
		ciWorker = ciWorker.WithLogPublisher(jobLogPublisher)
	}
	// CI executes in-process (a detached goroutine per run). This is the
	// single-node execution model; distributing runs to a separate worker
	// fleet over asynq is a future enhancement. Runs are persisted either way,
	// so the UI reflects real state.
	ciDispatcher := ci.NewDispatcher(nil, ciWorker)
	ciTrigger := ci.NewTrigger(wfRepo, ciDispatcher)

	// Re-dispatch execution when a run is re-run from the UI.
	wfRepo.SetRerunDispatcher(func(_ context.Context, orgID uuid.UUID, run *entity.WorkflowRun) {
		bg := context.Background()
		repo, rerr := repoRepo.GetByID(bg, run.RepositoryID, orgID)
		if rerr != nil || repo == nil {
			return
		}
		_ = ciTrigger.Redispatch(bg, orgID, repo.GitPath, run)
	})

	// Fire CI on push (HTTP + SSH).
	gitHTTPHandler.SetPushListener(func(ctx context.Context, repo *handler.ResolvedGitRepository, branch, newSHA string, userID int64) {
		_ = ciTrigger.OnPush(context.Background(), repo.OrganizationID, repo.ID, repo.DiskPath, branch, newSHA, loginForUserID(context.Background(), userRepo, userID))
	})

	// Workflow run read/cancel/rerun/jobs endpoints (the Actions UI).
	workflowRunHandler := handler.NewWorkflowRunHandler(
		workflowusecase.NewListRunsUsecase(wfRepo),
		workflowusecase.NewGetRunUsecase(wfRepo),
		workflowusecase.NewCancelRunUsecase(wfRepo),
		workflowusecase.NewRerunRunUsecase(wfRepo),
		workflowusecase.NewListJobsUsecase(infrarepo.NewWorkflowJobListingAdapter(workflowJobRepo)),
		resolveRepo,
		nil,
		nil,
	)
	workflowRunHandler.SetAccess(repoAccess)
	workflowRunHandler.RegisterRoutes(v3, authMiddleware)

	// Manual workflow dispatch.
	actionsDispatchHandler := handler.NewActionsDispatchHandler(
		resolveRepo,
		func(c echo.Context, repo *entity.Repository) bool {
			if repo.OrganizationID == middleware.UserUUIDFromContext(c) {
				return true
			}
			ok, err := membershipAdapter.HasWriteAccess(c.Request().Context(), middleware.UserIDFromContext(c), repo.OrganizationID)
			if err == nil && ok {
				return true
			}
			perm, perr := collaboratorRepo.GetPermission(c.Request().Context(), repo.ID, middleware.UserUUIDFromContext(c))
			return perr == nil && (perm == entity.CollaboratorPermWrite || perm == entity.CollaboratorPermAdmin)
		},
		func(ctx context.Context, diskPath, ref string) (string, error) {
			return gitSvc.ResolveRef(ctx, diskPath, ref)
		},
		func(c echo.Context) string { return loginForUserID(c.Request().Context(), userRepo, middleware.UserIDFromContext(c)) },
		func(ctx context.Context, repo *entity.Repository, workflowFile, branch, sha, actor string) (*entity.WorkflowRun, error) {
			return ciTrigger.DispatchWorkflow(ctx, repo.OrganizationID, repo.ID, repo.GitPath, workflowFile, branch, sha, actor)
		},
	)
	actionsDispatchHandler.RegisterRoutes(v3, authMiddleware)

	// Self-hosted runner registration/heartbeat API.
	runnerRepo := infrarepo.NewRunnerRepository(sqlxDB)
	runnerTokenRepo := infrarepo.NewRunnerRegistrationTokenRepository(sqlxDB)
	runnerAuditRepo := infrarepo.NewAuditLogRepository(sqlxDB)
	runnerHandler := handler.NewRunnerHandlerWithDeps(
		actionsusecase.NewCreateRegistrationTokenUsecase(runnerTokenRepo),
		actionsusecase.NewRegisterRunnerUsecase(runnerRepo, runnerTokenRepo, runnerAuditRepo),
		actionsusecase.NewListRunnersUsecase(runnerRepo),
		actionsusecase.NewDeleteRunnerUsecase(runnerRepo, runnerAuditRepo),
		actionsusecase.NewHeartbeatRunnerUsecase(runnerRepo),
		func(c echo.Context) (uuid.UUID, error) {
			org := c.Param("org")
			if o, oerr := orgRepo.GetByLogin(c.Request().Context(), org); oerr == nil && o != nil {
				return middleware.Int64ToUUID(o.ID), nil
			}
			// Fall back to a personal namespace: the owner's user id.
			if u, uerr := entityUserRepo.GetByLogin(c.Request().Context(), org); uerr == nil && u != nil {
				return u.ID, nil
			}
			return uuid.Nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		},
		func(c echo.Context, orgID uuid.UUID) (string, error) {
			// Owner of a personal namespace, or an org admin/owner.
			if orgID == middleware.UserUUIDFromContext(c) {
				return entity.RoleAdmin, nil
			}
			role, err := membershipRepo.GetRole(c.Request().Context(), orgID, middleware.UserUUIDFromContext(c))
			if err != nil {
				return entity.RoleMember, nil
			}
			return role, nil
		},
	)
	// Frontend calls /api/v1/:org/actions/runners*, so mount under that prefix.
	runnerHandler.RegisterRoutes(v1.Group("/:org/actions", authMiddleware))

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
			&sshRepoAuthorizer{
				resolver:      repoGitResolver,
				memberships:   membershipAdapter,
				collaborators: collaboratorRepo,
			},
			hostKey,
		)
		go func() {
			middleware.Log().Info("ssh server listening", "addr", sshListenAddr)
			if err := sshServer.Start(sshListenAddr); err != nil && !errors.Is(err, gossh.ErrServerClosed) {
				middleware.Log().Info("ssh server stopped", "error", err)
			}
		}()
	}
	return sshServer, nil
}

func loadLicensesFromFile(path string) []handler.LicenseEntry {
	log := logger.Global()
	data, err := os.ReadFile(path)
	if err != nil {
		log.Warn().Err(err).Str("path", path).Msg("failed to load licenses file")
		return []handler.LicenseEntry{}
	}

	var entries []handler.LicenseEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		log.Warn().Err(err).Str("path", path).Msg("failed to parse licenses file")
		return []handler.LicenseEntry{}
	}
	return entries
}

type noopMCPEnqueuer struct{}

func (noopMCPEnqueuer) EnqueueMCPVerification(context.Context, queue.MCPVerificationPayload) error {
	return errors.New("redis not configured")
}

type noopImportEnqueuer struct{}

func (noopImportEnqueuer) EnqueueGitHubImport(context.Context, uuid.UUID, uuid.UUID) error {
	return errors.New("redis not configured")
}

type asynqPRMergeableEnqueuer struct {
	client *asynq.Client
}

func (e asynqPRMergeableEnqueuer) Enqueue(ctx context.Context, payload prusecase.PRMergeableEnqueuePayload) error {
	_, err := queue.EnqueuePRMergeableCheck(ctx, e.client, queue.PRMergeableCheckPayload{
		GitPath: payload.GitPath,
		HeadRef: payload.HeadRef,
		BaseRef: payload.BaseRef,
		PRID:    payload.PRID,
	})
	return err
}

type artifactMinioStorage struct {
	client *minio.Client
}

func newArtifactMinioStorage(endpoint, accessKey, secretKey string, useTLS bool) (*artifactMinioStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useTLS,
	})
	if err != nil {
		return nil, err
	}
	return &artifactMinioStorage{client: client}, nil
}

func (s *artifactMinioStorage) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}

func (s *artifactMinioStorage) PresignedPutURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedPutObject(ctx, bucket, key, expiry)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *artifactMinioStorage) PresignedGetURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *artifactMinioStorage) DeleteObject(ctx context.Context, bucket, key string) error {
	return s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

func getenv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getenvBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
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

// loginForUserID resolves a user id to its login for CI run attribution,
// returning "" when the user cannot be found (attribution is best-effort).
func loginForUserID(ctx context.Context, users repo.IUserRepository, userID int64) string {
	if userID == 0 {
		return ""
	}
	if u, err := users.GetByID(ctx, userID); err == nil && u != nil {
		return u.Login
	}
	return ""
}

// personalOrgStore is the slice of the entity organization repository the
// personal-org creator needs.
type personalOrgStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Organization, error)
	Create(ctx context.Context, org *entity.Organization) error
}

// personalOrgCreator provisions a user's personal organization (id == user id)
// so org-scoped foreign keys resolve for personal repositories. Idempotent:
// an existing row (e.g. from the backfill migration) is treated as success.
type personalOrgCreator struct {
	orgs personalOrgStore
}

func (p *personalOrgCreator) EnsurePersonalOrg(ctx context.Context, userID int64, login string) error {
	orgID := middleware.Int64ToUUID(userID)
	if existing, err := p.orgs.GetByID(ctx, orgID); err == nil && existing != nil {
		return nil
	}
	err := p.orgs.Create(ctx, &entity.Organization{
		ID:       orgID,
		Login:    login,
		Name:     login,
		PlanTier: entity.PlanFree,
	})
	if errors.Is(err, domain.ErrConflict) {
		return nil
	}
	return err
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

		logAttrs := []any{
			"request_id", requestID,
			"http_method", c.Request().Method,
			"path", c.Request().URL.Path,
			"request_headers", logger.MaskHeaders(c.Request().Header),
		}

		var he *echo.HTTPError
		if errors.As(err, &he) {
			message := httpErrorMessage(he)
			code := httpStatusToCode(he.Code)
			middleware.Log().Error("request error", append(logAttrs, "status", he.Code, "error", message)...)
			if writeErr := handler.RespondError(c, he.Code, code, message, requestID); writeErr != nil {
				c.Logger().Error(writeErr)
			}
			return
		}

		status, code := handler.DomainErrorToHTTP(err)
		middleware.Log().Error("request error", append(logAttrs, "status", status, "error", err.Error())...)
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

type branchProtectionReadRepo struct {
	db *sqlx.DB
}

func (r *branchProtectionReadRepo) GetByPattern(ctx context.Context, orgID, repoID uuid.UUID, pattern string) (*handler.BranchProtectionDetail, error) {
	query := `SELECT pattern, required_reviews FROM branch_protections WHERE organization_id = ? AND repository_id = ? AND pattern = ?`
	query = r.db.Rebind(query)

	var (
		p       string
		reviews int
	)
	err := r.db.QueryRowxContext(ctx, query, orgID, repoID, pattern).Scan(&p, &reviews)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &handler.BranchProtectionDetail{
		Pattern:                      p,
		RequiredApprovingReviewCount: reviews,
		RequiredStatusChecksContexts: []string{},
	}, nil
}

func (r *branchProtectionReadRepo) ListByRepository(ctx context.Context, orgID, repoID uuid.UUID) ([]*handler.BranchProtectionDetail, error) {
	query := `SELECT pattern, required_reviews FROM branch_protections WHERE organization_id = ? AND repository_id = ? ORDER BY pattern`
	query = r.db.Rebind(query)

	rows, err := r.db.QueryxContext(ctx, query, orgID, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*handler.BranchProtectionDetail, 0)
	for rows.Next() {
		var (
			p       string
			reviews int
		)
		if err := rows.Scan(&p, &reviews); err != nil {
			return nil, err
		}
		result = append(result, &handler.BranchProtectionDetail{
			Pattern:                      p,
			RequiredApprovingReviewCount: reviews,
			RequiredStatusChecksContexts: []string{},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

type branchProtectionWriteRepo struct {
	db *sqlx.DB
}

func (r *branchProtectionWriteRepo) GetByPattern(ctx context.Context, orgID, repoID uuid.UUID, pattern string) (*repoUC.BranchProtectionRule, error) {
	query := `SELECT pattern, required_reviews FROM branch_protections WHERE organization_id = ? AND repository_id = ? AND pattern = ?`
	query = r.db.Rebind(query)

	var (
		p       string
		reviews int
	)
	err := r.db.QueryRowxContext(ctx, query, orgID, repoID, pattern).Scan(&p, &reviews)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &repoUC.BranchProtectionRule{
		Pattern:                      p,
		RequiredApprovingReviewCount: reviews,
	}, nil
}

func (r *branchProtectionWriteRepo) Upsert(ctx context.Context, orgID, repoID uuid.UUID, rule *repoUC.BranchProtectionRule) (*repoUC.BranchProtectionRule, error) {
	query := `
		INSERT OR REPLACE INTO branch_protections (
			id, organization_id, repository_id, pattern, required_reviews, required_checks, created_at
		) VALUES (
			:id, :organization_id, :repository_id, :pattern, :required_reviews, :required_checks, :created_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":               uuid.New(),
		"organization_id":  orgID,
		"repository_id":    repoID,
		"pattern":          rule.Pattern,
		"required_reviews": rule.RequiredApprovingReviewCount,
		"required_checks":  "[]",
		"created_at":       time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	return rule, nil
}

func (r *branchProtectionWriteRepo) DeleteByPattern(ctx context.Context, orgID, repoID uuid.UUID, pattern string) error {
	query := `DELETE FROM branch_protections WHERE organization_id = ? AND repository_id = ? AND pattern = ?`
	query = r.db.Rebind(query)

	result, err := r.db.ExecContext(ctx, query, orgID, repoID, pattern)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

// actionSecretEncryptor adapts SecretEncryptor for GitHub-compatible secrets API wiring.
type actionSecretEncryptor struct {
	*crypto.SecretEncryptor
}

func newActionSecretEncryptorFromEnv() *actionSecretEncryptor {
	return &actionSecretEncryptor{SecretEncryptor: crypto.NewSecretEncryptorFromEnv()}
}

func (e *actionSecretEncryptor) KeyID() string {
	return "open-git-action-secrets-v1"
}

func (e *actionSecretEncryptor) PublicKeyBase64() string {
	// Placeholder public key for GitHub API shape compatibility.
	return "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
}

func (e *actionSecretEncryptor) DecryptSealedBox(ciphertext []byte) ([]byte, error) {
	return e.Decrypt(ciphertext)
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
