ALTER TABLE posts ADD COLUMN media_url TEXT NOT NULL DEFAULT '';

UPDATE posts p SET media_url = (
    SELECT CONCAT('http://localhost:9000/', m.bucket, '/', m.object_key)
    FROM media m
    WHERE m.post_id = p.id
    ORDER BY m.position
    LIMIT 1
);

ALTER TABLE media DROP CONSTRAINT IF EXISTS uq_media_post_position;
ALTER TABLE media DROP COLUMN IF EXISTS position;
ALTER TABLE posts DROP COLUMN IF EXISTS views_count;
