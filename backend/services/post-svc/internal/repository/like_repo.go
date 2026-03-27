package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAlreadyLiked = errors.New("already liked")

type LikeRepository interface {
	LikePost(ctx context.Context, postID, userID string) error
	UnlikePost(ctx context.Context, postID, userID string) error
	IsLiked(ctx context.Context, postID, userID string) (bool, error)
	AreLiked(ctx context.Context, postIDs []string, userID string) (map[string]bool, error)
	CountByPost(ctx context.Context, postID string) (int64, error)
}

type likeRepo struct {
	pool *pgxpool.Pool
}

func NewLikeRepository(pool *pgxpool.Pool) LikeRepository {
	return &likeRepo{pool: pool}
}

func (r *likeRepo) LikePost(ctx context.Context, postID, userID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO likes (user_id, post_id) VALUES ($1, $2)`,
		userID, postID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyLiked
		}
		return err
	}
	return nil
}

func (r *likeRepo) UnlikePost(ctx context.Context, postID, userID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM likes WHERE user_id = $1 AND post_id = $2`,
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

func (r *likeRepo) IsLiked(ctx context.Context, postID, userID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = $1 AND post_id = $2)`,
		userID, postID,
	).Scan(&exists)
	return exists, err
}

func (r *likeRepo) CountByPost(ctx context.Context, postID string) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM likes WHERE post_id = $1`, postID,
	).Scan(&count)
	return count, err
}

func (r *likeRepo) AreLiked(ctx context.Context, postIDs []string, userID string) (map[string]bool, error) {
	if len(postIDs) == 0 || userID == "" {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT post_id FROM likes WHERE user_id = $1 AND post_id = ANY($2)`,
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
