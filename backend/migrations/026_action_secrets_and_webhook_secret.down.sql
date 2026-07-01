DROP TABLE IF EXISTS action_secret_repositories;
DROP INDEX IF EXISTS idx_action_secrets_org_repo_name;
DROP TABLE IF EXISTS action_secrets;
ALTER TABLE webhooks DROP COLUMN secret_encrypted;
