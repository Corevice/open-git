package worker

import (
	"context"
	"os"
	"os/exec"
)

// Sandbox modes for CI step execution.
const (
	SandboxModeNone   = "none"
	SandboxModeDocker = "docker"
)

// sandbox describes how CI workflow steps are executed.
type sandbox struct {
	// mode is SandboxModeNone (direct on host) or SandboxModeDocker
	// (ephemeral, network-isolated container).
	mode  string
	image string
}

func newSandbox(mode, image string) sandbox {
	if mode != SandboxModeDocker {
		mode = SandboxModeNone
	}
	if image == "" {
		image = "alpine:3"
	}
	return sandbox{mode: mode, image: image}
}

// baseStepEnv is the controlled environment a step runs with. The server's own
// environment (which holds JWT_SECRET, DB credentials, etc.) is NEVER inherited
// — only this fixed base plus the workflow's declared secrets.
func baseStepEnv(workdir string) []string {
	return []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=" + workdir,
		"CI=true",
	}
}

// buildCommand builds the *exec.Cmd that runs script with the given secret
// environment in workdir. secretEnv is a list of KEY=VALUE entries for the
// workflow's secrets only.
//
// In "none" mode the step runs as `sh -c` directly on the host, but with a
// clean environment and its working directory confined to workdir. This is
// appropriate only for trusted single-user/team instances.
//
// In "docker" mode the step runs inside a fresh, disposable container with the
// working directory bind-mounted, no network, resource caps, and only the
// declared secrets injected — providing real isolation from the server host
// and from other jobs, suitable for untrusted or multi-tenant workflows.
func (s sandbox) buildCommand(ctx context.Context, workdir string, secretEnv []string, script string) *exec.Cmd {
	if s.mode == SandboxModeDocker {
		args := []string{
			"run", "--rm", "-i",
			"--network", "none",
			"--memory", "1g",
			"--cpus", "1",
			"--pids-limit", "512",
			"-v", workdir + ":/workspace",
			"-w", "/workspace",
			"-e", "CI=true",
		}
		for _, e := range secretEnv {
			args = append(args, "-e", e)
		}
		args = append(args, s.image, "sh", "-c", script)
		cmd := exec.CommandContext(ctx, "docker", args...)
		// The docker CLI itself only needs enough environment to reach the
		// daemon; the step's environment is what was passed via -e above.
		cmd.Env = dockerCLIEnv()
		return cmd
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Dir = workdir
	cmd.Env = append(baseStepEnv(workdir), secretEnv...)
	return cmd
}

// dockerCLIEnv returns the minimal environment the docker CLI needs to talk to
// the daemon, without leaking the rest of the server's environment.
func dockerCLIEnv() []string {
	env := []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
	for _, key := range []string{"DOCKER_HOST", "DOCKER_TLS_VERIFY", "DOCKER_CERT_PATH", "HOME"} {
		if v := os.Getenv(key); v != "" {
			env = append(env, key+"="+v)
		}
	}
	return env
}
