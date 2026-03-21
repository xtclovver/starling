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
	AreReposted(ctx context.Context, postIDs []string, userID string) (map[string]bool, error)
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

func (r *repostRepo) AreReposted(ctx context.Context, postIDs []string, userID string) (map[string]bool, error) {
	if len(postIDs) == 0 || userID == "" {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT post_id FROM reposts WHERE user_id = $1 AND post_id = ANY($2) AND type = 'repost'`,
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

func (r *repostRepo) QuotePost(ctx context.Context, postID, userID, content string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO reposts (user_id, post_id, quote_content, type) VALUES ($1, $2, $3, 'quote') RETURNING id`,
		userID, postID, content,
	).Scan(&id)
	return id, err
}
