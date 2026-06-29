-- SQLite does not support DROP COLUMN; workflow_runs columns added in the up
-- migration (run_number, name, event, head_branch, head_sha, actor_id) cannot
-- be reverted here.
-- workflow_jobs is managed by migration 013_workflow_jobs

DROP TABLE IF EXISTS artifacts;
DROP TABLE IF EXISTS workflow_steps;
