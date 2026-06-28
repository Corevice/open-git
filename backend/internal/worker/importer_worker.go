package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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

type cloneFunc func(ctx context.Context, cloneURL, tmpDir, destPath string) error

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
	cloneFn       cloneFunc
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

func (w *ImporterWorker) WithCloneFn(fn cloneFunc) *ImporterWorker {
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

	jobID, err := uuid.Parse(payload.ImportJobID)
	if err != nil {
		return fmt.Errorf("invalid import job id: %w: %w", err, asynq.SkipRetry)
	}
	orgID, err := uuid.Parse(payload.OrganizationID)
	if err != nil {
		return fmt.Errorf("invalid organization id: %w: %w", err, asynq.SkipRetry)
	}

	job, err := w.importJobs.GetByID(ctx, jobID)
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

	if w.shouldRunPhase(payload.ResumeFromPhase, entity.ImportJobPhaseClone) {
		if err := w.importJobs.UpdatePhase(ctx, jobID, entity.ImportJobPhaseClone); err != nil {
			return fail(entity.ImportJobPhaseClone, err)
		}
		if err := w.runPhase(ctx, jobID, entity.ImportJobPhaseClone, func() error {
			return w.ingestClone(ctx, jobID, orgID, job.TargetName, ownerLogin, payload.SourceOwner, payload.SourceRepo, payload.Token)
		}); err != nil {
			return fail(entity.ImportJobPhaseClone, err)
		}
	}

	if w.shouldRunPhase(payload.ResumeFromPhase, entity.ImportJobPhaseMetadata) {
		if err := w.importJobs.UpdatePhase(ctx, jobID, entity.ImportJobPhaseMetadata); err != nil {
			return fail(entity.ImportJobPhaseMetadata, err)
		}
		if err := w.runPhase(ctx, jobID, entity.ImportJobPhaseMetadata, func() error {
			var ingestErr error
			repoID, gitPath, ingestErr = w.ingestMetadata(ctx, jobID, orgID, job, payload.SourceOwner, payload.SourceRepo, payload.Token, ownerLogin)
			return ingestErr
		}); err != nil {
			return fail(entity.ImportJobPhaseMetadata, err)
		}
	}

	if repoID == uuid.Nil {
		var resolveErr error
		repoID, gitPath, resolveErr = w.resolveTargetRepo(ctx, job)
		if resolveErr != nil {
			return fail(entity.ImportJobPhaseMetadata, resolveErr)
		}
	}

	if w.shouldRunPhase(payload.ResumeFromPhase, entity.ImportJobPhaseIssues) {
		if err := w.importJobs.UpdatePhase(ctx, jobID, entity.ImportJobPhaseIssues); err != nil {
			return fail(entity.ImportJobPhaseIssues, err)
		}
		if err := w.runPhase(ctx, jobID, entity.ImportJobPhaseIssues, func() error {
			cp, err := w.checkpoints.GetCheckpoint(ctx, jobID, entity.ImportJobPhaseIssues)
			if err != nil {
				return err
			}
			return w.ingestIssues(ctx, jobID, orgID, repoID, payload.SourceOwner, payload.SourceRepo, payload.Token, cp)
		}); err != nil {
			return fail(entity.ImportJobPhaseIssues, err)
		}
	}

	if w.shouldRunPhase(payload.ResumeFromPhase, entity.ImportJobPhasePullRequests) {
		if err := w.importJobs.UpdatePhase(ctx, jobID, entity.ImportJobPhasePullRequests); err != nil {
			return fail(entity.ImportJobPhasePullRequests, err)
		}
		if err := w.runPhase(ctx, jobID, entity.ImportJobPhasePullRequests, func() error {
			cp, err := w.checkpoints.GetCheckpoint(ctx, jobID, entity.ImportJobPhasePullRequests)
			if err != nil {
				return err
			}
			return w.ingestPullRequests(ctx, jobID, orgID, repoID, gitPath, payload.SourceOwner, payload.SourceRepo, payload.Token, cp)
		}); err != nil {
			return fail(entity.ImportJobPhasePullRequests, err)
		}
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
	w.logger.Error("import failed", "job_id", jobID.String(), "phase", string(phase), "error", err)
	_ = w.importJobs.SetError(ctx, jobID, err.Error())
	_ = w.importJobs.UpdateStatus(ctx, jobID, entity.ImportJobStatusFailed)
	return err
}

func (w *ImporterWorker) shouldRunPhase(resumeFrom string, phase entity.ImportJobPhase) bool {
	if resumeFrom == "" {
		return true
	}
	return phaseIndex(phase) >= phaseIndex(entity.ImportJobPhase(resumeFrom))
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
	case entity.ImportJobPhaseWiki:
		return 4
	case entity.ImportJobPhaseDone:
		return 5
	default:
		return 0
	}
}

