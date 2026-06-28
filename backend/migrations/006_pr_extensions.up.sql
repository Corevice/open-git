ALTER TABLE pull_requests ADD COLUMN title TEXT;
ALTER TABLE pull_requests ADD COLUMN body TEXT;
UPDATE pull_requests SET title = '' WHERE title IS NULL;
UPDATE pull_requests SET body = '' WHERE body IS NULL;
ALTER TABLE pull_requests ALTER COLUMN title SET NOT NULL;
ALTER TABLE pull_requests ALTER COLUMN body SET NOT NULL;
