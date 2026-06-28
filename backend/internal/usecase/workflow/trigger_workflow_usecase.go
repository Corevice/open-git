package workflow

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	infrgit "github.com/open-git/backend/internal/infrastructure/git"
	"github.com/open-git/backend/internal/infrastructure/queue"
	wfparser "github.com/open-git/backend/internal/infrastructure/workflow"
)

const (
	runStatusQueued    = "queued"
	runStatusCompleted = "completed"

	conclusionFailure = "failure"
)

type TriggerWorkflowInput struct {
	RepositoryID uuid.UUID
	OrgID        uuid.UUID
	ActorID      uuid.UUID
	Event        string
	HeadSHA      string
	HeadBranch   string
	Ref          string
	Inputs       map[string]string
}

type TriggerWorkflowOutput struct {
	RunID uuid.UUID
}

type WorkflowScheduleEnqueuer interface {
	EnqueueSchedule(ctx context.Context, payload queue.WorkflowSchedulePayload) error
}

type asynqWorkflowScheduleEnqueuer struct {
	client *asynq.Client
}

func newAsynqWorkflowScheduleEnqueuer(client *asynq.Client) WorkflowScheduleEnqueuer {
	return &asynqWorkflowScheduleEnqueuer{client: client}
}

func (e *asynqWorkflowScheduleEnqueuer) EnqueueSchedule(ctx context.Context, payload queue.WorkflowSchedulePayload) error {
	_, err := queue.EnqueueWorkflowSchedule(ctx, e.client, payload)
	return err
}

type workflowParser interface {
	Parse(data []byte) (*wfparser.Workflow, error)
	Analyze(wf *wfparser.Workflow) (*wfparser.WorkflowIR, error)
}

type defaultWorkflowParser struct{}

func (defaultWorkflowParser) Parse(data []byte) (*wfparser.Workflow, error) {
	return wfparser.ParseWorkflow(data)
}

func (defaultWorkflowParser) Analyze(wf *wfparser.Workflow) (*wfparser.WorkflowIR, error) {
	return wfparser.AnalyzeWorkflow(wf)
}

type repositoryGitPathLookup interface {
	GetByID(ctx context.Context, repoID, orgID uuid.UUID) (*entity.Repository, error)
}

type TriggerWorkflowUsecase struct {
	workflowRepo domainrepo.IWorkflowRepository
	runRepo      domainrepo.IWorkflowRunRepository
	jobRepo      domainrepo.IWorkflowJobRepository
	stepRepo     domainrepo.IWorkflowStepRepository
	repoRepo     repositoryGitPathLookup
	enqueuer     WorkflowScheduleEnqueuer
	parser       workflowParser
	readFile     func(repoPath, ref, filePath string) ([]byte, error)
}

func NewTriggerWorkflowUsecase(
	workflowRepo domainrepo.IWorkflowRepository,
	runRepo domainrepo.IWorkflowRunRepository,
	jobRepo domainrepo.IWorkflowJobRepository,
	stepRepo domainrepo.IWorkflowStepRepository,
	repoRepo repositoryGitPathLookup,
	client *asynq.Client,
) *TriggerWorkflowUsecase {
	return NewTriggerWorkflowUsecaseWithDeps(
		workflowRepo,
		runRepo,
		jobRepo,
		stepRepo,
		repoRepo,
		newAsynqWorkflowScheduleEnqueuer(client),
		defaultWorkflowParser{},
		infrgit.ReadFile,
	)
}

func NewTriggerWorkflowUsecaseWithDeps(
	workflowRepo domainrepo.IWorkflowRepository,
	runRepo domainrepo.IWorkflowRunRepository,
	jobRepo domainrepo.IWorkflowJobRepository,
	stepRepo domainrepo.IWorkflowStepRepository,
	repoRepo repositoryGitPathLookup,
	enqueuer WorkflowScheduleEnqueuer,
	parser workflowParser,
	readFile func(repoPath, ref, filePath string) ([]byte, error),
) *TriggerWorkflowUsecase {
	if readFile == nil {
		readFile = infrgit.ReadFile
	}
	return &TriggerWorkflowUsecase{
		workflowRepo: workflowRepo,
		runRepo:      runRepo,
		jobRepo:      jobRepo,
		stepRepo:     stepRepo,
		repoRepo:     repoRepo,
		enqueuer:     enqueuer,
		parser:       parser,
		readFile:     readFile,
	}
}

