DROP INDEX IF EXISTS idx_repo_collaborators_repository_id;
DROP TABLE IF EXISTS repository_collaborators;

DROP INDEX IF EXISTS idx_ssh_keys_user_id;
DROP TABLE IF EXISTS ssh_keys;

ALTER TABLE repositories DROP COLUMN description;
ALTER TABLE repositories DROP COLUMN disk_path;
ALTER TABLE repositories DROP COLUMN is_empty;
