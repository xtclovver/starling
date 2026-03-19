package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAlreadyReposted = errors.New("already reposted")

type RepostRepository interface {
	Repost(ctx context.Context, postID, userID string) error
	Unrepost(ctx context.Context, postID, userID string) error
	IsReposted(ctx context.Context, postID, userID string) (bool, error)
	QuotePost(ctx context.Context, postID, userID, content string) (string, error)
}

type repostRepo struct {
	pool *pgxpool.Pool
}

func NewRepostRepository(pool *pgxpool.Pool) RepostRepository {
	return &repostRepo{pool: pool}
}

func (r *repostRepo) Repost(ctx context.Context, postID, userID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO reposts (user_id, post_id, type) VALUES ($1, $2, 'repost')`,
		userID, postID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyReposted
		}
		return err
	}
	return nil
}

func (r *repostRepo) Unrepost(ctx context.Context, postID, userID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM reposts WHERE user_id = $1 AND post_id = $2 AND type = 'repost'`,
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

func (r *repostRepo) IsReposted(ctx context.Context, postID, userID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM reposts WHERE user_id = $1 AND post_id = $2 AND type = 'repost')`,
		userID, postID,
	).Scan(&exists)
	return exists, err
}

func (r *repostRepo) QuotePost(ctx context.Context, postID, userID, content string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO reposts (user_id, post_id, quote_content, type) VALUES ($1, $2, $3, 'quote') RETURNING id`,
		userID, postID, content,
	).Scan(&id)
	return id, err
}
