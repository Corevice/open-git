CREATE TABLE action_verifications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL,
    "trigger" varchar NOT NULL,
    status varchar NOT NULL,
    requested_by uuid REFERENCES users(id),
    started_at timestamptz,
    finished_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_action_verifications_organization_id ON action_verifications(organization_id);

CREATE TABLE action_compatibility_results (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL,
    repository_id uuid REFERENCES repositories(id),
    action_name varchar NOT NULL,
    action_version varchar NOT NULL,
    status varchar NOT NULL DEFAULT 'untested',
    note text,
    golden_diff jsonb,
    verified_at timestamptz,
    verification_id uuid REFERENCES action_verifications(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(organization_id, action_name, action_version)
);

CREATE INDEX idx_action_compatibility_results_organization_id ON action_compatibility_results(organization_id);

CREATE TABLE action_cache_entries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL,
    action_name varchar NOT NULL,
    resolved_ref varchar NOT NULL,
    storage_path varchar NOT NULL,
    cached_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(organization_id, action_name, resolved_ref)
);

CREATE INDEX idx_action_cache_entries_organization_id ON action_cache_entries(organization_id);
