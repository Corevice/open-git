package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/middleware"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
)

type PutBranchProtectionRequest struct {
	RequiredStatusChecks struct {
		Strict   bool     `json:"strict"`
		Contexts []string `json:"contexts"`
	} `json:"required_status_checks"`
	RequiredPullRequestReviews struct {
		DismissStaleReviews          bool `json:"dismiss_stale_reviews"`
		RequireCodeOwnerReviews      bool `json:"require_code_owner_reviews"`
		RequiredApprovingReviewCount int  `json:"required_approving_review_count"`
	} `json:"required_pull_request_reviews"`
	Restrictions                   interface{} `json:"restrictions"`
	EnforceAdmins                  bool        `json:"enforce_admins"`
	AllowForcePushes               bool        `json:"allow_force_pushes"`
	AllowDeletions                 bool        `json:"allow_deletions"`
	RequiredLinearHistory          bool        `json:"required_linear_history"`
	RequiredConversationResolution bool        `json:"required_conversation_resolution"`
}

type branchProtectionResponse struct {
	RequiredStatusChecks struct {
		Strict   bool     `json:"strict"`
		Contexts []string `json:"contexts"`
	} `json:"required_status_checks"`
	RequiredPullRequestReviews struct {
		DismissStaleReviews          bool `json:"dismiss_stale_reviews"`
		RequireCodeOwnerReviews      bool `json:"require_code_owner_reviews"`
		RequiredApprovingReviewCount int  `json:"required_approving_review_count"`
	} `json:"required_pull_request_reviews"`
	Restrictions                   interface{} `json:"restrictions"`
	EnforceAdmins                  bool        `json:"enforce_admins"`
	AllowForcePushes               bool        `json:"allow_force_pushes"`
	AllowDeletions                 bool        `json:"allow_deletions"`
	RequiredLinearHistory          bool        `json:"required_linear_history"`
	RequiredConversationResolution bool        `json:"required_conversation_resolution"`
}

type BranchProtectionDetail struct {
	Pattern                          string
	RequiredApprovingReviewCount     int
	DismissStaleReviews              bool
	RequireCodeOwnerReviews          bool
	RequiredStatusChecksStrict       bool
	RequiredStatusChecksContexts     []string
	EnforceAdmins                    bool
	AllowForcePushes                 bool
	AllowDeletions                   bool
	RequiredLinearHistory            bool
	RequiredConversationResolution   bool
}

type IBranchProtectionRepository interface {
	GetByPattern(ctx context.Context, orgID, repoID uuid.UUID, pattern string) (*BranchProtectionDetail, error)
	ListByRepository(ctx context.Context, orgID, repoID uuid.UUID) ([]*BranchProtectionDetail, error)
}

type BranchProtectionHandler struct {
	branchProtectionRepo IBranchProtectionRepository
	upsertUC             *repoUC.UpsertBranchProtectionUsecase
	deleteUC             *repoUC.DeleteBranchProtectionUsecase
	auditLog             domainrepo.IAuditLogRepository
	resolveRepo          func(c echo.Context, owner, repo string) (*entity.Repository, error)
	checkRepoAdmin       func(c echo.Context, repo *entity.Repository) error
}

func NewBranchProtectionHandler(
	branchProtectionRepo IBranchProtectionRepository,
	upsertUC *repoUC.UpsertBranchProtectionUsecase,
	deleteUC *repoUC.DeleteBranchProtectionUsecase,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
	checkRepoAdmin func(c echo.Context, repo *entity.Repository) error,
	auditLog domainrepo.IAuditLogRepository,
) *BranchProtectionHandler {
	return &BranchProtectionHandler{
		branchProtectionRepo: branchProtectionRepo,
		upsertUC:             upsertUC,
		deleteUC:             deleteUC,
		auditLog:             auditLog,
		resolveRepo:          resolveRepo,
		checkRepoAdmin:       checkRepoAdmin,
	}
}

func (h *BranchProtectionHandler) recordAudit(ctx context.Context, orgID, actorID uuid.UUID, actorLogin, action, rulePattern string) {
	if h.auditLog == nil {
		return
	}
	_ = actorLogin
	_ = h.auditLog.Create(ctx, &entity.AuditLog{
		OrganizationID: orgID,
		ActorID:        actorID,
		Action:         action,
		TargetType:     "branch_protection",
		TargetID:       rulePattern,
		CreatedAt:      time.Now().UTC(),
	})
}

func (h *BranchProtectionHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	repoScope := middleware.RequireScope("repo")
	g.GET("/repos/:owner/:repo/branches/:branch/protection", h.GetBranchProtection, auth, repoScope)
	g.PUT("/repos/:owner/:repo/branches/:branch/protection", h.UpsertBranchProtection, auth, repoScope)
	g.DELETE("/repos/:owner/:repo/branches/:branch/protection", h.DeleteBranchProtection, auth, repoScope)
}