func (w *ImporterWorker) runPhase(ctx context.Context, jobID uuid.UUID, phase entity.ImportJobPhase, fn func() error) error {
	cp, err := w.checkpoints.GetCheckpoint(ctx, jobID, phase)
	if err != nil {
		return err
	}
	if cp != nil && cp.Completed {
		return nil
	}
	if err := fn(); err != nil {
		return err
	}
	return w.checkpoints.MarkPhaseComplete(ctx, jobID, phase)
}

func (w *ImporterWorker) mapUser(ctx context.Context, jobID uuid.UUID, githubLogin, displayName string) uuid.UUID {
	if githubLogin == "" {
		return uuid.Nil
	}

	existing, err := w.userMappings.GetMappingByLogin(ctx, jobID, githubLogin)
	if err == nil && existing != nil {
		if existing.LocalUserID != nil {
			return *existing.LocalUserID
		}
		return uuid.Nil
	}

	mapping := &entity.ImportUserMapping{
		ImportJobID:       jobID,
		GitHubLogin:       githubLogin,
		GitHubDisplayName: displayName,
	}

	localUser, err := w.users.GetByLogin(ctx, githubLogin)
	if err == nil && localUser != nil {
		mapping.LocalUserID = &localUser.ID
		_ = w.userMappings.UpsertMapping(ctx, mapping)
		return localUser.ID
	}

	_ = w.userMappings.UpsertMapping(ctx, mapping)
	return uuid.Nil
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
	visibility := entity.VisibilityPublic
	if ghBool(ghRepo, "private") {
		visibility = entity.VisibilityPrivate
	}

	gitPath := w.repoGitPath(ownerLogin, job.TargetName)
	repository := &entity.Repository{
		OrganizationID: orgID,
		OwnerID:        job.CreatedBy,
		Name:           job.TargetName,
		Description:    description,
		GitPath:        gitPath,
		OwnerLogin:     ownerLogin,
		Visibility:     visibility,
		DefaultBranch:  "main",
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
	checkpoint *entity.ImportPhaseCheckpoint,
) error {
	page := 1
	if checkpoint != nil && checkpoint.LastCursor != "" {
		if p, err := strconv.Atoi(checkpoint.LastCursor); err == nil && p > 0 {
			page = p
		}
	}

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
			if err := w.importIssue(ctx, jobID, orgID, repoID, owner, repo, token, item); err != nil {
				w.logger.Warn("skipped issue", "job_id", jobID.String(), "error", err)
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
	item map[string]any,
) error {
	number := ghInt(item, "number")
	title := ghString(item, "title")
	body := ghString(item, "body")
	state := ghString(item, "state")
	if state == "" {
		state = "open"
	}

	authorLogin, authorName := ghUser(item, "user")
	authorID := w.mapUser(ctx, jobID, authorLogin, authorName)

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
		milestoneID, err := w.ensureMilestone(ctx, orgID, repoID, milestoneRaw)
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
			labelID, err := w.ensureLabel(ctx, orgID, repoID, labelMap)
			if err != nil {
				return err
			}
			if labelID != uuid.Nil {
				_ = w.labels.AddToIssue(ctx, repoID, number, labelID)
			}
		}
	}

	return w.importIssueComments(ctx, jobID, orgID, issue.ID, owner, repo, token, number)
}

