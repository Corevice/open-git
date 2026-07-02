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

### CI / Actions execution is not sandboxed

Workflow steps defined under `.github/workflows` are executed with `sh -c` in the
API server's own process and host (`backend/internal/worker/ci_worker.go`). There
is no container, VM, or user isolation between a workflow and the server.

Implications:

- Anyone able to push a workflow to a repository that the instance builds can run
  arbitrary commands on the server, with the server process's privileges and
  access (including the filesystem and any credentials it can read).
- This is acceptable for instances where all users who can push are trusted (for
  example a personal instance, or an internal team instance behind
  authentication).
- **Do not enable CI on a multi-tenant or public instance** without first
  isolating execution — for example by running jobs in ephemeral containers, as a
  dedicated unprivileged user, in a disposable VM, or exclusively on separate
  self-hosted runners that you control and isolate.

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