func (h *BranchProtectionHandler) RegisterInternalRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	g.GET("/repos/:owner/:repo/branch-protections", h.ListBranchProtections, auth)
}

func (h *BranchProtectionHandler) GetBranchProtection(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	pattern := c.Param("branch")
	rule, err := h.branchProtectionRepo.GetByPattern(c.Request().Context(), repo.OrganizationID, repo.ID, pattern)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Branch not protected"})
		}
		return err
	}

	return c.JSON(http.StatusOK, toBranchProtectionResponse(rule))
}

func (h *BranchProtectionHandler) UpsertBranchProtection(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	if err := h.checkRepoAdmin(c, repo); err != nil {
		return err
	}

	var req PutBranchProtectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid request"})
	}

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	pattern := c.Param("branch")
	ctx := c.Request().Context()
	action := "branch_protection.create"
	if _, err := h.branchProtectionRepo.GetByPattern(ctx, repo.OrganizationID, repo.ID, pattern); err == nil {
		action = "branch_protection.update"
	} else if !errors.Is(err, apperror.ErrNotFound) {
		return err
	}

	result, err := h.upsertUC.Execute(ctx, repo.OrganizationID, repo.ID, actorID, &repoUC.BranchProtectionRule{
		Pattern:                      pattern,
		RequiredApprovingReviewCount: req.RequiredPullRequestReviews.RequiredApprovingReviewCount,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "validation failed"})
		}
		return err
	}

	h.recordAudit(ctx, repo.OrganizationID, actorID, "", action, pattern)

	detail := branchProtectionDetailFromRequest(pattern, &req)
	detail.RequiredApprovingReviewCount = result.RequiredApprovingReviewCount
	return c.JSON(http.StatusOK, toBranchProtectionResponse(detail))
}

func (h *BranchProtectionHandler) DeleteBranchProtection(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	if err := h.checkRepoAdmin(c, repo); err != nil {
		return err
	}

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	pattern := c.Param("branch")
	ctx := c.Request().Context()
	if err := h.deleteUC.Execute(ctx, repo.OrganizationID, repo.ID, actorID, pattern); err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Branch not protected"})
		}
		return err
	}

	h.recordAudit(ctx, repo.OrganizationID, actorID, "", "branch_protection.delete", pattern)

	return c.NoContent(http.StatusNoContent)
}

func (h *BranchProtectionHandler) ListBranchProtections(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	rules, err := h.branchProtectionRepo.ListByRepository(c.Request().Context(), repo.OrganizationID, repo.ID)
	if err != nil {
		return err
	}

	resp := make([]branchProtectionResponse, 0, len(rules))
	for _, rule := range rules {
		resp = append(resp, toBranchProtectionResponse(rule))
	}
	return c.JSON(http.StatusOK, resp)
}

func branchProtectionDetailFromRequest(pattern string, req *PutBranchProtectionRequest) *BranchProtectionDetail {
	contexts := req.RequiredStatusChecks.Contexts
	if contexts == nil {
		contexts = []string{}
	}
	return &BranchProtectionDetail{
		Pattern:                          pattern,
		RequiredApprovingReviewCount:     req.RequiredPullRequestReviews.RequiredApprovingReviewCount,
		DismissStaleReviews:              req.RequiredPullRequestReviews.DismissStaleReviews,
		RequireCodeOwnerReviews:          req.RequiredPullRequestReviews.RequireCodeOwnerReviews,
		RequiredStatusChecksStrict:       req.RequiredStatusChecks.Strict,
		RequiredStatusChecksContexts:     contexts,
		EnforceAdmins:                    req.EnforceAdmins,
		AllowForcePushes:                 req.AllowForcePushes,
		AllowDeletions:                   req.AllowDeletions,
		RequiredLinearHistory:            req.RequiredLinearHistory,
		RequiredConversationResolution:   req.RequiredConversationResolution,
	}
}

func toBranchProtectionResponse(detail *BranchProtectionDetail) branchProtectionResponse {
	contexts := detail.RequiredStatusChecksContexts
	if contexts == nil {
		contexts = []string{}
	}
	resp := branchProtectionResponse{
		Restrictions:                   nil,
		EnforceAdmins:                  detail.EnforceAdmins,
		AllowForcePushes:                 detail.AllowForcePushes,
		AllowDeletions:                   detail.AllowDeletions,
		RequiredLinearHistory:            detail.RequiredLinearHistory,
		RequiredConversationResolution: detail.RequiredConversationResolution,
	}
	resp.RequiredStatusChecks.Strict = detail.RequiredStatusChecksStrict
	resp.RequiredStatusChecks.Contexts = contexts
	resp.RequiredPullRequestReviews.DismissStaleReviews = detail.DismissStaleReviews
	resp.RequiredPullRequestReviews.RequireCodeOwnerReviews = detail.RequireCodeOwnerReviews
	resp.RequiredPullRequestReviews.RequiredApprovingReviewCount = detail.RequiredApprovingReviewCount
	return resp
}