func (uc *TriggerWorkflowUsecase) Execute(ctx context.Context, input TriggerWorkflowInput) (*TriggerWorkflowOutput, error) {
	workflows, err := uc.workflowRepo.ListActiveByRepository(ctx, input.OrgID, input.RepositoryID)
	if err != nil {
		return nil, err
	}

	repo, err := uc.repoRepo.GetByID(ctx, input.RepositoryID, input.OrgID)
	if err != nil {
		return nil, err
	}

	ref := input.Ref
	if ref == "" {
		ref = input.HeadSHA
	}
	if ref == "" {
		ref = input.HeadBranch
	}

	var lastRunID uuid.UUID
	createdAny := false

	for _, wf := range workflows {
		data, err := uc.readFile(repo.GitPath, ref, wf.Path)
		if err != nil {
			return nil, fmt.Errorf("read workflow %q: %w", wf.Path, err)
		}

		parsed, err := uc.parser.Parse(data)
		if err != nil {
			runID, createErr := uc.createFailureRun(ctx, input, wf, err.Error())
			if createErr != nil {
				return nil, createErr
			}
			lastRunID = runID
			createdAny = true
			continue
		}

		ir, err := uc.parser.Analyze(parsed)
		if err != nil {
			if isDAGValidationError(err) {
				return nil, err
			}
			runID, createErr := uc.createFailureRun(ctx, input, wf, err.Error())
			if createErr != nil {
				return nil, createErr
			}
			lastRunID = runID
			createdAny = true
			continue
		}

		if err := validateWorkflowIR(ir); err != nil {
			return nil, err
		}

		if len(ir.Jobs) == 0 {
			runID, createErr := uc.createFailureRun(ctx, input, wf, "workflow must contain at least one job")
			if createErr != nil {
				return nil, createErr
			}
			lastRunID = runID
			createdAny = true
			continue
		}

		if !matchesTrigger(input.Event, input.HeadBranch, input.Ref, ir.On) {
			continue
		}

		runID, err := uc.createQueuedRun(ctx, input, wf, ir)
		if err != nil {
			return nil, err
		}

		if err := uc.enqueuer.EnqueueSchedule(ctx, queue.WorkflowSchedulePayload{
			RunID: runID.String(),
			OrgID: input.OrgID.String(),
		}); err != nil {
			return nil, err
		}

		lastRunID = runID
		createdAny = true
	}

	if !createdAny {
		return &TriggerWorkflowOutput{}, nil
	}

	return &TriggerWorkflowOutput{RunID: lastRunID}, nil
}

func (uc *TriggerWorkflowUsecase) createFailureRun(
	ctx context.Context,
	input TriggerWorkflowInput,
	wf *entity.Workflow,
	errMsg string,
) (uuid.UUID, error) {
	runNumber, err := uc.runRepo.IncrementRunNumber(ctx, input.RepositoryID, wf.ID)
	if err != nil {
		return uuid.Nil, err
	}

	now := time.Now().UTC()
	runID := uuid.New()
	run := &entity.WorkflowRun{
		ID:                runID,
		OrganizationID:      input.OrgID,
		RepositoryID:        input.RepositoryID,
		WorkflowID:          wf.ID,
		Workflow:            wf.Path,
		HeadSHA:             input.HeadSHA,
		HeadBranch:          input.HeadBranch,
		Event:               input.Event,
		Status:              runStatusCompleted,
		Conclusion:          conclusionFailure,
		RunNumber:           runNumber,
		RunAttempt:          1,
		TriggeredByUserID:   input.ActorID,
		ErrorMessage:        errMsg,
		CreatedAt:           now,
		UpdatedAt:           now,
		StartedAt:           &now,
		CompletedAt:         &now,
	}

	if err := uc.runRepo.Create(ctx, run); err != nil {
		return uuid.Nil, err
	}

	return runID, nil
}

