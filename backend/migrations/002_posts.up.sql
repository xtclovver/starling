CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id),
    content VARCHAR(280) NOT NULL,
    media_url TEXT NOT NULL DEFAULT '',
    likes_count INT NOT NULL DEFAULT 0 CHECK (likes_count >= 0),
    comments_count INT NOT NULL DEFAULT 0 CHECK (comments_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_posts_user_created ON posts (user_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_created_at ON posts (created_at DESC) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_posts_updated_at
    BEFORE UPDATE ON posts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
