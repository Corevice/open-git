# Security Policy

## Reporting a vulnerability

If you discover a security vulnerability, please report it privately rather than
opening a public issue. Use GitHub's "Report a vulnerability" (Security →
Advisories) on this repository, or contact the maintainers directly. Please
include steps to reproduce and the affected version/commit. We aim to acknowledge
reports promptly and will coordinate a fix and disclosure timeline with you.

## Deployment security model

open-git is designed to be self-hosted. Its default posture assumes a
**single user or a trusted team**. Read the following before exposing an
instance to untrusted or public users.

### CI / Actions execution isolation

How workflow steps under `.github/workflows` are executed is controlled by
`CI_SANDBOX_MODE`:

- `none` (default) — steps run with `sh -c` directly on the API server's host.
  They run with a **clean environment** (the server's own environment, including
  `JWT_SECRET` and database credentials, is never inherited — only the workflow's
  declared secrets are injected) and with their working directory confined to a
  fresh temporary directory that is deleted after the run. There is, however, no
  container/VM/user isolation: a workflow can still run arbitrary commands as the
  server's OS user. **This mode is only appropriate when everyone who can push a
  workflow is trusted** (a personal instance, or an internal team behind
  authentication).
- `docker` — each job runs inside a fresh, disposable container
  (`CI_SANDBOX_IMAGE`, default `alpine:3`) with the working directory
  bind-mounted, **no network** (`--network none`), CPU/memory/pid limits, and
  only the declared secrets injected. The host filesystem and the server process
  are not reachable from the job. **Use this mode for multi-tenant or public
  instances.** It requires the server to have access to a Docker daemon.

Verified isolation in `docker` mode: workflow steps see the sandbox image's OS
(not the host), cannot read the server's data directory, cannot reach the
network, and do not receive the server's environment.

Regardless of mode, do not run untrusted workflows on an instance whose Docker
daemon or host you cannot afford to have abused; when in doubt, use `docker`
mode (or dedicated, isolated self-hosted runners) and keep the daemon locked
down.

### Secrets

- Action secrets and webhook signing secrets are encrypted at rest using
  `WEBHOOK_SECRET_KEY`. Set it to a strong, random value and keep it stable
  across restarts (rotating it makes existing secrets undecryptable).
- `JWT_SECRET` signs authentication tokens; use a long random value and keep it
  secret.
- OAuth client secrets are stored only as SHA-256 hashes and are shown to the
  user exactly once (on creation and regeneration).

### Transport

- Use `TLS_MODE=acme` or `TLS_MODE=custom` in production. `selfsigned` is for
  local development only.
- Git credentials (personal access tokens) are sent over HTTP Basic auth for the
  Git-over-HTTP protocol, so HTTPS is required to protect them in transit.

### Authentication and authorization

- Access is enforced per repository (owner, organization membership, and
  collaborator permissions) and per token scope. Private repositories are not
  served to unauthenticated requests.
- Prefer SSH keys or scoped personal access tokens over long-lived broad-scope
  tokens.

## Supported versions

This project is pre-1.0 and moves quickly; security fixes are applied to the
`main` branch. Pin to a commit you have reviewed for production use.
