CREATE TABLE media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id),
    post_id UUID REFERENCES posts (id),
    bucket VARCHAR(63) NOT NULL,
    object_key TEXT NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_media_post ON media (post_id) WHERE post_id IS NOT NULL;
CREATE INDEX idx_media_user ON media (user_id);
