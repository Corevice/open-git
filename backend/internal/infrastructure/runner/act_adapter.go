package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/open-git/backend/internal/domain/entity"
)

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
	if len(steps) == 0 {
		steps = []*Step{{Number: 1, Name: job.Name}}
	}
	return []byte(buildWorkflowYAML(job, steps))
}

func (a *ActAdapter) Execute(ctx context.Context, job ActJobPayload) error {
	if a.dockerHost == "" {
		return fmt.Errorf("docker unavailable: DOCKER_HOST not configured")
	}
	if len(job.WorkflowYAML) == 0 {
		return fmt.Errorf("workflow YAML is required")
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

	cmd := exec.CommandContext(ctx, "act", "--workflow", "/dev/stdin")
	cmd.Env = env
	cmd.Stdin = bytes.NewReader(job.WorkflowYAML)

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
	return cmd.Process.Signal(syscall.SIGTERM)
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
		if strings.ContainsAny(entry, "\x00\n\r") {
			return nil, fmt.Errorf("invalid env entry")
		}
		key, _, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid env entry %q", entry)
		}
		env = append(env, entry)
	}
	return env, nil
}
