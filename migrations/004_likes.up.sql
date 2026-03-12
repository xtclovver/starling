CREATE TABLE likes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id),
    post_id UUID REFERENCES posts (id),
    comment_id UUID REFERENCES comments (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT likes_target_xor CHECK (
        (post_id IS NOT NULL AND comment_id IS NULL) OR
        (post_id IS NULL AND comment_id IS NOT NULL)
    ),
    CONSTRAINT likes_post_uniq UNIQUE (user_id, post_id),
    CONSTRAINT likes_comment_uniq UNIQUE (user_id, comment_id)
);

CREATE INDEX idx_likes_post_id ON likes (post_id) WHERE post_id IS NOT NULL;
CREATE INDEX idx_likes_comment_id ON likes (comment_id) WHERE comment_id IS NOT NULL;
