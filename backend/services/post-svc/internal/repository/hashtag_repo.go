package repository

import (
	"context"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usedcvnt/microtwitter/post-svc/internal/model"
)

var hashtagRegex = regexp.MustCompile(`#(\w+)`)

type HashtagRepository interface {
	UpsertAndLink(ctx context.Context, postID string, tags []string) error
	UnlinkAll(ctx context.Context, postID string) error
	GetPostsByHashtag(ctx context.Context, tag, cursor string, limit int) ([]model.Post, string, bool, error)
	GetTrending(ctx context.Context, limit int) ([]model.TrendingHashtag, error)
	GetTagsByPostID(ctx context.Context, postID string) ([]string, error)
	GetTagsByPostIDs(ctx context.Context, postIDs []string) (map[string][]string, error)
}

type hashtagRepo struct {
	pool *pgxpool.Pool
}

func NewHashtagRepository(pool *pgxpool.Pool) HashtagRepository {
	return &hashtagRepo{pool: pool}
}

func ExtractHashtags(content string) []string {
	matches := hashtagRegex.FindAllStringSubmatch(content, -1)
	seen := make(map[string]struct{})
	tags := make([]string, 0, len(matches))
	for _, m := range matches {
		tag := strings.ToLower(m[1])
		if _, ok := seen[tag]; !ok {
			seen[tag] = struct{}{}
			tags = append(tags, tag)
		}
	}
	return tags
}

func (r *hashtagRepo) UpsertAndLink(ctx context.Context, postID string, tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, tag := range tags {
		var hashtagID string
		err := tx.QueryRow(ctx,
			`INSERT INTO hashtags (tag) VALUES ($1)
			 ON CONFLICT (tag) DO UPDATE SET tag = EXCLUDED.tag
			 RETURNING id`,
			tag,
		).Scan(&hashtagID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO post_hashtags (post_id, hashtag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			postID, hashtagID,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *hashtagRepo) UnlinkAll(ctx context.Context, postID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM post_hashtags WHERE post_id = $1`, postID)
	return err
}

func (r *hashtagRepo) GetPostsByHashtag(ctx context.Context, tag, cursor string, limit int) ([]model.Post, string, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args := []any{strings.ToLower(tag), limit + 1}
	q := `SELECT p.id, p.user_id, p.content, p.views_count, p.likes_count, p.comments_count, p.reposts_count, p.created_at, p.updated_at, p.edited_at
		  FROM posts p
		  INNER JOIN post_hashtags ph ON ph.post_id = p.id
		  INNER JOIN hashtags h ON h.id = ph.hashtag_id AND h.tag = $1
		  WHERE p.deleted_at IS NULL`

	if cursor != "" {
		q += ` AND (p.created_at, p.id) < (
			(SELECT created_at FROM posts WHERE id = $3),
			$3
		)`
		args = append(args, cursor)
	}

	q += ` ORDER BY p.created_at DESC, p.id DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	posts := make([]model.Post, 0)
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Content, &p.ViewsCount, &p.LikesCount, &p.CommentsCount, &p.RepostsCount, &p.CreatedAt, &p.UpdatedAt, &p.EditedAt); err != nil {
			return nil, "", false, err
		}
		posts = append(posts, p)
	}

	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}

	var nextCursor string
	if hasMore {
		nextCursor = posts[limit-1].ID
	}

	return posts, nextCursor, hasMore, nil
}

func (r *hashtagRepo) GetTrending(ctx context.Context, limit int) ([]model.TrendingHashtag, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	rows, err := r.pool.Query(ctx,
		`SELECT h.tag, COUNT(*) as post_count
		 FROM post_hashtags ph
		 INNER JOIN hashtags h ON h.id = ph.hashtag_id
		 INNER JOIN posts p ON p.id = ph.post_id AND p.deleted_at IS NULL
		 WHERE ph.created_at > NOW() - INTERVAL '7 days'
		 GROUP BY h.tag
		 ORDER BY post_count DESC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trending []model.TrendingHashtag
	for rows.Next() {
		var t model.TrendingHashtag
		if err := rows.Scan(&t.Tag, &t.PostCount); err != nil {
			return nil, err
		}
		trending = append(trending, t)
	}
	return trending, nil
}

func (r *hashtagRepo) GetTagsByPostID(ctx context.Context, postID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT h.tag FROM hashtags h
		 INNER JOIN post_hashtags ph ON ph.hashtag_id = h.id
		 WHERE ph.post_id = $1
		 ORDER BY h.tag`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (r *hashtagRepo) GetTagsByPostIDs(ctx context.Context, postIDs []string) (map[string][]string, error) {
	if len(postIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT ph.post_id, h.tag FROM hashtags h
		 INNER JOIN post_hashtags ph ON ph.hashtag_id = h.id
		 WHERE ph.post_id = ANY($1)
		 ORDER BY ph.post_id, h.tag`,
		postIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var postID, tag string
		if err := rows.Scan(&postID, &tag); err != nil {
			return nil, err
		}
		result[postID] = append(result[postID], tag)
	}
	return result, nil
}
