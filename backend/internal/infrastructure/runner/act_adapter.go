package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/open-git/backend/internal/domain/entity"
)

const (
	maxWorkflowYAMLSize = 64 * 1024
	maxEnvEntryLen      = 4096
	cancelWaitTimeout   = 30 * time.Second
)

var actWorkflowPrefix = "on: workflow_dispatch\njobs:\n"

var safeEnvKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var blockedEnvKeys = map[string]struct{}{
	"DOCKER_HOST": {},
	"PATH":        {},
	"LD_PRELOAD":  {},
	"LD_LIBRARY_PATH": {},
	"HOME":        {},
}

type ActAdapter struct {
	dockerHost string
	mu         sync.Mutex
	processes  map[string]*exec.Cmd
}

func NewActAdapter(dockerHost string) *ActAdapter {
	return &ActAdapter{
		dockerHost: dockerHost,
		processes:  make(map[string]*exec.Cmd),
	}
}

// BuildActWorkflowYAML builds minimal workflow YAML for act execution.
func BuildActWorkflowYAML(job *entity.WorkflowJob, steps []*Step) []byte {
	if job == nil {
		return nil
	}
	if err := validateActJobName(job.Name); err != nil {
		return nil
	}
	if len(steps) == 0 {
		steps = []*Step{{Number: 1, Name: job.Name}}
	}
	return []byte(buildWorkflowYAML(job, steps))
}

func (a *ActAdapter) Execute(ctx context.Context, job RunnerJobPayload) error {
	if a.dockerHost == "" {
		return fmt.Errorf("docker unavailable: DOCKER_HOST not configured")
	}
	if len(job.WorkflowYAML) == 0 {
		return fmt.Errorf("workflow YAML is required")
	}
	if err := validateActWorkflowYAML(job.WorkflowYAML); err != nil {
		return fmt.Errorf("validate workflow YAML: %w", err)
	}

	if job.TimeoutMinutes > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(job.TimeoutMinutes)*time.Minute)
		defer cancel()
	}

	env, err := buildActEnv(a.dockerHost, job.Env)
	if err != nil {
		return fmt.Errorf("build act env: %w", err)
	}

	workflowFile, err := writeWorkflowYAMLFile(job.WorkflowYAML)
	if err != nil {
		return fmt.Errorf("write workflow file: %w", err)
	}
	defer os.Remove(workflowFile)

	cmd := exec.CommandContext(ctx, "act", "--workflow", workflowFile)
	cmd.Env = env

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start act: %w", err)
	}

	a.mu.Lock()
	a.processes[job.JobID] = cmd
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.processes, job.JobID)
		a.mu.Unlock()
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("act execution failed: %w: %s", err, output.String())
	}
	return nil
}

func (a *ActAdapter) Cancel(_ context.Context, jobID string) error {
	a.mu.Lock()
	cmd, ok := a.processes[jobID]
	a.mu.Unlock()
	if !ok || cmd.Process == nil {
		return nil
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(cancelWaitTimeout):
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("kill act process after cancel timeout: %w", err)
		}
		<-done
		return nil
	}
}

func validateActWorkflowYAML(yaml []byte) error {
	if len(yaml) > maxWorkflowYAMLSize {
		return fmt.Errorf("workflow YAML exceeds size limit")
	}
	content := string(yaml)
	if !strings.HasPrefix(content, actWorkflowPrefix) {
		return fmt.Errorf("unexpected workflow YAML structure")
	}
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "run:") && !strings.Contains(line, "run: echo ") {
			return fmt.Errorf("workflow YAML contains disallowed run command")
		}
	}
	return nil
}

func writeWorkflowYAMLFile(yaml []byte) (string, error) {
	f, err := os.CreateTemp("", "act-workflow-*.yaml")
	if err != nil {
		return "", err
	}
	path := f.Name()
	if _, err := f.Write(yaml); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

func buildActEnv(dockerHost string, extra []string) ([]string, error) {
	env := []string{"DOCKER_HOST=" + dockerHost}
	if path := os.Getenv("PATH"); path != "" {
		env = append(env, "PATH="+path)
	}
	for _, entry := range extra {
		if entry == "" {
			continue
		}
		if len(entry) > maxEnvEntryLen {
			return nil, fmt.Errorf("env entry exceeds size limit")
		}
		if strings.ContainsAny(entry, "\x00\n\r") {
			return nil, fmt.Errorf("invalid env entry")
		}
		key, _, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid env entry %q", entry)
		}
		if _, blocked := blockedEnvKeys[key]; blocked {
			return nil, fmt.Errorf("env key %q not allowed", key)
		}
		if !safeEnvKey.MatchString(key) {
			return nil, fmt.Errorf("invalid env key %q", key)
		}
		env = append(env, entry)
	}
	return env, nil
}
