CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts (id),
    user_id UUID NOT NULL REFERENCES users (id),
    parent_id UUID REFERENCES comments (id),
    content VARCHAR(500) NOT NULL,
    likes_count INT NOT NULL DEFAULT 0 CHECK (likes_count >= 0),
    depth INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT comments_max_depth CHECK (depth <= 5)
);

CREATE INDEX idx_comments_post_root ON comments (post_id, created_at ASC) WHERE deleted_at IS NULL AND parent_id IS NULL;
CREATE INDEX idx_comments_parent ON comments (parent_id, created_at ASC) WHERE deleted_at IS NULL AND parent_id IS NOT NULL;
CREATE INDEX idx_comments_user ON comments (user_id) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_comments_updated_at
    BEFORE UPDATE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