func (uc *TriggerWorkflowUsecase) createQueuedRun(
	ctx context.Context,
	input TriggerWorkflowInput,
	wf *entity.Workflow,
	ir *wfparser.WorkflowIR,
) (uuid.UUID, error) {
	runNumber, err := uc.runRepo.IncrementRunNumber(ctx, input.RepositoryID, wf.ID)
	if err != nil {
		return uuid.Nil, err
	}

	now := time.Now().UTC()
	runID := uuid.New()
	run := &entity.WorkflowRun{
		ID:              runID,
		OrganizationID:  input.OrgID,
		RepositoryID:    input.RepositoryID,
		WorkflowID:      wf.ID,
		Workflow:        wf.Path,
		HeadSHA:         input.HeadSHA,
		HeadBranch:      input.HeadBranch,
		Event:           input.Event,
		Status:          runStatusQueued,
		Conclusion:      "",
		RunNumber:       runNumber,
		RunAttempt:      1,
		TriggeredByUserID: input.ActorID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := uc.runRepo.Create(ctx, run); err != nil {
		return uuid.Nil, err
	}

	jobs := make([]*entity.WorkflowJob, 0, len(ir.Jobs))
	steps := make([]*entity.WorkflowStep, 0)

	for jobName, irJob := range ir.Jobs {
		jobID := uuid.New()
		job := &entity.WorkflowJob{
			ID:             jobID,
			RunID:          runID,
			OrganizationID: input.OrgID,
			Name:           jobName,
			Status:         "queued",
			Conclusion:     "",
			Needs:          append([]string(nil), irJob.Needs...),
			RunnerLabel:    irJob.RunsOn,
			CreatedAt:      now,
		}
		jobs = append(jobs, job)

		for i, irStep := range irJob.Steps {
			stepName := irStep.ID
			if stepName == "" {
				stepName = irStep.Uses
			}
			if stepName == "" {
				stepName = irStep.Run
			}
			if stepName == "" {
				stepName = fmt.Sprintf("Step %d", i+1)
			}

			steps = append(steps, &entity.WorkflowStep{
				ID:         uuid.New(),
				JobID:      jobID,
				Number:     i + 1,
				Name:       stepName,
				Status:     "queued",
				Conclusion: "",
			})
		}
	}

	if len(jobs) > 0 {
		for _, job := range jobs {
			if err := uc.jobRepo.Create(ctx, job); err != nil {
				return uuid.Nil, err
			}
		}
	}
	if len(steps) > 0 {
		if err := uc.stepRepo.CreateBatch(ctx, steps); err != nil {
			return uuid.Nil, err
		}
	}

	return runID, nil
}

func validateWorkflowIR(ir *wfparser.WorkflowIR) error {
	if ir == nil {
		return fmt.Errorf("%w: workflow IR is nil", domain.ErrValidation)
	}
	if len(ir.Jobs) == 0 {
		return nil
	}
	if len(ir.DAG.Order) < len(ir.Jobs) {
		return fmt.Errorf("%w: cyclic dependency detected in workflow jobs", domain.ErrValidation)
	}
	for jobID, job := range ir.Jobs {
		for _, need := range job.Needs {
			if _, ok := ir.Jobs[need]; !ok {
				return fmt.Errorf("%w: job %q needs unknown job %q", domain.ErrValidation, jobID, need)
			}
		}
	}
	return nil
}

func isDAGValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cyclic dependency") || strings.Contains(msg, "needs unknown job")
}

func matchesTrigger(event, headBranch, ref string, on map[string]any) bool {
	if len(on) == 0 {
		return false
	}

	trigger, ok := on[event]
	if !ok {
		return false
	}

	triggerMap, ok := trigger.(map[string]any)
	if !ok || len(triggerMap) == 0 {
		return true
	}

	branch := normalizeBranch(headBranch)
	if branch == "" {
		branch = normalizeBranch(ref)
	}

	if branches, ok := triggerMap["branches"]; ok {
		patterns := toStringSlice(branches)
		if len(patterns) > 0 && !matchesBranchFilter(patterns, branch) {
			return false
		}
	}

	if tags, ok := triggerMap["tags"]; ok {
		tagRef := normalizeTagRef(ref)
		patterns := toStringSlice(tags)
		if len(patterns) > 0 && !matchesBranchFilter(patterns, tagRef) {
			return false
		}
	}

	if paths, ok := triggerMap["paths"]; ok {
		_ = toStringSlice(paths)
	}

	return true
}

func normalizeBranch(branch string) string {
	branch = strings.TrimSpace(branch)
	return strings.TrimPrefix(branch, "refs/heads/")
}

func normalizeTagRef(ref string) string {
	ref = strings.TrimSpace(ref)
	return strings.TrimPrefix(ref, "refs/tags/")
}

func toStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		return []string{v}
	default:
		return nil
	}
}

func matchesBranchFilter(patterns []string, branch string) bool {
	if branch == "" {
		return false
	}
	for _, pattern := range patterns {
		if matchBranchPattern(pattern, branch) {
			return true
		}
	}
	return false
}

func matchBranchPattern(pattern, branch string) bool {
	pattern = normalizeBranch(pattern)
	if pattern == branch {
		return true
	}
	if strings.Contains(pattern, "*") {
		ok, err := path.Match(pattern, branch)
		return err == nil && ok
	}
	return false
}
