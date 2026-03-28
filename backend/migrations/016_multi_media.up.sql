-- Add position to media table for ordering within a post
ALTER TABLE media ADD COLUMN position INT NOT NULL DEFAULT 0;

-- Add unique constraint to prevent duplicate positions per post
ALTER TABLE media ADD CONSTRAINT uq_media_post_position UNIQUE (post_id, position);

-- Add views_count to posts
ALTER TABLE posts ADD COLUMN views_count INT NOT NULL DEFAULT 0;

-- Migrate existing media_url data into media table
INSERT INTO media (user_id, post_id, bucket, object_key, content_type, position)
SELECT
    p.user_id,
    p.id,
    'media',
    SUBSTRING(p.media_url FROM '/media/(.+)$'),
    'image/jpeg',
    0
FROM posts p
WHERE p.media_url IS NOT NULL
  AND p.media_url != ''
  AND p.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM media m WHERE m.post_id = p.id);

-- Drop media_url from posts
ALTER TABLE posts DROP COLUMN media_url;
