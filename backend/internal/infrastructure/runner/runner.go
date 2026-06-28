package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/open-git/backend/internal/domain/entity"
)

type Step struct {
	Number int
	Name   string
}

type Runner interface {
	ExecuteJob(
		ctx context.Context,
		job *entity.WorkflowJob,
		steps []*Step,
		secrets map[string]string,
		logFn func(offset int64, chunk string),
	) error
}

type MockRunner struct {
	Chunks []string
	Err    error
}

func (m *MockRunner) ExecuteJob(
	_ context.Context,
	_ *entity.WorkflowJob,
	_ []*Step,
	_ map[string]string,
	logFn func(offset int64, chunk string),
) error {
	var offset int64
	for _, chunk := range m.Chunks {
		logFn(offset, chunk)
		offset++
	}
	return m.Err
}

type DockerActRunner struct {
	ActPath string
}

func NewDockerActRunner(actPath string) *DockerActRunner {
	if actPath == "" {
		actPath = "act"
	}
	return &DockerActRunner{ActPath: actPath}
}

func (r *DockerActRunner) ExecuteJob(
	ctx context.Context,
	job *entity.WorkflowJob,
	steps []*Step,
	secrets map[string]string,
	logFn func(offset int64, chunk string),
) error {
	workflowYAML := buildWorkflowYAML(job, steps)
	args := []string{"--workflow", "/dev/stdin", "--job", job.Name}
	for key, value := range secrets {
		args = append(args, "--input", key+"="+value)
	}

	cmd := exec.CommandContext(ctx, r.ActPath, args...)
	cmd.Stdin = strings.NewReader(workflowYAML)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start act: %w", err)
	}

	secretValues := make([]string, 0, len(secrets))
	for _, value := range secrets {
		secretValues = append(secretValues, value)
	}

	var offset int64
	streamLines := func(reader io.Reader) error {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := MaskSecrets(scanner.Text(), secretValues)
			logFn(offset, line)
			offset++
		}
		return scanner.Err()
	}

	if err := streamLines(stdout); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("read act stdout: %w", err)
	}
	if err := streamLines(stderr); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("read act stderr: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("act execution failed: %w", err)
	}
	return nil
}

func buildWorkflowYAML(job *entity.WorkflowJob, steps []*Step) string {
	var b strings.Builder
	b.WriteString("on: workflow_dispatch\njobs:\n")
	b.WriteString("  ")
	b.WriteString(job.Name)
	b.WriteString(":\n    runs-on: ubuntu-latest\n    steps:\n")
	for _, step := range steps {
		b.WriteString("      - name: ")
		b.WriteString(strconvQuote(step.Name))
		b.WriteString("\n        run: echo ")
		b.WriteString(strconvQuote(step.Name))
		b.WriteString("\n")
	}
	return b.String()
}

func strconvQuote(s string) string {
	if strings.ContainsAny(s, ":'\"\n") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
