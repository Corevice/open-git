CREATE TABLE organizations (
    id TEXT PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    plan_tier TEXT NOT NULL DEFAULT 'free',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE memberships (
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    PRIMARY KEY (organization_id, user_id)
);

CREATE INDEX idx_memberships_organization_id ON memberships(organization_id);

CREATE TABLE repositories (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    owner_id TEXT NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'internal', 'public')),
    default_branch TEXT NOT NULL DEFAULT 'main',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(owner_id, name)
);

CREATE INDEX idx_repositories_organization_id ON repositories(organization_id);

CREATE TABLE access_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    token_hash TEXT NOT NULL,
    scopes TEXT NOT NULL DEFAULT '[]',
    expires_at TIMESTAMP,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE oauth_apps (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES users(id),
    client_id TEXT NOT NULL UNIQUE,
    client_secret_hash TEXT NOT NULL,
    redirect_uris TEXT NOT NULL DEFAULT '[]',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE issues (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    number INTEGER NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL DEFAULT 'open',
    author_id TEXT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(repository_id, number)
);

CREATE INDEX idx_issues_organization_id ON issues(organization_id);

CREATE TABLE pull_requests (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    number INTEGER NOT NULL,
    head_ref TEXT NOT NULL,
    base_ref TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'open',
    merged_at TIMESTAMP,
    author_id TEXT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(repository_id, number)
);

CREATE INDEX idx_pull_requests_organization_id ON pull_requests(organization_id);

CREATE TABLE comments (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    issue_id TEXT NOT NULL REFERENCES issues(id),
    author_id TEXT NOT NULL REFERENCES users(id),
    body TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_comments_organization_id ON comments(organization_id);

CREATE TABLE labels (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    name TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(repository_id, name)
);

CREATE INDEX idx_labels_organization_id ON labels(organization_id);

CREATE TABLE milestones (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL DEFAULT 'open',
    due_on TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_milestones_organization_id ON milestones(organization_id);

CREATE TABLE reviews (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    pull_request_id TEXT NOT NULL REFERENCES pull_requests(id),
    reviewer_id TEXT NOT NULL REFERENCES users(id),
    state TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    submitted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_reviews_organization_id ON reviews(organization_id);

CREATE TABLE branch_protections (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    pattern TEXT NOT NULL,
    required_reviews INTEGER NOT NULL DEFAULT 0,
    required_checks TEXT NOT NULL DEFAULT '[]',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(repository_id, pattern)
);

CREATE INDEX idx_branch_protections_organization_id ON branch_protections(organization_id);

CREATE TABLE webhooks (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    url TEXT NOT NULL,
    secret_ref TEXT NOT NULL DEFAULT '',
    events TEXT NOT NULL DEFAULT '[]',
    active INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_webhooks_organization_id ON webhooks(organization_id);

CREATE TABLE workflow_runs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    workflow TEXT NOT NULL,
    status TEXT NOT NULL,
    conclusion TEXT,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX idx_workflow_runs_organization_id ON workflow_runs(organization_id);

CREATE TABLE secrets (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    name TEXT NOT NULL,
    encrypted_value TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(repository_id, name)
);

CREATE INDEX idx_secrets_organization_id ON secrets(organization_id);

CREATE TABLE audit_logs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    actor_id TEXT NOT NULL REFERENCES users(id),
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_organization_id ON audit_logs(organization_id);
