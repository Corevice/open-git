-- SQLite does not support DROP COLUMN; workflow_runs columns added in the up
-- migration (run_number, name, event, head_branch, head_sha, actor_id) cannot
-- be reverted here.
-- Migration 018 (task spec ref 011): workflow run history schema extension.

DROP INDEX IF EXISTS idx_artifacts_run_id;
DROP TABLE IF EXISTS artifacts;

DROP INDEX IF EXISTS idx_workflow_steps_job_id;
DROP TABLE IF EXISTS workflow_steps;

DROP INDEX IF EXISTS idx_workflow_jobs_org_run;
DROP TABLE IF EXISTS workflow_jobs;
