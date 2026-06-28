package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/service"
	"github.com/open-git/backend/internal/middleware"
	prusecase "github.com/open-git/backend/internal/usecase/pr"
)

type PullRequestHandler struct {
	createPRUC        *prusecase.CreatePRUsecase
	mergePRUC         *prusecase.MergePRUsecase
	prRepo            repository.IPullRequestRepository
	reviewRepo        repository.IReviewRepository
	reviewCommentRepo repository.IReviewCommentRepository
	gitSvc            service.GitService
	resolveRepo       func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

func NewPullRequestHandler(
	createPRUC *prusecase.CreatePRUsecase,
	mergePRUC *prusecase.MergePRUsecase,
	prRepo repository.IPullRequestRepository,
	reviewRepo repository.IReviewRepository,
	reviewCommentRepo repository.IReviewCommentRepository,
	gitSvc service.GitService,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *PullRequestHandler {
	return &PullRequestHandler{
		createPRUC:        createPRUC,
		mergePRUC:         mergePRUC,
		prRepo:            prRepo,
		reviewRepo:        reviewRepo,
		reviewCommentRepo: reviewCommentRepo,
		gitSvc:            gitSvc,
		resolveRepo:       resolveRepo,
	}
}

func (h *PullRequestHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	repoScope := middleware.RequireScope("repo")
	g.GET("/repos/:owner/:repo/pulls", h.ListPullRequests, auth, repoScope)
	g.POST("/repos/:owner/:repo/pulls", h.CreatePullRequest, auth, repoScope)
	g.GET("/repos/:owner/:repo/pulls/:number", h.GetPullRequest, auth, repoScope)
	g.PATCH("/repos/:owner/:repo/pulls/:number", h.UpdatePullRequest, auth, repoScope)
	g.PUT("/repos/:owner/:repo/pulls/:number/merge", h.MergePullRequest, auth, repoScope)
	g.GET("/repos/:owner/:repo/pulls/:number/files", h.GetPullRequestFiles, auth, repoScope)
	g.POST("/repos/:owner/:repo/pulls/:number/reviews", h.CreateReview, auth, repoScope)
	g.GET("/repos/:owner/:repo/pulls/:number/reviews", h.ListReviews, auth, repoScope)
	g.GET("/repos/:owner/:repo/pulls/:number/comments", h.ListReviewComments, auth, repoScope)
	g.POST("/repos/:owner/:repo/pulls/:number/comments", h.CreateReviewComment, auth, repoScope)
}

type createPullRequestRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
}

type updatePullRequestRequest struct {
	Title *string `json:"title"`
	Body  *string `json:"body"`
	State *string `json:"state"`
}

type mergePullRequestRequest struct {
	MergeMethod string `json:"merge_method"`
}

type createReviewRequest struct {
	Event    string                      `json:"event"`
	Body     string                      `json:"body"`
	Comments []createReviewCommentItem   `json:"comments"`
}

type createReviewCommentItem struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Body string `json:"body"`
}

type createReviewCommentRequest struct {
	Body string `json:"body"`
	Path string `json:"path"`
	Line int    `json:"line"`
	Side string `json:"side"`
}

type pullRequestResponse struct {
	ID       uuid.UUID `json:"id"`
	Number   int       `json:"number"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	HeadRef  string    `json:"head_ref"`
	BaseRef  string    `json:"base_ref"`
	State    string    `json:"state"`
	NodeID   string    `json:"node_id"`
	HTMLURL  string    `json:"html_url"`
	MergedAt *string   `json:"merged_at"`
}

type mergePullRequestResponse struct {
	Merged  bool   `json:"merged"`
	Message string `json:"message"`
}

type pullRequestFileResponse struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch"`
	SHA       string `json:"sha"`
	BlobURL   string `json:"blob_url"`
	RawURL    string `json:"raw_url"`
}

type pullRequestFilesResponse struct {
	Files     []pullRequestFileResponse `json:"files"`
	Truncated bool                      `json:"truncated"`
}

type reviewResponse struct {
	ID            uuid.UUID `json:"id"`
	PullRequestID uuid.UUID `json:"pull_request_id"`
	User          uuid.UUID `json:"user"`
	State         string    `json:"state"`
	Body          string    `json:"body"`
	CommitSHA     string    `json:"commit_sha"`
	SubmittedAt   string    `json:"submitted_at"`
}

type reviewCommentResponse struct {
	ID            uuid.UUID  `json:"id"`
	PullRequestID uuid.UUID  `json:"pull_request_id"`
	ReviewID      *uuid.UUID `json:"review_id"`
	User          uuid.UUID  `json:"user"`
	Path          string     `json:"path"`
	Body          string     `json:"body"`
	Line          int        `json:"line"`
	Side          string     `json:"side"`
	CreatedAt     string     `json:"created_at"`
	UpdatedAt     string     `json:"updated_at"`
}

var reviewEventToState = map[string]string{
	"APPROVE":          entity.ReviewStateApproved,
	"REQUEST_CHANGES":  entity.ReviewStateChangesRequested,
	"COMMENT":          entity.ReviewStateCommented,
}

