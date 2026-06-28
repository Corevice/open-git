-- SQLite does not support DROP COLUMN; workflow_runs columns added in the up
-- migration (run_number, name, event, head_branch, head_sha, actor_id) and
-- workflow_jobs columns (started_at, runner_name, runner_labels) cannot be
-- reverted here.

DROP TABLE IF EXISTS artifacts;
DROP TABLE IF EXISTS workflow_steps;
