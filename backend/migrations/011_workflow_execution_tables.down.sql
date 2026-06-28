DROP TABLE action_secrets;
DROP TABLE artifacts;
DROP TABLE job_logs;
DROP TABLE workflow_steps;
DROP TABLE workflow_jobs;
ALTER TABLE workflow_runs DROP COLUMN IF EXISTS triggered_by_user_id, DROP COLUMN IF EXISTS event, DROP COLUMN IF EXISTS run_attempt, DROP COLUMN IF EXISTS run_number, DROP COLUMN IF EXISTS head_branch, DROP COLUMN IF EXISTS head_sha;
