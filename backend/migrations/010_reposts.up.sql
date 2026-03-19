CREATE TABLE reposts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    post_id UUID NOT NULL REFERENCES posts(id),
    quote_content VARCHAR(280),
    type VARCHAR(10) NOT NULL CHECK (type IN ('repost', 'quote')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_reposts_simple_unique ON reposts (user_id, post_id) WHERE type = 'repost';
CREATE INDEX idx_reposts_user_created ON reposts (user_id, created_at DESC);
CREATE INDEX idx_reposts_post ON reposts (post_id);

ALTER TABLE posts ADD COLUMN reposts_count INT NOT NULL DEFAULT 0 CHECK (reposts_count >= 0);
