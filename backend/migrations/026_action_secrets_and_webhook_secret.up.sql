-- Webhooks: the repository reads and writes an encrypted signing secret in a
-- column named secret_encrypted, but the initial schema only ever created an
-- unused secret_ref column, so every hook create/list failed with
-- "no such column: secret_encrypted". Add the nullable byte column the code
-- actually uses (NULL when the hook has no secret). The legacy secret_ref
-- column is left in place to keep this change non-destructive.
ALTER TABLE webhooks ADD COLUMN secret_encrypted BYTEA;

-- Action secrets (repository- and organization-scoped CI secrets) were fully
-- implemented in the repository, usecases, handler and tests, but no migration
-- ever created their tables, so every /actions/secrets request failed with
-- "no such table: action_secrets". Create them here.
--
-- No foreign keys on organization_id: personal repositories use the owner's
-- user id as the organization id and have no organizations row, so a FK would
-- reject personal-repo secrets. This mirrors the schema the repository tests
-- exercise.
CREATE TABLE action_secrets (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    repository_id TEXT,
    name TEXT NOT NULL,
    encrypted_value BYTEA NOT NULL,
    key_id TEXT NOT NULL DEFAULT '',
    visibility TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- COALESCE (rather than SQLite-only IFNULL) so the expression is portable to
-- Postgres, and so organization-level secrets (repository_id IS NULL) are
-- de-duplicated by name. The repository's upsert targets this same expression.
CREATE UNIQUE INDEX idx_action_secrets_org_repo_name
    ON action_secrets (organization_id, COALESCE(repository_id, ''), name);

CREATE TABLE action_secret_repositories (
    secret_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    PRIMARY KEY (secret_id, repository_id)
);
