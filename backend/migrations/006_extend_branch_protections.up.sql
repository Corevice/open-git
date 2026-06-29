-- Migration numbering: prior migrations in this repo (verified on disk):
--   001_initial_schema
--   002_webhook_delivery_status
--   003_user_profile_fields
--   004_org_description
--   005_repository_fields
-- This is the next sequential migration (006). Task docs may reference "002" for
-- this feature slice, but 002 is occupied by webhook_delivery_status; golang-migrate
-- requires monotonic filenames — do not reuse 002.
-- Target DB: SQLite only (see down migration for DROP COLUMN limitation).
--
-- NOTE: SQLite does not support ADD COLUMN IF NOT EXISTS. Re-running this migration
-- after partial failure will error on duplicate columns; down migration cannot remove
-- those columns (see down.sql).

ALTER TABLE branch_protections ADD COLUMN dismiss_stale_reviews INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN require_code_owner_reviews INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN required_status_checks_strict INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN enforce_admins INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN allow_force_pushes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN allow_deletions INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN required_linear_history INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN required_conversation_resolution INTEGER NOT NULL DEFAULT 0;
ALTER TABLE branch_protections ADD COLUMN updated_at TIMESTAMP;

-- organization_id is already indexed by 001_initial_schema (idx_branch_protections_organization_id).
CREATE INDEX IF NOT EXISTS idx_bp_org_repo ON branch_protections (organization_id, repository_id);

-- id must be supplied by the application layer as a UUID (SQLite has no native UUID default).
-- length(id) <= 64 is a generous upper bound (2× standard UUID length) to allow prefixed
-- hex IDs; standard UUIDv4 (36 chars) fits comfortably. Non-UUID IDs longer than 64 chars
-- are rejected here — widen the CHECK via a follow-up migration if a new ID format is adopted.
-- context length is capped to mitigate DoS via oversized check names.
-- Tenant isolation: organization_id lives on branch_protections (via rule_id FK). Application
-- queries must JOIN branch_protections to scope by tenant; a direct organization_id column
-- here would duplicate data and is omitted intentionally.
CREATE TABLE branch_protection_required_checks (
    id TEXT PRIMARY KEY CHECK(
        length(id) >= 1 AND length(id) <= 64
        AND length(lower(id)) = 36
        AND lower(id) GLOB '[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]-[0-9a-f][0-9a-f][0-9a-f][0-9a-f]-[0-9a-f][0-9a-f][0-9a-f][0-9a-f]-[0-9a-f][0-9a-f][0-9a-f][0-9a-f]-[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]'
    ),
    rule_id TEXT NOT NULL REFERENCES branch_protections(id) ON DELETE CASCADE,
    context TEXT NOT NULL CHECK(length(context) >= 1 AND length(context) <= 255),
    UNIQUE(rule_id, context)
);
