package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
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

func (a *ActAdapter) Execute(ctx context.Context, job ActJobPayload) error {
	if a.dockerHost == "" {
		return fmt.Errorf("docker unavailable: DOCKER_HOST not configured")
	}

	if job.TimeoutMinutes > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(job.TimeoutMinutes)*time.Minute)
		defer cancel()
	}

	tmpFile, err := os.CreateTemp("", "act-workflow-*.yml")
	if err != nil {
		return fmt.Errorf("create workflow temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(job.WorkflowYAML); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write workflow temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close workflow temp file: %w", err)
	}

	cmd := exec.CommandContext(ctx, "act", "--workflow", tmpPath)
	cmd.Env = append(os.Environ(), job.Env...)
	cmd.Env = append(cmd.Env, "DOCKER_HOST="+a.dockerHost)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	a.mu.Lock()
	a.processes[job.JobID] = cmd
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.processes, job.JobID)
		a.mu.Unlock()
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start act: %w", err)
	}

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
