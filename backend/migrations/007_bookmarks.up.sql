CREATE TABLE bookmarks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    post_id UUID NOT NULL REFERENCES posts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT bookmarks_unique UNIQUE (user_id, post_id)
);
CREATE INDEX idx_bookmarks_user_created ON bookmarks (user_id, created_at DESC);
