package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usedcvnt/microtwitter/post-svc/internal/model"
)

var ErrAlreadyBookmarked = errors.New("already bookmarked")

type BookmarkRepository interface {
	Bookmark(ctx context.Context, postID, userID string) error
	Unbookmark(ctx context.Context, postID, userID string) error
	IsBookmarked(ctx context.Context, postID, userID string) (bool, error)
	AreBookmarked(ctx context.Context, postIDs []string, userID string) (map[string]bool, error)
	GetByUser(ctx context.Context, userID, cursor string, limit int) ([]model.Post, string, bool, error)
}

type bookmarkRepo struct {
	pool *pgxpool.Pool
}

func NewBookmarkRepository(pool *pgxpool.Pool) BookmarkRepository {
	return &bookmarkRepo{pool: pool}
}

func (r *bookmarkRepo) Bookmark(ctx context.Context, postID, userID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO bookmarks (user_id, post_id) VALUES ($1, $2)`,
		userID, postID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyBookmarked
		}
		return err
	}
	return nil
}

func (r *bookmarkRepo) Unbookmark(ctx context.Context, postID, userID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM bookmarks WHERE user_id = $1 AND post_id = $2`,
		userID, postID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *bookmarkRepo) IsBookmarked(ctx context.Context, postID, userID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM bookmarks WHERE user_id = $1 AND post_id = $2)`,
		userID, postID,
	).Scan(&exists)
	return exists, err
}

func (r *bookmarkRepo) AreBookmarked(ctx context.Context, postIDs []string, userID string) (map[string]bool, error) {
	if len(postIDs) == 0 || userID == "" {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT post_id FROM bookmarks WHERE user_id = $1 AND post_id = ANY($2)`,
		userID, postIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var postID string
		if err := rows.Scan(&postID); err != nil {
			return nil, err
		}
		result[postID] = true
	}
	return result, nil
}

func (r *bookmarkRepo) GetByUser(ctx context.Context, userID, cursor string, limit int) ([]model.Post, string, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args := []any{userID, limit + 1}
	q := `SELECT p.id, p.user_id, p.content, p.media_url, p.likes_count, p.comments_count, p.reposts_count, p.created_at, p.updated_at, p.edited_at
		  FROM posts p
		  INNER JOIN bookmarks b ON b.post_id = p.id AND b.user_id = $1
		  WHERE p.deleted_at IS NULL`

	if cursor != "" {
		q += ` AND (b.created_at, p.id) < (
			(SELECT b2.created_at FROM bookmarks b2 INNER JOIN posts p2 ON p2.id = b2.post_id WHERE p2.id = $3 AND b2.user_id = $1),
			$3
		)`
		args = append(args, cursor)
	}

	q += ` ORDER BY b.created_at DESC, p.id DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	posts := make([]model.Post, 0)
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Content, &p.MediaURL, &p.LikesCount, &p.CommentsCount, &p.RepostsCount, &p.CreatedAt, &p.UpdatedAt, &p.EditedAt); err != nil {
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
