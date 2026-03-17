package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAlreadyLiked = errors.New("already liked")

type CommentLikeRepository interface {
	Like(ctx context.Context, commentID, userID string) error
	Unlike(ctx context.Context, commentID, userID string) error
	IncrementLikes(ctx context.Context, commentID string, delta int) error
}

type commentLikeRepo struct {
	pool *pgxpool.Pool
}

func NewCommentLikeRepository(pool *pgxpool.Pool) CommentLikeRepository {
	return &commentLikeRepo{pool: pool}
}

func (r *commentLikeRepo) Like(ctx context.Context, commentID, userID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO likes (user_id, comment_id) VALUES ($1, $2)`,
		userID, commentID,
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

func (r *commentLikeRepo) Unlike(ctx context.Context, commentID, userID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM likes WHERE user_id = $1 AND comment_id = $2`,
		userID, commentID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *commentLikeRepo) IncrementLikes(ctx context.Context, commentID string, delta int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE comments SET likes_count = GREATEST(likes_count + $1, 0) WHERE id = $2`,
		delta, commentID,
	)
	return err
}
