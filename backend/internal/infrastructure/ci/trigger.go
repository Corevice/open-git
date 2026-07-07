// Package ci glues git pushes and manual dispatches to workflow-run creation
// and execution. It is the producer side of the CI pipeline: it scans a
// repository's .github/workflows at a commit, records workflow_runs rows, and
// hands execution to the CI worker — via asynq when Redis is configured, or
// an in-process goroutine on single-node deployments.
package ci

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gopkg.in/yaml.v3"

	"github.com/open-git/backend/internal/domain/entity"
	infragit "github.com/open-git/backend/internal/infrastructure/git"
	"github.com/open-git/backend/internal/infrastructure/workflow"
	"github.com/open-git/backend/internal/worker"
)

const workflowsDir = ".github/workflows"

// Dispatcher hands a CI run payload to the executor.
type Dispatcher struct {
	client *asynq.Client
	worker *worker.CIWorker
}

// NewDispatcher builds a dispatcher. client may be nil (no Redis); execution
// then happens in-process on a goroutine, which is correct for single-node
// deployments — runs survive as 'queued'/'in_progress' rows either way.
func NewDispatcher(client *asynq.Client, ciWorker *worker.CIWorker) *Dispatcher {
	return &Dispatcher{client: client, worker: ciWorker}
}

func (d *Dispatcher) Dispatch(ctx context.Context, payload worker.CIRunPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	task := asynq.NewTask(worker.TypeCIRun, data)

	if d.client != nil {
		_, err := d.client.EnqueueContext(ctx, task)
		return err
	}
	if d.worker == nil {
		return fmt.Errorf("ci dispatcher has neither a queue client nor a worker")
	}
	go func() {
		// Detached from the request context: the push/dispatch HTTP request
		// finishing must not cancel the run.
		_ = d.worker.HandleCIRun(context.Background(), task)
	}()
	return nil
}

// RunCreator persists new workflow runs.
type RunCreator interface {
	Create(ctx context.Context, organizationID uuid.UUID, run *entity.WorkflowRun) error
}

// Trigger creates workflow runs from repository events.
type Trigger struct {
	runs       RunCreator
	dispatcher *Dispatcher
}

func NewTrigger(runs RunCreator, dispatcher *Dispatcher) *Trigger {
	return &Trigger{runs: runs, dispatcher: dispatcher}
}

// OnPush creates and dispatches a run for every workflow under
// .github/workflows at the pushed commit that listens to the push event.
// Errors are returned for logging but must never fail the push itself.
func (t *Trigger) OnPush(ctx context.Context, organizationID, repositoryID uuid.UUID, diskPath, branch, sha, actorLogin string) error {
	files, err := listWorkflowFiles(diskPath, sha)
	if err != nil || len(files) == 0 {
		return err
	}

	var firstErr error
	for _, file := range files {
		data, _, err := infragit.GetBlob(diskPath, sha, path.Join(workflowsDir, file))
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("read workflow %s: %w", file, err)
			}
			continue
		}
		wf, err := workflow.ParseWorkflow(data)
		if err != nil {
			// An invalid workflow file simply doesn't trigger; GitHub surfaces
			// this in the UI, we skip it.
			continue
		}
		if !workflowListensTo(wf.On, "push") {
			continue
		}
		if _, err := t.createAndDispatch(ctx, organizationID, repositoryID, file, branch, sha, "push", actorLogin, data); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// DispatchWorkflow creates and dispatches a run for one workflow file at the
// given branch head (the manual workflow-dispatch entry point).
func (t *Trigger) DispatchWorkflow(ctx context.Context, organizationID, repositoryID uuid.UUID, diskPath, workflowFile, branch, sha, actorLogin string) (*entity.WorkflowRun, error) {
	data, _, err := infragit.GetBlob(diskPath, sha, path.Join(workflowsDir, workflowFile))
	if err != nil {
		return nil, fmt.Errorf("read workflow %s: %w", workflowFile, err)
	}
	if _, err := workflow.ParseWorkflow(data); err != nil {
		return nil, fmt.Errorf("parse workflow %s: %w", workflowFile, err)
	}
	return t.createAndDispatch(ctx, organizationID, repositoryID, workflowFile, branch, sha, "workflow_dispatch", actorLogin, data)
}