func (w *ImporterWorker) importIssueComments(
	ctx context.Context,
	jobID, orgID, issueID uuid.UUID,
	owner, repo, token string,
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
			authorID := w.mapUser(ctx, jobID, login, displayName)
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

func (w *ImporterWorker) ensureLabel(ctx context.Context, orgID, repoID uuid.UUID, raw map[string]any) (uuid.UUID, error) {
	name := ghString(raw, "name")
	if name == "" {
		return uuid.Nil, nil
	}
	existing, err := w.labels.GetByName(ctx, repoID, name)
	if err == nil && existing != nil {
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
	return label.ID, nil
}

func (w *ImporterWorker) ensureMilestone(ctx context.Context, orgID, repoID uuid.UUID, raw map[string]any) (uuid.UUID, error) {
	number := ghInt(raw, "number")
	if number == 0 {
		return uuid.Nil, nil
	}
	existing, err := w.milestones.GetByNumber(ctx, repoID, number)
	if err == nil && existing != nil {
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
	return milestone.ID, nil
}

func (w *ImporterWorker) ingestPullRequests(
	ctx context.Context,
	jobID, orgID, repoID uuid.UUID,
	gitPath, owner, repo, token string,
	checkpoint *entity.ImportPhaseCheckpoint,
) error {
	page := 1
	if checkpoint != nil && checkpoint.LastCursor != "" {
		if p, err := strconv.Atoi(checkpoint.LastCursor); err == nil && p > 0 {
			page = p
		}
	}

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
			if err := w.importPullRequest(ctx, jobID, orgID, repoID, gitPath, item); err != nil {
				w.logger.Warn("skipped pull request", "job_id", jobID.String(), "error", err)
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
	item map[string]any,
) error {
	headRef := ghNestedString(item, "head", "ref")
	baseRef := ghNestedString(item, "base", "ref")
	headSHA := ghNestedString(item, "head", "sha")
	baseSHA := ghNestedString(item, "base", "sha")

	login, displayName := ghUser(item, "user")
	authorID := w.mapUser(ctx, jobID, login, displayName)

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

func (w *ImporterWorker) ingestClone(
	ctx context.Context,
	jobID, orgID uuid.UUID,
	targetName, ownerLogin, sourceOwner, sourceRepo, token string,
) error {
	_ = orgID

	if w.gitStoragePath == "" {
		return fmt.Errorf("%s is not configured", gitStorageEnv)
	}

	tmpDir, err := os.MkdirTemp("", "open-git-import-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	destPath := w.repoGitPath(ownerLogin, targetName)
	cloneURL := fmt.Sprintf("https://%s@github.com/%s/%s.git", token, sourceOwner, sourceRepo)

	if w.cloneFn != nil {
		return w.cloneFn(ctx, cloneURL, tmpDir, destPath)
	}

	if _, err := gogit.PlainClone(tmpDir, false, &gogit.CloneOptions{URL: cloneURL}); err != nil {
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
		URL: "file://" + filepath.ToSlash(absTmp),
	}); err != nil {
		return fmt.Errorf("mirror to local storage: %w", err)
	}

	w.logger.Info("clone complete", "job_id", jobID.String(), "dest", destPath)
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

func (w *ImporterWorker) repoGitPath(ownerLogin, repoName string) string {
	return filepath.Join(w.gitStoragePath, ownerLogin, repoName+".git")
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
	resp, items, err := w.fetchList(ctx, url, token, jobID)
	if err != nil {
		return resp, nil, err
	}
	if len(items) == 0 {
		return resp, nil, fmt.Errorf("empty response from %s", url)
	}
	return resp, items[0], nil
}

func (w *ImporterWorker) fetchList(ctx context.Context, url, token string, jobID uuid.UUID) (*http.Response, []map[string]any, error) {
	for {
		resp, items, err := w.fetchPage(ctx, url, token)
		if err != nil {
			return resp, nil, err
		}
		if err := w.handleRateLimit(ctx, resp, jobID); err != nil {
			if errors.Is(err, ErrRateLimitExceeded) {
				continue
			}
			return resp, nil, err
		}
		return resp, items, nil
	}
}

func (w *ImporterWorker) handleRateLimit(ctx context.Context, resp *http.Response, jobID uuid.UUID) error {
	err := w.checkRateLimitHeaders(ctx, resp)
	if !errors.Is(err, ErrRateLimitExceeded) {
		return err
	}

	resetStr := resp.Header.Get(rateLimitResetHeader)
	resetTs, parseErr := strconv.ParseInt(resetStr, 10, 64)
	if parseErr != nil || resetTs <= 0 {
		return ErrRateLimitExceeded
	}

	wait := time.Until(time.Unix(resetTs, 0))
	if wait > 0 {
		w.logger.Info("rate limit reached, sleeping", "job_id", jobID.String(), "wait_seconds", wait.Seconds())
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	return ErrRateLimitExceeded
}

func (w *ImporterWorker) fetchPage(ctx context.Context, url, token string) (*http.Response, []map[string]any, error) {
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
		return resp, nil, fmt.Errorf("non-2xx response: %d body=%s", resp.StatusCode, truncate(string(body), 200))
	}

	items, err := decodeItems(body)
	if err != nil {
		return resp, nil, fmt.Errorf("decode body: %w", err)
	}
	return resp, items, nil
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

func (w *ImporterWorker) checkRateLimitHeaders(ctx context.Context, resp *http.Response) error {
	if resp == nil {
		return nil
	}
	remainingStr := resp.Header.Get(rateLimitRemainingHeader)
	if remainingStr == "" {
		return nil
	}
	remaining, err := strconv.Atoi(remainingStr)
	if err != nil {
		return nil
	}
	if remaining == 0 {
		return ErrRateLimitExceeded
	}
	if remaining >= rateLimitThreshold {
		return nil
	}
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
	s := ghString(m, key)
	if s == "" {
		if v, ok := m[key].(float64); ok {
			return int(v)
		}
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
