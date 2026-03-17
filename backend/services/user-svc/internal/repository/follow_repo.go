package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadyFollowing = errors.New("already following")
	ErrSelfFollow       = errors.New("cannot follow yourself")
)

type FollowRepository interface {
	Follow(ctx context.Context, followerID, followingID string) error
	Unfollow(ctx context.Context, followerID, followingID string) error
	GetFollowers(ctx context.Context, userID, cursor string, limit int) ([]string, string, error)
	GetFollowing(ctx context.Context, userID, cursor string, limit int) ([]string, string, error)
}

type followRepo struct {
	pool *pgxpool.Pool
}

func NewFollowRepository(pool *pgxpool.Pool) FollowRepository {
	return &followRepo{pool: pool}
}

func (r *followRepo) Follow(ctx context.Context, followerID, followingID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO follows (follower_id, following_id) VALUES ($1, $2)`,
		followerID, followingID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return ErrAlreadyFollowing
			}
			if pgErr.Code == "23514" && strings.Contains(pgErr.ConstraintName, "no_self") {
				return ErrSelfFollow
			}
		}
		return err
	}
	return nil
}

func (r *followRepo) Unfollow(ctx context.Context, followerID, followingID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM follows WHERE follower_id = $1 AND following_id = $2`,
		followerID, followingID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *followRepo) GetFollowers(ctx context.Context, userID, cursor string, limit int) ([]string, string, error) {
	return r.getFollowList(ctx, "follower_id", "following_id", userID, cursor, limit)
}

func (r *followRepo) GetFollowing(ctx context.Context, userID, cursor string, limit int) ([]string, string, error) {
	return r.getFollowList(ctx, "following_id", "follower_id", userID, cursor, limit)
}

func (r *followRepo) getFollowList(ctx context.Context, selectCol, whereCol, userID, cursor string, limit int) ([]string, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args := []any{userID, limit + 1}
	q := `SELECT ` + selectCol + `, created_at, id FROM follows WHERE ` + whereCol + ` = $1`

	if cursor != "" {
		q += ` AND (created_at, id) < (
			(SELECT created_at FROM follows WHERE id = $3),
			$3
		)`
		args = append(args, cursor)
	}

	q += ` ORDER BY created_at DESC, id DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	type row struct {
		targetID  string
		followID  string
	}
	var results []row
	for rows.Next() {
		var r row
		var createdAt interface{}
		if err := rows.Scan(&r.targetID, &createdAt, &r.followID); err != nil {
			return nil, "", err
		}
		results = append(results, r)
	}

	var nextCursor string
	if len(results) > limit {
		results = results[:limit]
		nextCursor = results[limit-1].followID
	}

	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.targetID
	}
	return ids, nextCursor, nil
}