// Redispatch re-reads the workflow YAML at the run's recorded commit and
// re-enqueues execution for the (already reset) run. Used by rerun.
func (t *Trigger) Redispatch(ctx context.Context, organizationID uuid.UUID, diskPath string, run *entity.WorkflowRun) error {
	data, _, err := infragit.GetBlob(diskPath, run.HeadSHA, path.Join(workflowsDir, run.Workflow))
	if err != nil {
		return fmt.Errorf("read workflow %s at %s: %w", run.Workflow, run.HeadSHA, err)
	}
	return t.dispatcher.Dispatch(ctx, worker.CIRunPayload{
		WorkflowRunID:  run.ID.String(),
		RepositoryID:   run.RepositoryID.String(),
		OrganizationID: organizationID.String(),
		WorkflowYAML:   data,
		HeadSHA:        run.HeadSHA,
		HeadBranch:     run.HeadBranch,
		Event:          run.Event,
		Actor:          run.ActorLogin,
		Workflow:       run.Workflow,
		RunNumber:      run.RunNumber,
	})
}

func (t *Trigger) createAndDispatch(ctx context.Context, organizationID, repositoryID uuid.UUID, workflowFile, branch, sha, event, actorLogin string, yamlData []byte) (*entity.WorkflowRun, error) {
	run := &entity.WorkflowRun{
		// ID is assigned by the repository as an int64-compatible UUID so the
		// Actions API can expose it as a stable numeric id.
		RepositoryID: repositoryID,
		Workflow:     workflowFile,
		HeadSHA:      sha,
		HeadBranch:   branch,
		Event:        event,
		ActorLogin:   actorLogin,
		Status:       "queued",
	}
	if err := t.runs.Create(ctx, organizationID, run); err != nil {
		return nil, fmt.Errorf("create workflow run: %w", err)
	}
	if err := t.dispatcher.Dispatch(ctx, worker.CIRunPayload{
		WorkflowRunID:  run.ID.String(),
		RepositoryID:   repositoryID.String(),
		OrganizationID: organizationID.String(),
		WorkflowYAML:   yamlData,
		HeadSHA:        run.HeadSHA,
		HeadBranch:     run.HeadBranch,
		Event:          run.Event,
		Actor:          run.ActorLogin,
		Workflow:       run.Workflow,
		RunNumber:      run.RunNumber,
	}); err != nil {
		return nil, fmt.Errorf("dispatch workflow run: %w", err)
	}
	return run, nil
}

func listWorkflowFiles(diskPath, ref string) ([]string, error) {
	entries, err := infragit.GetTree(diskPath, ref, workflowsDir)
	if err != nil {
		// No .github/workflows directory — nothing to trigger.
		return nil, nil
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Type != infragit.TreeEntryTypeFile {
			continue
		}
		name := path.Base(entry.Path)
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
			files = append(files, name)
		}
	}
	return files, nil
}

// workflowListensTo reports whether the workflow's `on:` covers the event.
// Supports the scalar (`on: push`), sequence (`on: [push, pull_request]`) and
// mapping (`on: {push: {branches: [...]}}`) forms; branch filters within the
// mapping form are not evaluated (every branch triggers).
func workflowListensTo(on yaml.Node, event string) bool {
	node := on
	// Resolve document/alias indirection defensively.
	for node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = *node.Content[0]
	}
	switch node.Kind {
	case yaml.ScalarNode:
		return node.Value == event
	case yaml.SequenceNode:
		for _, item := range node.Content {
			if item.Value == event {
				return true
			}
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == event {
				return true
			}
		}
	}
	return false
}
