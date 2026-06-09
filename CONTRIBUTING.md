# Contributing to open-git

Thank you for your interest in contributing. This guide covers local setup, testing, and the pull request workflow.

## Development Setup

The fastest way to get a full stack running locally is Docker Compose:

```bash
docker compose up -d
```

This starts the API, web frontend, and supporting services. Adjust environment variables via a local `.env` file (never commit secrets).

For component-only development, install dependencies in the repo root and in `backend/` as needed, then run the web and API processes separately.

## Running Tests

Run the full test suite from the repository root:

```bash
task test
```

You can also run tests individually:

```bash
# Frontend (Vitest / typecheck)
npm test
npm run lint

# Backend (Go)
cd backend && go test ./...
```

## Commit Convention

This project follows [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` — new feature
- `fix:` — bug fix
- `docs:` — documentation only
- `chore:` — tooling, CI, or maintenance
- `test:` — adding or updating tests
- `refactor:` — code change that neither fixes a bug nor adds a feature

Example: `feat: add repository webhook retry logic`

## Pull Request Process

1. **Branch naming** — use descriptive prefixes such as `feat/`, `fix/`, or `chore/` (e.g. `feat/webhook-retries`).
2. **Keep changes focused** — one logical change per PR when possible.
3. **Required checks** — all CI checks must pass before merge (lint, typecheck, tests, and build as applicable to changed paths).
4. **Review** — address maintainer feedback; CODEOWNERS will be requested automatically.
5. **Merge** — squash or merge per repository policy once approved and green.
