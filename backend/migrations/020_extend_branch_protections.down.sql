DROP TABLE branch_protection_required_checks;
DROP INDEX idx_bp_org_repo;

ALTER TABLE branch_protections DROP COLUMN updated_at;
ALTER TABLE branch_protections DROP COLUMN required_conversation_resolution;
ALTER TABLE branch_protections DROP COLUMN required_linear_history;
ALTER TABLE branch_protections DROP COLUMN allow_deletions;
ALTER TABLE branch_protections DROP COLUMN allow_force_pushes;
ALTER TABLE branch_protections DROP COLUMN enforce_admins;
ALTER TABLE branch_protections DROP COLUMN required_status_checks_strict;
ALTER TABLE branch_protections DROP COLUMN require_code_owner_reviews;
ALTER TABLE branch_protections DROP COLUMN dismiss_stale_reviews;
