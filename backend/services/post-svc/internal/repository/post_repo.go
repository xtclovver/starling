package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usedcvnt/microtwitter/post-svc/internal/model"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
)

type PostRepository interface {
	Create(ctx context.Context, userID, content, mediaURL string) (*model.Post, error)
	GetByID(ctx context.Context, id string) (*model.Post, error)
	SoftDelete(ctx context.Context, id, userID string) error
	GetFeed(ctx context.Context, userID, cursor string, limit int) ([]model.Post, string, bool, error)
	GetGlobalFeed(ctx context.Context, cursor string, limit int) ([]model.Post, string, bool, error)
	GetByUser(ctx context.Context, userID, cursor string, limit int) ([]model.Post, string, bool, error)
	IncrementLikes(ctx context.Context, postID string, delta int) error
	Update(ctx context.Context, id, userID, content string) (*model.Post, error)
	IncrementReposts(ctx context.Context, postID string, delta int) error
}

type postRepo struct {
	pool *pgxpool.Pool
}

func NewPostRepository(pool *pgxpool.Pool) PostRepository {
	return &postRepo{pool: pool}
}

func (r *postRepo) Create(ctx context.Context, userID, content, mediaURL string) (*model.Post, error) {
	p := &model.Post{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO posts (user_id, content, media_url)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, content, media_url, likes_count, comments_count, reposts_count, created_at, updated_at, edited_at`,
		userID, content, mediaURL,
	).Scan(&p.ID, &p.UserID, &p.Content, &p.MediaURL, &p.LikesCount, &p.CommentsCount, &p.RepostsCount, &p.CreatedAt, &p.UpdatedAt, &p.EditedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *postRepo) GetByID(ctx context.Context, id string) (*model.Post, error) {
	p := &model.Post{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, content, media_url, likes_count, comments_count, reposts_count, created_at, updated_at, edited_at
		 FROM posts WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&p.ID, &p.UserID, &p.Content, &p.MediaURL, &p.LikesCount, &p.CommentsCount, &p.RepostsCount, &p.CreatedAt, &p.UpdatedAt, &p.EditedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *postRepo) SoftDelete(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE posts SET deleted_at = NOW() WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		id, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Check if exists but different owner
		var exists bool
		_ = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1 AND deleted_at IS NULL)`, id).Scan(&exists)
		if exists {
			return ErrForbidden
		}
		return ErrNotFound
	}
	return nil
}

func (r *postRepo) GetFeed(ctx context.Context, userID, cursor string, limit int) ([]model.Post, string, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args := []any{userID, limit + 1}
	q := `SELECT p.id, p.user_id, p.content, p.media_url, p.likes_count, p.comments_count, p.reposts_count, p.created_at, p.updated_at, p.edited_at
		  FROM posts p
		  INNER JOIN follows f ON f.following_id = p.user_id AND f.follower_id = $1
		  WHERE p.deleted_at IS NULL`

	if cursor != "" {
		q += ` AND (p.created_at, p.id) < (
			(SELECT created_at FROM posts WHERE id = $3),
			$3
		)`
		args = append(args, cursor)
	}

	q += ` ORDER BY p.created_at DESC, p.id DESC LIMIT $2`

	return r.queryPosts(ctx, q, args, limit)
}

func (r *postRepo) GetByUser(ctx context.Context, userID, cursor string, limit int) ([]model.Post, string, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args := []any{userID, limit + 1}
	q := `SELECT id, user_id, content, media_url, likes_count, comments_count, reposts_count, created_at, updated_at, edited_at
		  FROM posts
		  WHERE user_id = $1 AND deleted_at IS NULL`

	if cursor != "" {
		q += ` AND (created_at, id) < (
			(SELECT created_at FROM posts WHERE id = $3),
			$3
		)`
		args = append(args, cursor)
	}

	q += ` ORDER BY created_at DESC, id DESC LIMIT $2`

	return r.queryPosts(ctx, q, args, limit)
}

func (r *postRepo) IncrementLikes(ctx context.Context, postID string, delta int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE posts SET likes_count = GREATEST(likes_count + $1, 0) WHERE id = $2`,
		delta, postID,
	)
	return err
}

func (r *postRepo) GetGlobalFeed(ctx context.Context, cursor string, limit int) ([]model.Post, string, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args := []any{limit + 1}
	q := `SELECT id, user_id, content, media_url, likes_count, comments_count, reposts_count, created_at, updated_at, edited_at
		  FROM posts
		  WHERE deleted_at IS NULL`

	if cursor != "" {
		q += ` AND (created_at, id) < (
			(SELECT created_at FROM posts WHERE id = $2),
			$2
		)`
		args = append(args, cursor)
	}

	q += ` ORDER BY created_at DESC, id DESC LIMIT $1`

	return r.queryPosts(ctx, q, args, limit)
}

func (r *postRepo) Update(ctx context.Context, id, userID, content string) (*model.Post, error) {
	p := &model.Post{}
	err := r.pool.QueryRow(ctx,
		`UPDATE posts SET content = $1, edited_at = NOW(), updated_at = NOW()
		 WHERE id = $2 AND user_id = $3 AND deleted_at IS NULL
		 RETURNING id, user_id, content, media_url, likes_count, comments_count, reposts_count, created_at, updated_at, edited_at`,
		content, id, userID,
	).Scan(&p.ID, &p.UserID, &p.Content, &p.MediaURL, &p.LikesCount, &p.CommentsCount, &p.RepostsCount, &p.CreatedAt, &p.UpdatedAt, &p.EditedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			_ = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1 AND deleted_at IS NULL)`, id).Scan(&exists)
			if exists {
				return nil, ErrForbidden
			}
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *postRepo) IncrementReposts(ctx context.Context, postID string, delta int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE posts SET reposts_count = GREATEST(reposts_count + $1, 0) WHERE id = $2`,
		delta, postID,
	)
	return err
}

func (r *postRepo) queryPosts(ctx context.Context, query string, args []any, limit int) ([]model.Post, string, bool, error) {
	rows, err := r.pool.Query(ctx, query, args...)
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
