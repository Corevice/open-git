-- Migration numbering: 001–005 already exist (initial_schema, webhook_delivery_status,
-- user_profile_fields, org_description, repository_fields). This is the next sequential
-- migration (006). Task docs may reference "002" for this feature slice, but 002 is
-- occupied; golang-migrate requires monotonic filenames — do not reuse 002.
-- Target DB: SQLite only (see down migration for DROP COLUMN limitation).

ALTER TABLE branch_protections ADD COLUMN dismiss_stale_reviews INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN require_code_owner_reviews INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN required_status_checks_strict INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN enforce_admins INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN allow_force_pushes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN allow_deletions INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN required_linear_history INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN required_conversation_resolution INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN updated_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_bp_organization_id ON branch_protections (organization_id);
CREATE INDEX IF NOT EXISTS idx_bp_org_repo ON branch_protections (organization_id, repository_id);

-- id must be supplied by the application layer as a UUID (SQLite has no native UUID default).
-- context length is capped to mitigate DoS via oversized check names.
CREATE TABLE branch_protection_required_checks (
    id TEXT PRIMARY KEY CHECK(length(id) >= 1 AND length(id) <= 64),
    rule_id TEXT NOT NULL REFERENCES branch_protections(id) ON DELETE CASCADE,
    context TEXT NOT NULL CHECK(length(context) >= 1 AND length(context) <= 255),
    UNIQUE(rule_id, context)
);
