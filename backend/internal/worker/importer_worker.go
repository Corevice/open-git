package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

const (
	TypeGitHubImport = "import:github"

	githubAPIBase = "https://api.github.com"
	gitStorageEnv = "GIT_STORAGE_PATH"

	rateLimitRemainingHeader = "X-RateLimit-Remaining"
	rateLimitResetHeader     = "X-RateLimit-Reset"

	rateLimitThreshold = 10

	// maxRateLimitWait bounds how long the worker will block inline on a
	// rate-limit reset, protecting against absurd/malicious upstream reset
	// headers. Beyond this, the caller re-fetches and re-evaluates.
	maxRateLimitWait = 30 * time.Second

	defaultPageSize = 100

	importerHTTPTimeout = 30 * time.Second
)

var ErrRateLimitExceeded = errors.New("github rate limit exceeded")

type GitHubImportPayload struct {
	ImportJobID     string `json:"import_job_id"`
	OrganizationID  string `json:"organization_id"`
	SourceOwner     string `json:"source_owner"`
	SourceRepo      string `json:"source_repo"`
	Token           string `json:"token"`
	ResumeFromPhase string `json:"resume_from_phase"`
}

type CloneFunc func(ctx context.Context, repoURL, token, tmpDir, destPath string) error

var (
	githubIdentPattern = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9._-]*[a-zA-Z0-9])?$`)
	ansiEscapePattern  = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	ctrlCharPattern    = regexp.MustCompile(`[\x00-\x1f\x7f-\x9f]`)
)

type ImporterWorker struct {
	importJobs    domainrepo.IImportJobRepository
	userMappings  domainrepo.IImportUserMappingRepository
	checkpoints   domainrepo.IImportPhaseCheckpointRepository
	issues        domainrepo.IIssueRepository
	labels        domainrepo.ILabelRepository
	milestones    domainrepo.IMilestoneRepository
	comments      domainrepo.ICommentRepository
	pullRequests  domainrepo.IPullRequestRepository
	repositories  domainrepo.IRepositoryRepository
	users         domainrepo.IUserRepository
	httpClient    *http.Client
	apiBase       string
	gitStoragePath string
	logger        *slog.Logger
	now           func() time.Time
	cloneFn       CloneFunc
}

func NewImporterWorker(
	importJobs domainrepo.IImportJobRepository,
	userMappings domainrepo.IImportUserMappingRepository,
	checkpoints domainrepo.IImportPhaseCheckpointRepository,
	issues domainrepo.IIssueRepository,
	labels domainrepo.ILabelRepository,
	milestones domainrepo.IMilestoneRepository,
	comments domainrepo.ICommentRepository,
	pullRequests domainrepo.IPullRequestRepository,
	repositories domainrepo.IRepositoryRepository,
	users domainrepo.IUserRepository,
) *ImporterWorker {
	return &ImporterWorker{
		importJobs:     importJobs,
		userMappings:   userMappings,
		checkpoints:    checkpoints,
		issues:         issues,
		labels:         labels,
		milestones:     milestones,
		comments:       comments,
		pullRequests:   pullRequests,
		repositories:   repositories,
		users:          users,
		httpClient:     &http.Client{Timeout: importerHTTPTimeout},
		apiBase:        githubAPIBase,
		gitStoragePath: os.Getenv(gitStorageEnv),
		logger:         slog.Default(),
		now:            func() time.Time { return time.Now().UTC() },
	}
}

func (w *ImporterWorker) WithAPIBase(base string) *ImporterWorker {
	w.apiBase = strings.TrimRight(base, "/")
	return w
}

func (w *ImporterWorker) WithHTTPClient(c *http.Client) *ImporterWorker {
	w.httpClient = c
	return w
}

func (w *ImporterWorker) WithGitStoragePath(path string) *ImporterWorker {
	w.gitStoragePath = path
	return w
}

func (w *ImporterWorker) WithLogger(logger *slog.Logger) *ImporterWorker {
	if logger != nil {
		w.logger = logger
	}
	return w
}

func (w *ImporterWorker) WithCloneFn(fn CloneFunc) *ImporterWorker {
	w.cloneFn = fn
	return w
}

func (w *ImporterWorker) HandleGitHubImport(ctx context.Context, task *asynq.Task) error {
	var payload GitHubImportPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal importer payload: %w: %w", err, asynq.SkipRetry)
	}
	if payload.ImportJobID == "" {
		return fmt.Errorf("importer payload missing import job id: %w", asynq.SkipRetry)
	}
	if payload.SourceOwner == "" || payload.SourceRepo == "" {
		return fmt.Errorf("importer payload missing source repo: %w", asynq.SkipRetry)
	}
	if err := validateGitHubIdent(payload.SourceOwner); err != nil {
		return fmt.Errorf("invalid source owner: %w: %w", err, asynq.SkipRetry)
	}
	if err := validateGitHubIdent(payload.SourceRepo); err != nil {
		return fmt.Errorf("invalid source repo: %w: %w", err, asynq.SkipRetry)
	}

	jobID, err := uuid.Parse(payload.ImportJobID)
	if err != nil {
		return fmt.Errorf("invalid import job id: %w: %w", err, asynq.SkipRetry)
	}
	orgID, err := uuid.Parse(payload.OrganizationID)
	if err != nil {
		return fmt.Errorf("invalid organization id: %w: %w", err, asynq.SkipRetry)
	}

	job, err := w.importJobs.GetByIDAndOrg(ctx, jobID, orgID)
	if err != nil {
		return fmt.Errorf("load import job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("import job not found: %w", asynq.SkipRetry)
	}

	if err := w.importJobs.UpdateStatus(ctx, jobID, entity.ImportJobStatusRunning); err != nil {
		return fmt.Errorf("mark job running: %w", err)
	}

	ownerLogin, err := w.resolveOwnerLogin(ctx, job)
	if err != nil {
		return w.failImport(ctx, jobID, entity.ImportJobPhaseClone, err)
	}

	fail := func(phase entity.ImportJobPhase, err error) error {
		return w.failImport(ctx, jobID, phase, err)
	}

	var (
		repoID  uuid.UUID
		gitPath string
	)

	if err := w.runPhase(ctx, jobID, entity.ImportJobPhaseClone, payload.ResumeFromPhase, func(_ *entity.ImportPhaseCheckpoint) error {
		if err := w.importJobs.UpdatePhase(ctx, jobID, entity.ImportJobPhaseClone); err != nil {
			return err
		}
		return w.ingestClone(ctx, jobID, orgID, job.TargetName, ownerLogin, payload.SourceOwner, payload.SourceRepo, payload.Token)
	}); err != nil {
		return fail(entity.ImportJobPhaseClone, err)
	}

	if err := w.runPhase(ctx, jobID, entity.ImportJobPhaseMetadata, payload.ResumeFromPhase, func(_ *entity.ImportPhaseCheckpoint) error {
		if err := w.importJobs.UpdatePhase(ctx, jobID, entity.ImportJobPhaseMetadata); err != nil {
			return err
		}
		var ingestErr error
		repoID, gitPath, ingestErr = w.ingestMetadata(ctx, jobID, orgID, job, payload.SourceOwner, payload.SourceRepo, payload.Token, ownerLogin)
		return ingestErr
	}); err != nil {
		return fail(entity.ImportJobPhaseMetadata, err)
	}

	if repoID == uuid.Nil || gitPath == "" {
		var resolveErr error
		resolvedID, resolvedPath, resolveErr := w.resolveTargetRepo(ctx, job)
		if resolveErr != nil {
			return fail(entity.ImportJobPhaseMetadata, resolveErr)
		}
		if repoID == uuid.Nil {
			repoID = resolvedID
		}
		if gitPath == "" {
			gitPath = resolvedPath
		}
	}

	if err := w.runPhase(ctx, jobID, entity.ImportJobPhaseIssues, payload.ResumeFromPhase, func(cp *entity.ImportPhaseCheckpoint) error {
		if err := w.importJobs.UpdatePhase(ctx, jobID, entity.ImportJobPhaseIssues); err != nil {
			return err
		}
		return w.ingestIssues(ctx, jobID, orgID, repoID, payload.SourceOwner, payload.SourceRepo, payload.Token, job.CreatedBy, cp)
	}); err != nil {
		return fail(entity.ImportJobPhaseIssues, err)
	}

	if err := w.runPhase(ctx, jobID, entity.ImportJobPhasePullRequests, payload.ResumeFromPhase, func(cp *entity.ImportPhaseCheckpoint) error {
		if err := w.importJobs.UpdatePhase(ctx, jobID, entity.ImportJobPhasePullRequests); err != nil {
			return err
		}
		return w.ingestPullRequests(ctx, jobID, orgID, repoID, gitPath, payload.SourceOwner, payload.SourceRepo, payload.Token, job.CreatedBy, cp)
	}); err != nil {
		return fail(entity.ImportJobPhasePullRequests, err)
	}

	if err := w.importJobs.UpdatePhase(ctx, jobID, entity.ImportJobPhaseDone); err != nil {
		return fail(entity.ImportJobPhaseDone, err)
	}
	if err := w.importJobs.UpdateStatus(ctx, jobID, entity.ImportJobStatusCompleted); err != nil {
		return fail(entity.ImportJobPhaseDone, err)
	}
	return nil
}

func (w *ImporterWorker) failImport(ctx context.Context, jobID uuid.UUID, phase entity.ImportJobPhase, err error) error {
	w.logger.Error("import failed", "job_id", jobID.String(), "phase", string(phase), "error", sanitizeLog(err.Error()))
	_ = w.importJobs.SetError(ctx, jobID, fmt.Sprintf("import failed during %s phase", phase))
	_ = w.importJobs.UpdateStatus(ctx, jobID, entity.ImportJobStatusFailed)
	return err
}


func phaseIndex(phase entity.ImportJobPhase) int {
	switch phase {
	case entity.ImportJobPhaseClone:
		return 0
	case entity.ImportJobPhaseMetadata:
		return 1
	case entity.ImportJobPhaseIssues:
		return 2
	case entity.ImportJobPhasePullRequests:
		return 3
	case entity.ImportJobPhaseDone:
		return 4
	default:
		return 0
	}
}

func (w *ImporterWorker) runPhase(ctx context.Context, jobID uuid.UUID, phase entity.ImportJobPhase, resumeFrom string, fn func(*entity.ImportPhaseCheckpoint) error) error {
	if resumeFrom != "" && phaseIndex(phase) < phaseIndex(entity.ImportJobPhase(resumeFrom)) {
		return nil
	}
	cp, err := w.checkpoints.GetCheckpoint(ctx, jobID, phase)
	if err != nil {
		return err
	}
	if cp != nil && cp.Completed {
		return nil
	}
	if err := fn(cp); err != nil {
		return err
	}
	return w.checkpoints.MarkPhaseComplete(ctx, jobID, phase)
}

func (w *ImporterWorker) mapUser(ctx context.Context, jobID uuid.UUID, githubLogin, displayName string, fallbackAuthorID uuid.UUID) uuid.UUID {
	if githubLogin == "" {
		return fallbackAuthorID
	}

	existing, err := w.userMappings.GetMappingByLogin(ctx, jobID, githubLogin)
	if err != nil {
		w.logger.Warn("failed to load user mapping", "job_id", jobID.String(), "login", githubLogin, "error", sanitizeLog(err.Error()))
		return fallbackAuthorID
	}
	if existing != nil {
		if existing.LocalUserID != nil {
			return *existing.LocalUserID
		}
		return fallbackAuthorID
	}

	mapping := &entity.ImportUserMapping{
		ImportJobID:       jobID,
		GitHubLogin:       githubLogin,
		GitHubDisplayName: displayName,
	}

	localUser, lookupErr := w.users.GetByLogin(ctx, githubLogin)
	if lookupErr != nil {
		w.logger.Warn("failed to lookup local user", "job_id", jobID.String(), "login", githubLogin, "error", sanitizeLog(lookupErr.Error()))
	}
	if lookupErr == nil && localUser != nil {
		mapping.LocalUserID = &localUser.ID
		if upsertErr := w.userMappings.UpsertMapping(ctx, mapping); upsertErr != nil {
			w.logger.Warn("failed to save user mapping", "job_id", jobID.String(), "login", githubLogin, "error", sanitizeLog(upsertErr.Error()))
		}
		return localUser.ID
	}

	if upsertErr := w.userMappings.UpsertMapping(ctx, mapping); upsertErr != nil {
		w.logger.Warn("failed to save ghost user mapping", "job_id", jobID.String(), "login", githubLogin, "error", sanitizeLog(upsertErr.Error()))
	}
	return fallbackAuthorID
}

func (w *ImporterWorker) ingestMetadata(
	ctx context.Context,
	jobID, orgID uuid.UUID,
	job *entity.ImportJob,
	owner, repo, token, ownerLogin string,
) (uuid.UUID, string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", w.apiBase, owner, repo)
	_, ghRepo, err := w.fetchObject(ctx, url, token, jobID)
	if err != nil {
		return uuid.Nil, "", err
	}

	description := ghString(ghRepo, "description")
	defaultBranch := ghString(ghRepo, "default_branch")
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	visibility := entity.VisibilityPublic
	if ghBool(ghRepo, "private") {
		visibility = entity.VisibilityPrivate
	}

	gitPath, err := w.repoGitPath(ownerLogin, job.TargetName)
	if err != nil {
		return uuid.Nil, "", err
	}
	repository := &entity.Repository{
		OrganizationID: orgID,
		OwnerID:        job.CreatedBy,
		Name:           job.TargetName,
		Description:    description,
		GitPath:        gitPath,
		OwnerLogin:     ownerLogin,
		Visibility:     visibility,
		DefaultBranch:  defaultBranch,
		CreatedAt:      w.now(),
	}

	if err := w.repositories.Create(ctx, repository); err != nil {
		return uuid.Nil, "", fmt.Errorf("create repository: %w", err)
	}
	if err := w.importJobs.SetTargetRepository(ctx, jobID, repository.ID); err != nil {
		return uuid.Nil, "", fmt.Errorf("set target repository: %w", err)
	}
	return repository.ID, gitPath, nil
}

func (w *ImporterWorker) ingestIssues(
	ctx context.Context,
	jobID, orgID, repoID uuid.UUID,
	owner, repo, token string,
	fallbackAuthorID uuid.UUID,
	checkpoint *entity.ImportPhaseCheckpoint,
) error {
	labelCache, err := w.preloadLabelCache(ctx, repoID)
	if err != nil {
		return fmt.Errorf("preload labels: %w", err)
	}
	milestoneCache, err := w.preloadMilestoneCache(ctx, repoID)
	if err != nil {
		return fmt.Errorf("preload milestones: %w", err)
	}

	page := resumePageFromCheckpoint(checkpoint)

	for {
		url := fmt.Sprintf("%s/repos/%s/%s/issues?state=all&page=%d&per_page=%d", w.apiBase, owner, repo, page, defaultPageSize)
		_, items, err := w.fetchList(ctx, url, token, jobID)
		if err != nil {
			return fmt.Errorf("fetch issues page %d: %w", page, err)
		}
		if len(items) == 0 {
			break
		}

		for _, item := range items {
			if _, isPR := item["pull_request"]; isPR {
				continue
			}
			if err := w.importIssue(ctx, jobID, orgID, repoID, owner, repo, token, fallbackAuthorID, item, labelCache, milestoneCache); err != nil {
				w.logger.Warn("skipped issue", "job_id", jobID.String(), "error", sanitizeLog(err.Error()))
			}
		}

		cursor := strconv.Itoa(page)
		if err := w.checkpoints.SaveCheckpoint(ctx, &entity.ImportPhaseCheckpoint{
			ImportJobID: jobID,
			Phase:       entity.ImportJobPhaseIssues,
			LastCursor:  cursor,
		}); err != nil {
			return err
		}

		if len(items) < defaultPageSize {
			break
		}
		page++
	}
	return nil
}

func (w *ImporterWorker) importIssue(
	ctx context.Context,
	jobID, orgID, repoID uuid.UUID,
	owner, repo, token string,
	fallbackAuthorID uuid.UUID,
	item map[string]any,
	labelCache map[string]uuid.UUID,
	milestoneCache map[int]uuid.UUID,
) error {
	number := ghInt(item, "number")
	title := ghString(item, "title")
	body := ghString(item, "body")
	state := ghString(item, "state")
	if state == "" {
		state = "open"
	}

	authorLogin, authorName := ghUser(item, "user")
	authorID := w.mapUser(ctx, jobID, authorLogin, authorName, fallbackAuthorID)

	issue := &entity.Issue{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Number:         number,
		Title:          title,
		Body:           body,
		State:          state,
		AuthorID:       authorID,
		AuthorLogin:    authorLogin,
		CreatedAt:      ghTime(item, "created_at"),
		UpdatedAt:      ghTime(item, "updated_at"),
	}
	if closedAt := ghTimePtr(item, "closed_at"); closedAt != nil {
		issue.ClosedAt = closedAt
	}

	if milestoneRaw, ok := item["milestone"].(map[string]any); ok && milestoneRaw != nil {
		milestoneID, err := w.ensureMilestone(ctx, orgID, repoID, milestoneRaw, milestoneCache)
		if err != nil {
			return err
		}
		if milestoneID != uuid.Nil {
			issue.MilestoneID = &milestoneID
		}
	}

	if err := w.issues.Create(ctx, issue); err != nil {
		return err
	}

	if labels, ok := item["labels"].([]any); ok {
		for _, raw := range labels {
			labelMap, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			labelID, err := w.ensureLabel(ctx, orgID, repoID, labelMap, labelCache)
			if err != nil {
				return err
			}
			if labelID != uuid.Nil {
				_ = w.labels.AddToIssue(ctx, repoID, number, labelID)
			}
		}
	}

	if err := w.importIssueComments(ctx, jobID, orgID, issue.ID, owner, repo, token, fallbackAuthorID, number); err != nil {
		w.logger.Warn("failed to import issue comments", "job_id", jobID.String(), "issue_number", number, "error", sanitizeLog(err.Error()))
	}
	return nil
}

func (w *ImporterWorker) importIssueComments(
	ctx context.Context,
	jobID, orgID, issueID uuid.UUID,
	owner, repo, token string,
	fallbackAuthorID uuid.UUID,
	issueNumber int,
) error {
	page := 1
	for {
		url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments?page=%d&per_page=%d", w.apiBase, owner, repo, issueNumber, page, defaultPageSize)
		_, items, err := w.fetchList(ctx, url, token, jobID)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}

		for _, item := range items {
			login, displayName := ghUser(item, "user")
			authorID := w.mapUser(ctx, jobID, login, displayName, fallbackAuthorID)
			comment := &entity.Comment{
				IssueID:        issueID,
				OrganizationID: orgID,
				AuthorID:       authorID,
				AuthorLogin:    login,
				Body:           ghString(item, "body"),
				CreatedAt:      ghTime(item, "created_at"),
				UpdatedAt:      ghTime(item, "updated_at"),
			}
			if err := w.comments.Create(ctx, comment); err != nil {
				return err
			}
		}

		if len(items) < defaultPageSize {
			return nil
		}
		page++
	}
}

func (w *ImporterWorker) preloadLabelCache(ctx context.Context, repoID uuid.UUID) (map[string]uuid.UUID, error) {
	cache := make(map[string]uuid.UUID)
	const pageSize = 100
	for page := 1; ; page++ {
		labels, total, err := w.labels.ListByRepo(ctx, repoID, page, pageSize)
		if err != nil {
			return nil, err
		}
		for _, label := range labels {
			cache[label.Name] = label.ID
		}
		if page*pageSize >= total {
			break
		}
	}
	return cache, nil
}

func (w *ImporterWorker) preloadMilestoneCache(ctx context.Context, repoID uuid.UUID) (map[int]uuid.UUID, error) {
	cache := make(map[int]uuid.UUID)
	const pageSize = 100
	for page := 1; ; page++ {
		milestones, total, err := w.milestones.ListByRepo(ctx, repoID, "", page, pageSize)
		if err != nil {
			return nil, err
		}
		for _, milestone := range milestones {
			cache[milestone.Number] = milestone.ID
		}
		if page*pageSize >= total {
			break
		}
	}
	return cache, nil
}

func (w *ImporterWorker) ensureLabel(ctx context.Context, orgID, repoID uuid.UUID, raw map[string]any, cache map[string]uuid.UUID) (uuid.UUID, error) {
	name := ghString(raw, "name")
	if name == "" {
		return uuid.Nil, nil
	}
	if cache != nil {
		if id, ok := cache[name]; ok {
			return id, nil
		}
	}
	existing, err := w.labels.GetByName(ctx, repoID, name)
	if err == nil && existing != nil {
		if cache != nil {
			cache[name] = existing.ID
		}
		return existing.ID, nil
	}
	color := strings.TrimPrefix(ghString(raw, "color"), "#")
	if len(color) != 6 {
		color = "ededed"
	}
	label := &entity.Label{
		RepositoryID:   repoID,
		OrganizationID: orgID,
		Name:           name,
		Color:          color,
		Description:    ghString(raw, "description"),
		CreatedAt:      w.now(),
	}
	if err := w.labels.Create(ctx, label); err != nil {
		return uuid.Nil, err
	}
	if cache != nil {
		cache[name] = label.ID
	}
	return label.ID, nil
}

func (w *ImporterWorker) ensureMilestone(ctx context.Context, orgID, repoID uuid.UUID, raw map[string]any, cache map[int]uuid.UUID) (uuid.UUID, error) {
	number := ghInt(raw, "number")
	if number == 0 {
		return uuid.Nil, nil
	}
	if cache != nil {
		if id, ok := cache[number]; ok {
			return id, nil
		}
	}
	existing, err := w.milestones.GetByNumber(ctx, repoID, number)
	if err == nil && existing != nil {
		if cache != nil {
			cache[number] = existing.ID
		}
		return existing.ID, nil
	}
	state := ghString(raw, "state")
	if state == "" {
		state = "open"
	}
	milestone := &entity.Milestone{
		RepositoryID:   repoID,
		OrganizationID: orgID,
		Number:         number,
		Title:          ghString(raw, "title"),
		Description:    ghString(raw, "description"),
		State:          state,
		DueOn:          ghTimePtr(raw, "due_on"),
		CreatedAt:      ghTime(raw, "created_at"),
		UpdatedAt:      ghTime(raw, "updated_at"),
	}
	if err := w.milestones.Create(ctx, milestone); err != nil {
		return uuid.Nil, err
	}
	if cache != nil {
		cache[number] = milestone.ID
	}
	return milestone.ID, nil
}

func (w *ImporterWorker) ingestPullRequests(
	ctx context.Context,
	jobID, orgID, repoID uuid.UUID,
	gitPath, owner, repo, token string,
	fallbackAuthorID uuid.UUID,
	checkpoint *entity.ImportPhaseCheckpoint,
) error {
	page := resumePageFromCheckpoint(checkpoint)

	for {
		url := fmt.Sprintf("%s/repos/%s/%s/pulls?state=all&page=%d&per_page=%d", w.apiBase, owner, repo, page, defaultPageSize)
		_, items, err := w.fetchList(ctx, url, token, jobID)
		if err != nil {
			return fmt.Errorf("fetch pull requests page %d: %w", page, err)
		}
		if len(items) == 0 {
			break
		}

		for _, item := range items {
			if err := w.importPullRequest(ctx, jobID, orgID, repoID, gitPath, fallbackAuthorID, item); err != nil {
				w.logger.Warn("skipped pull request", "job_id", jobID.String(), "error", sanitizeLog(err.Error()))
			}
		}

		cursor := strconv.Itoa(page)
		if err := w.checkpoints.SaveCheckpoint(ctx, &entity.ImportPhaseCheckpoint{
			ImportJobID: jobID,
			Phase:       entity.ImportJobPhasePullRequests,
			LastCursor:  cursor,
		}); err != nil {
			return err
		}

		if len(items) < defaultPageSize {
			break
		}
		page++
	}
	return nil
}

func (w *ImporterWorker) importPullRequest(
	ctx context.Context,
	jobID, orgID, repoID uuid.UUID,
	gitPath string,
	fallbackAuthorID uuid.UUID,
	item map[string]any,
) error {
	headRef := ghNestedString(item, "head", "ref")
	baseRef := ghNestedString(item, "base", "ref")
	headSHA := ghNestedString(item, "head", "sha")
	baseSHA := ghNestedString(item, "base", "sha")

	login, displayName := ghUser(item, "user")
	authorID := w.mapUser(ctx, jobID, login, displayName, fallbackAuthorID)

	mergedByLogin, mergedByName := ghUser(item, "merged_by")
	var mergedByID *uuid.UUID
	if mergedByLogin != "" {
		id := w.mapUser(ctx, jobID, mergedByLogin, mergedByName, fallbackAuthorID)
		mergedByID = &id
	}

	state := ghString(item, "state")
	if state == "" {
		state = entity.PullRequestStateOpen
	}
	if ghBool(item, "merged") {
		state = entity.PullRequestStateMerged
	}

	pr := &entity.PullRequest{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Number:         ghInt(item, "number"),
		Title:          ghString(item, "title"),
		Body:           ghString(item, "body"),
		Draft:          ghBool(item, "draft"),
		HeadRef:        headRef,
		BaseRef:        baseRef,
		HeadSHA:        headSHA,
		BaseSHA:        baseSHA,
		State:          state,
		AuthorID:       authorID,
		CreatedAt:      ghTime(item, "created_at"),
		UpdatedAt:      ghTime(item, "updated_at"),
	}

	headExists := headRef != "" && w.branchExists(gitPath, headRef)
	if ghBool(item, "merged") && headExists {
		pr.State = entity.PullRequestStateMerged
		if mergedAt := ghTimePtr(item, "merged_at"); mergedAt != nil {
			pr.MergedAt = mergedAt
		}
		pr.MergeCommitSHA = ghString(item, "merge_commit_sha")
		pr.MergedBy = mergedByID
	} else if ghBool(item, "merged") && !headExists {
		pr.State = entity.PullRequestStateClosed
		pr.MergedAt = nil
	}

	if err := w.pullRequests.Create(ctx, pr); err != nil {
		return err
	}

	if pr.State == entity.PullRequestStateMerged && headExists && pr.MergedAt != nil {
		mergedBy := authorID
		if pr.MergedBy != nil {
			mergedBy = *pr.MergedBy
		}
		_ = w.pullRequests.SetMerged(ctx, pr.ID, *pr.MergedAt, mergedBy, pr.MergeCommitSHA)
	}
	return nil
}

func githubRepoCloneURL(owner, repo string) string {
	return (&url.URL{
		Scheme: "https",
		Host:   "github.com",
		Path:   path.Join(owner, repo+".git"),
	}).String()
}

func (w *ImporterWorker) ingestClone(
	ctx context.Context,
	jobID, orgID uuid.UUID,
	targetName, ownerLogin, sourceOwner, sourceRepo, token string,
) error {
	if w.gitStoragePath == "" {
		return fmt.Errorf("%s is not configured", gitStorageEnv)
	}

	tmpDir, err := os.MkdirTemp("", "open-git-import-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	destPath, err := w.repoGitPath(ownerLogin, targetName)
	if err != nil {
		return err
	}
	repoURL := githubRepoCloneURL(sourceOwner, sourceRepo)

	if w.cloneFn != nil {
		return w.cloneFn(ctx, repoURL, token, tmpDir, destPath)
	}

	auth := &githttp.BasicAuth{Username: "x-access-token", Password: token}
	if _, err := gogit.PlainClone(tmpDir, true, &gogit.CloneOptions{URL: repoURL, Auth: auth}); err != nil {
		return fmt.Errorf("clone source repository: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}

	absTmp, err := filepath.Abs(tmpDir)
	if err != nil {
		return fmt.Errorf("resolve temp dir: %w", err)
	}

	if _, err := gogit.PlainClone(destPath, true, &gogit.CloneOptions{
		URL:    "file://" + filepath.ToSlash(absTmp),
		Mirror: true,
	}); err != nil {
		return fmt.Errorf("mirror to local storage: %w", err)
	}

	w.logger.Info("clone complete", "job_id", jobID.String(), "org_id", orgID.String(), "dest", destPath)
	return nil
}

func (w *ImporterWorker) branchExists(gitPath, branch string) bool {
	if gitPath == "" || branch == "" {
		return false
	}
	repo, err := gogit.PlainOpen(gitPath)
	if err != nil {
		return false
	}
	_, err = repo.Reference(plumbing.NewBranchReferenceName(branch), true)
	return err == nil
}

func (w *ImporterWorker) repoGitPath(ownerLogin, repoName string) (string, error) {
	if err := validateGitHubIdent(ownerLogin); err != nil {
		return "", fmt.Errorf("invalid owner login: %w", err)
	}
	if err := validateGitHubIdent(repoName); err != nil {
		return "", fmt.Errorf("invalid repository name: %w", err)
	}

	root := filepath.Clean(w.gitStoragePath)
	dest := filepath.Clean(filepath.Join(root, ownerLogin, repoName+".git"))

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve storage root: %w", err)
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return "", fmt.Errorf("resolve repository path: %w", err)
	}

	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("resolve storage root symlinks: %w", err)
		}
		realRoot = absRoot
	}

	parent := filepath.Dir(absDest)
	realParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("resolve repository parent symlinks: %w", err)
		}
		realParent = parent
	}
	absDest = filepath.Join(realParent, filepath.Base(absDest))

	rel, err := filepath.Rel(realRoot, absDest)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("repository path escapes storage root")
	}
	return absDest, nil
}

func (w *ImporterWorker) resolveOwnerLogin(ctx context.Context, job *entity.ImportJob) (string, error) {
	user, err := w.users.GetByID(ctx, job.CreatedBy)
	if err != nil {
		return "", fmt.Errorf("resolve owner login: %w", err)
	}
	if user == nil || user.Login == "" {
		return "", fmt.Errorf("owner login not found for job creator")
	}
	return user.Login, nil
}

func (w *ImporterWorker) resolveTargetRepo(ctx context.Context, job *entity.ImportJob) (uuid.UUID, string, error) {
	repo, err := w.repositories.GetByOwnerAndName(ctx, job.CreatedBy, job.TargetName)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("load target repository: %w", err)
	}
	if repo == nil {
		return uuid.Nil, "", fmt.Errorf("target repository %q not found", job.TargetName)
	}
	return repo.ID, repo.GitPath, nil
}

func (w *ImporterWorker) fetchObject(ctx context.Context, url, token string, jobID uuid.UUID) (*http.Response, map[string]any, error) {
	for {
		resp, body, err := w.fetchPage(ctx, url, token)
		if err != nil {
			return resp, nil, err
		}
		retry, err := w.handleRateLimit(ctx, resp, jobID)
		if err != nil {
			return resp, nil, err
		}
		if retry {
			continue
		}
		obj, err := decodeSingleObject(body)
		if err != nil {
			return resp, nil, fmt.Errorf("decode body: %w", err)
		}
		return resp, obj, nil
	}
}

func (w *ImporterWorker) fetchList(ctx context.Context, url, token string, jobID uuid.UUID) (*http.Response, []map[string]any, error) {
	for {
		resp, body, err := w.fetchPage(ctx, url, token)
		if err != nil {
			return resp, nil, err
		}
		retry, err := w.handleRateLimit(ctx, resp, jobID)
		if err != nil {
			return resp, nil, err
		}
		if retry {
			continue
		}
		items, err := decodeItems(body)
		if err != nil {
			return resp, nil, fmt.Errorf("decode body: %w", err)
		}
		return resp, items, nil
	}
}

func (w *ImporterWorker) handleRateLimit(ctx context.Context, resp *http.Response, jobID uuid.UUID) (bool, error) {
	if err := w.throttleIfLowRemaining(ctx, resp); err != nil {
		return false, err
	}

	remainingStr := resp.Header.Get(rateLimitRemainingHeader)
	if remainingStr == "" {
		return false, nil
	}
	remaining, err := strconv.Atoi(remainingStr)
	if err != nil || remaining > 0 {
		return false, nil
	}

	resetStr := resp.Header.Get(rateLimitResetHeader)
	resetTs, parseErr := strconv.ParseInt(resetStr, 10, 64)
	if parseErr != nil || resetTs <= 0 {
		return false, fmt.Errorf("rate limit exceeded without valid reset header: %w", ErrRateLimitExceeded)
	}

	wait := time.Until(time.Unix(resetTs, 0))
	// Bound the inline wait: the reset header is untrusted upstream input, and
	// an absurd or malicious value (far-future timestamp) must not block the
	// worker indefinitely. Cap it; the caller re-fetches and re-evaluates.
	if wait > maxRateLimitWait {
		wait = maxRateLimitWait
	}
	if wait > 0 {
		w.logger.Info("rate limit reached, sleeping", "job_id", jobID.String(), "wait_seconds", wait.Seconds())
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-timer.C:
		}
	}
	return true, nil
}

func (w *ImporterWorker) fetchPage(ctx context.Context, url, token string) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		return nil, nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp, nil, fmt.Errorf("non-2xx response: %d body=%s", resp.StatusCode, sanitizeLog(string(body)))
	}

	return resp, body, nil
}

func decodeSingleObject(body []byte) (map[string]any, error) {
	trim := strings.TrimSpace(string(body))
	if len(trim) == 0 {
		return nil, fmt.Errorf("empty response body")
	}
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func decodeItems(body []byte) ([]map[string]any, error) {
	trim := strings.TrimSpace(string(body))
	if len(trim) == 0 {
		return nil, nil
	}
	if trim[0] == '[' {
		var arr []map[string]any
		if err := json.Unmarshal(body, &arr); err != nil {
			return nil, err
		}
		return arr, nil
	}
	var single map[string]any
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, err
	}
	return []map[string]any{single}, nil
}

func (w *ImporterWorker) throttleIfLowRemaining(ctx context.Context, resp *http.Response) error {
	if resp == nil {
		return nil
	}
	remainingStr := resp.Header.Get(rateLimitRemainingHeader)
	if remainingStr == "" {
		return nil
	}
	remaining, err := strconv.Atoi(remainingStr)
	if err != nil || remaining <= 0 || remaining >= rateLimitThreshold {
		return nil
	}
	return w.sleepUntilRateLimitReset(ctx, resp)
}

func (w *ImporterWorker) sleepUntilRateLimitReset(ctx context.Context, resp *http.Response) error {
	resetStr := resp.Header.Get(rateLimitResetHeader)
	resetTs, err := strconv.ParseInt(resetStr, 10, 64)
	if err != nil || resetTs <= 0 {
		return nil
	}
	wait := time.Until(time.Unix(resetTs, 0))
	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func resumePageFromCheckpoint(checkpoint *entity.ImportPhaseCheckpoint) int {
	page := 1
	if checkpoint != nil && checkpoint.LastCursor != "" {
		if p, err := strconv.Atoi(checkpoint.LastCursor); err == nil && p > 0 {
			page = p + 1
		}
	}
	return page
}

func ghString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

func ghNestedString(m map[string]any, outer, inner string) string {
	if m == nil {
		return ""
	}
	raw, ok := m[outer].(map[string]any)
	if !ok || raw == nil {
		return ""
	}
	return ghString(raw, inner)
}

func ghInt(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(t)
		return n
	default:
		return 0
	}
}

func ghBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}

func ghUser(m map[string]any, key string) (login, displayName string) {
	if m == nil {
		return "", ""
	}
	raw, ok := m[key].(map[string]any)
	if !ok || raw == nil {
		return "", ""
	}
	login = ghString(raw, "login")
	displayName = login
	if name := ghString(raw, "name"); name != "" {
		displayName = name
	}
	return login, displayName
}

func ghTime(m map[string]any, key string) time.Time {
	if t := ghTimePtr(m, key); t != nil {
		return *t
	}
	return time.Time{}
}

func ghTimePtr(m map[string]any, key string) *time.Time {
	raw := ghString(m, key)
	if raw == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &t
}

func validateGitHubIdent(name string) error {
	if name == "" || len(name) > 100 {
		return fmt.Errorf("invalid github identifier %q", name)
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid github identifier %q", name)
	}
	if !githubIdentPattern.MatchString(name) {
		return fmt.Errorf("invalid github identifier %q", name)
	}
	return nil
}

func sanitizeLog(s string) string {
	s = ansiEscapePattern.ReplaceAllString(s, "")
	s = ctrlCharPattern.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
