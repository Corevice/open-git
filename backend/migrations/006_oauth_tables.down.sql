DROP TABLE IF EXISTS oauth_authorizations;
DROP TABLE IF EXISTS oauth_access_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;

-- SQLite does not support DROP COLUMN before version 3.35.
-- For rollback on SQLite, recreate oauth_apps without the added columns instead.
-- PostgreSQL column removal:
ALTER TABLE oauth_apps DROP COLUMN name;
ALTER TABLE oauth_apps DROP COLUMN homepage_url;
ALTER TABLE oauth_apps DROP COLUMN owner_type;
ALTER TABLE oauth_apps DROP COLUMN owner_user_id;
ALTER TABLE oauth_apps DROP COLUMN organization_id;
ALTER TABLE oauth_apps DROP COLUMN updated_at;
