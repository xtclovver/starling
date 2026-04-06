ALTER TABLE posts ADD COLUMN author_banned BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_posts_author_banned ON posts (author_banned) WHERE author_banned = TRUE;
