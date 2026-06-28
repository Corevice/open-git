DROP INDEX IF EXISTS idx_oauth_grants_user_id;
DROP TABLE IF EXISTS oauth_grants;

ALTER TABLE access_tokens DROP COLUMN oauth_application_id;
ALTER TABLE access_tokens DROP COLUMN token_last_eight;
ALTER TABLE access_tokens DROP COLUMN token_prefix;
ALTER TABLE access_tokens DROP COLUMN last_used_at;
ALTER TABLE access_tokens DROP COLUMN note;

ALTER TABLE oauth_apps DROP COLUMN homepage_url;
ALTER TABLE oauth_apps DROP COLUMN name;