func (h *PullRequestHandler) ListPullRequests(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	state := c.QueryParam("state")
	if state == "" {
		state = "open"
	}

	pulls, total, err := h.prRepo.ListByRepo(c.Request().Context(), repo.ID, repository.ListPullRequestsFilter{
		State:   state,
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		return err
	}

	setPaginationHeaders(c, page, perPage, total)
	return c.JSON(http.StatusOK, toPullRequestResponses(pulls, c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *PullRequestHandler) CreatePullRequest(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	var req createPullRequestRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Title == "" {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", []GitHubFieldError{{Resource: "PullRequest", Field: "title", Code: "missing_field"}})
	}
	if req.Head == req.Base {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", []GitHubFieldError{{Resource: "PullRequest", Field: "head", Code: "invalid"}})
	}

	pr, err := h.createPRUC.Execute(c.Request().Context(), prusecase.CreatePRInput{
		OrganizationID: actor.OrganizationID,
		RepositoryID:   repo.ID,
		GitPath:        repo.GitPath,
		ActorID:        actor.UserID,
		Title:          req.Title,
		Body:           req.Body,
		HeadRef:        req.Head,
		BaseRef:        req.Base,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return err
	}

	return c.JSON(http.StatusCreated, toPullRequestResponse(pr, c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *PullRequestHandler) GetPullRequest(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pull request number")
	}

	pr, err := h.prRepo.GetByNumber(c.Request().Context(), repo.ID, number)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, toPullRequestResponse(pr, c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *PullRequestHandler) UpdatePullRequest(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pull request number")
	}

	pr, err := h.prRepo.GetByNumber(c.Request().Context(), repo.ID, number)
	if err != nil {
		return err
	}

	var req updatePullRequestRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Title != nil {
		length := utf8.RuneCountInString(*req.Title)
		if length < 1 || length > 256 {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid title")
		}
		pr.Title = *req.Title
	}
	if req.Body != nil {
		pr.Body = *req.Body
	}
	if req.State != nil {
		pr.State = *req.State
	}

	if err := h.prRepo.Update(c.Request().Context(), pr); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, toPullRequestResponse(pr, c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *PullRequestHandler) MergePullRequest(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pull request number")
	}

	var req mergePullRequestRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.MergeMethod != "" && req.MergeMethod != "merge" && req.MergeMethod != "squash" && req.MergeMethod != "rebase" {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid merge_method")
	}

	_, err = h.mergePRUC.Execute(c.Request().Context(), prusecase.MergePRInput{
		OrganizationID: actor.OrganizationID,
		RepositoryID:   repo.ID,
		GitPath:        repo.GitPath,
		ActorID:        actor.UserID,
		Number:         number,
		MergeMethod:    req.MergeMethod,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrAlreadyMerged) || errors.Is(err, apperror.ErrProtectionNotSatisfied) {
			return echo.NewHTTPError(http.StatusMethodNotAllowed, err.Error())
		}
		if errors.Is(err, apperror.ErrConflict) {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return err
	}

	return c.JSON(http.StatusOK, mergePullRequestResponse{
		Merged:  true,
		Message: "Pull Request successfully merged",
	})
}

func (h *PullRequestHandler) GetPullRequestFiles(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pull request number")
	}

	pr, err := h.prRepo.GetByNumber(c.Request().Context(), repo.ID, number)
	if err != nil {
		return err
	}

	base := pr.BaseSHA
	head := pr.HeadSHA
	if base == "" {
		base = pr.BaseRef
	}
	if head == "" {
		head = pr.HeadRef
	}

	diffs, truncated, err := h.gitSvc.GetDiff(c.Request().Context(), repo.GitPath, base, head, 3000)
	if err != nil {
		return err
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")
	host := c.Request().Host
	files := make([]pullRequestFileResponse, 0, len(diffs))
	for _, d := range diffs {
		blobURL := "https://" + host + "/" + owner + "/" + repoName + "/blob/" + head + "/" + d.Filename
		rawURL := "https://" + host + "/" + owner + "/" + repoName + "/raw/" + head + "/" + d.Filename
		files = append(files, pullRequestFileResponse{
			Filename:  d.Filename,
			Status:    d.Status,
			Additions: d.Additions,
			Deletions: d.Deletions,
			Patch:     d.Patch,
			SHA:       head,
			BlobURL:   blobURL,
			RawURL:    rawURL,
		})
	}

	return c.JSON(http.StatusOK, pullRequestFilesResponse{
		Files:     files,
		Truncated: truncated,
	})
}

func (h *PullRequestHandler) CreateReview(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pull request number")
	}

	pr, err := h.prRepo.GetByNumber(c.Request().Context(), repo.ID, number)
	if err != nil {
		return err
	}
	if pr.State == entity.PullRequestStateClosed || pr.State == entity.PullRequestStateMerged {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "cannot review a closed pull request")
	}

	var req createReviewRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	state, ok := reviewEventToState[strings.ToUpper(req.Event)]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid review event")
	}

	now := time.Now().UTC()
	review := &entity.Review{
		ID:            uuid.New(),
		PullRequestID: pr.ID,
		ReviewerID:    actor.UserID,
		State:         state,
		Body:          req.Body,
		CommitSHA:     pr.HeadSHA,
		SubmittedAt:   &now,
		CreatedAt:     now,
	}

	if err := h.reviewRepo.Create(c.Request().Context(), review); err != nil {
		return err
	}

	ctx := c.Request().Context()
	for _, comment := range req.Comments {
		reviewID := review.ID
		rc := &entity.ReviewComment{
			ID:            uuid.New(),
			PullRequestID: pr.ID,
			AuthorID:      actor.UserID,
			ReviewID:      &reviewID,
			Path:          comment.Path,
			Body:          comment.Body,
			Line:          comment.Line,
			Side:          entity.SideRight,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := h.reviewCommentRepo.Create(ctx, rc); err != nil {
			return err
		}
	}

	return c.JSON(http.StatusOK, toReviewResponse(review))
}

func (h *PullRequestHandler) ListReviews(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pull request number")
	}

	pr, err := h.prRepo.GetByNumber(c.Request().Context(), repo.ID, number)
	if err != nil {
		return err
	}

	reviews, err := h.reviewRepo.ListByPR(c.Request().Context(), pr.ID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, toReviewResponses(reviews))
}

func (h *PullRequestHandler) ListReviewComments(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pull request number")
	}

	pr, err := h.prRepo.GetByNumber(c.Request().Context(), repo.ID, number)
	if err != nil {
		return err
	}

	comments, err := h.reviewCommentRepo.ListByPR(c.Request().Context(), pr.ID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, toReviewCommentResponses(comments))
}

func (h *PullRequestHandler) CreateReviewComment(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pull request number")
	}

	pr, err := h.prRepo.GetByNumber(c.Request().Context(), repo.ID, number)
	if err != nil {
		return err
	}

	var req createReviewCommentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	side := req.Side
	if side == "" {
		side = entity.SideRight
	}

	now := time.Now().UTC()
	comment := &entity.ReviewComment{
		ID:            uuid.New(),
		PullRequestID: pr.ID,
		AuthorID:      actor.UserID,
		Path:          req.Path,
		Body:          req.Body,
		Line:          req.Line,
		Side:          side,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := h.reviewCommentRepo.Create(c.Request().Context(), comment); err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, toReviewCommentResponse(comment))
}

func toPullRequestResponse(pr *entity.PullRequest, owner, repoName, host string) pullRequestResponse {
	resp := pullRequestResponse{
		ID:      pr.ID,
		Number:  pr.Number,
		Title:   pr.Title,
		Body:    pr.Body,
		HeadRef: pr.HeadRef,
		BaseRef: pr.BaseRef,
		State:   pr.State,
		NodeID:  PRNodeID(pr.ID),
		HTMLURL: "https://" + host + "/" + owner + "/" + repoName + "/pull/" + strconv.Itoa(pr.Number),
	}
	if pr.MergedAt != nil {
		formatted := pr.MergedAt.UTC().Format("2006-01-02T15:04:05Z")
		resp.MergedAt = &formatted
	}
	return resp
}

func toPullRequestResponses(pulls []*entity.PullRequest, owner, repoName, host string) []pullRequestResponse {
	result := make([]pullRequestResponse, 0, len(pulls))
	for _, pr := range pulls {
		result = append(result, toPullRequestResponse(pr, owner, repoName, host))
	}
	return result
}

func toReviewResponse(review *entity.Review) reviewResponse {
	resp := reviewResponse{
		ID:            review.ID,
		PullRequestID: review.PullRequestID,
		User:          review.ReviewerID,
		State:         review.State,
		Body:          review.Body,
		CommitSHA:     review.CommitSHA,
	}
	if review.SubmittedAt != nil {
		resp.SubmittedAt = review.SubmittedAt.UTC().Format(time.RFC3339)
	}
	return resp
}

func toReviewResponses(reviews []*entity.Review) []reviewResponse {
	result := make([]reviewResponse, 0, len(reviews))
	for _, review := range reviews {
		result = append(result, toReviewResponse(review))
	}
	return result
}

func toReviewCommentResponse(comment *entity.ReviewComment) reviewCommentResponse {
	return reviewCommentResponse{
		ID:            comment.ID,
		PullRequestID: comment.PullRequestID,
		ReviewID:      comment.ReviewID,
		User:          comment.AuthorID,
		Path:          comment.Path,
		Body:          comment.Body,
		Line:          comment.Line,
		Side:          comment.Side,
		CreatedAt:     comment.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     comment.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toReviewCommentResponses(comments []*entity.ReviewComment) []reviewCommentResponse {
	result := make([]reviewCommentResponse, 0, len(comments))
	for _, comment := range comments {
		result = append(result, toReviewCommentResponse(comment))
	}
	return result
}
