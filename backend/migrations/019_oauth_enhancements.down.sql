DROP INDEX IF EXISTS idx_oauth_grants_user_id;
DROP TABLE IF EXISTS oauth_grants;

ALTER TABLE oauth_apps DROP COLUMN homepage_url;
ALTER TABLE oauth_apps DROP COLUMN name;

ALTER TABLE access_tokens DROP COLUMN oauth_application_id;
ALTER TABLE access_tokens DROP COLUMN token_last_eight;
ALTER TABLE access_tokens DROP COLUMN token_prefix;
