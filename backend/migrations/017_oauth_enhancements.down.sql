DROP INDEX IF EXISTS idx_oauth_grants_user_id;
DROP TABLE IF EXISTS oauth_grants;

ALTER TABLE access_tokens DROP COLUMN IF EXISTS oauth_application_id;
ALTER TABLE access_tokens DROP COLUMN IF EXISTS token_last_eight;
ALTER TABLE access_tokens DROP COLUMN IF EXISTS token_prefix;
ALTER TABLE access_tokens DROP COLUMN IF EXISTS last_used_at;
ALTER TABLE access_tokens DROP COLUMN IF EXISTS note;

ALTER TABLE oauth_apps ALTER COLUMN homepage_url DROP DEFAULT;
ALTER TABLE oauth_apps ALTER COLUMN homepage_url DROP NOT NULL;
ALTER TABLE oauth_apps DROP COLUMN IF EXISTS homepage_url;
ALTER TABLE oauth_apps ALTER COLUMN name DROP DEFAULT;
ALTER TABLE oauth_apps ALTER COLUMN name DROP NOT NULL;
ALTER TABLE oauth_apps DROP COLUMN IF EXISTS name;
