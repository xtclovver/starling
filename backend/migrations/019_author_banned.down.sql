DROP INDEX IF EXISTS idx_posts_author_banned;
ALTER TABLE posts DROP COLUMN IF EXISTS author_banned;
