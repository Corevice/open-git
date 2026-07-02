# open-git

A self-hostable Git hosting platform — a lightweight, open-source alternative to
GitHub-style services. It serves Git over HTTP and SSH, and adds the collaboration
layer on top: repositories, issues, pull requests, code review, organizations,
webhooks, OAuth apps, and a built-in CI/Actions system.

The backend is a Go (Echo) API; the frontend is a Next.js 15 (App Router,
TypeScript, Tailwind) web app.

> **Status:** functional and self-hostable. Core Git hosting, issues/PRs, orgs,
> webhooks, secrets, OAuth apps and CI have been exercised end-to-end. Before
> exposing an instance to untrusted users, read [Security](#security) — in
> particular, CI jobs currently run directly on the server host without
> sandboxing.

## Features

- **Git hosting** over HTTP(S) and SSH with authentication and per-repository
  authorization (owner, organization members, collaborators).
- **Repositories**: create/browse/manage, file and tree browsing, commit history,
  blob viewing with syntax highlighting, branch management and branch protection.
- **Collaboration**: issues, labels, milestones, pull requests with reviews and
  merge (merge/squash), and repository/organization audit logs.
- **Organizations**: orgs, memberships/roles, org-owned repositories.
- **Access**: username/password auth, personal access tokens (scoped), SSH keys,
  and OAuth apps (authorize/token flow + app management).
- **Webhooks** and **Actions secrets** (encrypted at rest).
- **CI / Actions**: workflows under `.github/workflows` run on push or manual
  dispatch; per-run/-job status, live logs, and self-hosted runner registration.
- **User preferences**, contributor stats, and a REST API under `/api/v1` and
  `/api/v3`.

## Architecture

| Component      | Stack                                 | Location                                          |
| -------------- | ------------------------------------- | ------------------------------------------------- |
| API server     | Go + Echo, sqlx                       | `backend/` (module `github.com/open-git/backend`) |
| Web frontend   | Next.js 15, React 19, Tailwind        | repository root                                   |
| Database       | SQLite (default) or PostgreSQL        | migrations in `backend/migrations`                |
| Object storage | filesystem or S3/MinIO (CI artifacts) | optional                                          |
| Queue/cache    | in-process, or Redis when configured  | optional                                          |

The API serves both the REST API and the Git Smart HTTP endpoints; SSH is served
on a separate port. The frontend is a standalone Next.js app that talks to the
API via `NEXT_PUBLIC_API_BASE_URL` (and can proxy `/api/*` to it — see
`next.config.mjs`).

## Quick start

### Option A — Docker Compose (full stack)

```bash
docker compose -f docker-compose.dev.yml up --build
```

This brings up the frontend (`:3000`) and API (`:8080`). The production compose
file (`docker-compose.yml`) additionally wires PostgreSQL, Redis, MinIO, Traefik
(TLS) and observability, and expects `DOMAIN` / TLS configuration.

### Option B — Run the two processes directly

Prerequisites: Go (see `backend/go.mod`), Node.js 20+, and `git` on `PATH`.

Backend (from `backend/`, SQLite, no external services):

```bash
cd backend
DB_TYPE=sqlite \
DB_DSN=./data/og.db \
DB_AUTO_MIGRATE=true \
JWT_SECRET="change-me-to-a-long-random-string" \
WEBHOOK_SECRET_KEY="0000000000000000000000000000000000000000000000000000000000000000" \
GIT_DATA_ROOT=./data/git \
TLS_MODE=selfsigned \
PORT=8080 \
SSH_ENABLED=true SSH_PORT=2222 \
go run ./cmd/server
```

> Run the server from the `backend/` directory: migrations are loaded from the
> relative `./migrations` path.

Frontend (from the repository root):

```bash
npm install
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080 npm run dev
```

Open http://localhost:3000, create an account, and you're ready to create a
repository and push to it.

### Pushing your first repository

```bash
# Create a personal access token in the UI (Settings → tokens), then:
git remote add origin http://<user>:<token>@localhost:8080/<user>/<repo>
git push -u origin main

# Or over SSH after adding your public key in the UI:
git remote add origin ssh://git@localhost:2222/<user>/<repo>
git push -u origin main
```

## Configuration

Configuration is via environment variables. The most important ones:

| Variable                        | Default          | Notes                                                        |
| ------------------------------- | ---------------- | ------------------------------------------------------------ |
| `DB_TYPE`                       | `sqlite`         | `sqlite` or `postgres`                                       |
| `DB_DSN`                        | —                | required for `postgres`; file path for `sqlite`              |
| `DB_AUTO_MIGRATE`               | `false`          | run migrations on startup                                    |
| `JWT_SECRET`                    | —                | **required**; signs session tokens                           |
| `WEBHOOK_SECRET_KEY`            | —                | hex key used to encrypt secrets at rest                      |
| `GIT_DATA_ROOT`                 | `./data/git`     | where bare repositories live                                 |
| `PORT`                          | `8080`           | HTTP/API port                                                |
| `SSH_ENABLED` / `SSH_PORT`      | `false` / `2222` | Git over SSH                                                 |
| `TLS_MODE`                      | `acme`           | `acme`, `custom`, or `selfsigned` (use `selfsigned` locally) |
| `REDIS_ADDR`                    | —                | optional; enables Redis-backed queue/cache                   |
| `MINIO_ENDPOINT`                | —                | optional; S3/MinIO for CI artifacts                          |
| `WEB_BASE_URL` / `API_BASE_URL` | localhost        | public URLs                                                  |

On the frontend, `NEXT_PUBLIC_API_BASE_URL` must point at the API (it is a
build-time value for the standalone image).

## Development

The repository uses [Task](https://taskfile.dev):

```bash
task dev          # run frontend + backend
task test         # run all tests
task lint         # lint everything
task migrate:up   # apply migrations
```

Or per component:

```bash
# Backend
cd backend && go build ./... && go vet ./... && go test ./...

# Frontend
npm run lint && npx tsc --noEmit && npm test && npm run build
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full workflow.

## Security

This project is suitable for **self-hosting for yourself or a trusted team**.
Before exposing an instance to untrusted users, be aware of the following:

- **CI jobs are not sandboxed.** Workflow steps under `.github/workflows` run as
  `sh -c` in the API server's own process/host. Anyone who can push a workflow to
  a repository the instance builds can therefore run arbitrary commands on the
  server. For single-user or trusted-team instances this is equivalent to running
  your own scripts; for multi-tenant or public instances you must isolate
  execution (containers, a dedicated unprivileged user/VM, or dedicated runners)
  before enabling CI. See `backend/internal/worker/ci_worker.go`.
- **Set a strong `JWT_SECRET` and `WEBHOOK_SECRET_KEY`.** Secrets and webhook
  signing secrets are encrypted at rest with the latter.
- **Use TLS in production** (`TLS_MODE=acme` or `custom`); `selfsigned` is for
  local use only.
- Most testing to date has been on SQLite. The schema is written to be
  PostgreSQL-compatible, but exercise your target database before relying on it.

To report a vulnerability, please see [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE).
