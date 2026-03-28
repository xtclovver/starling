package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usedcvnt/microtwitter/media-svc/internal/model"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
)

type MediaRepository interface {
	Create(ctx context.Context, userID string, postID *string, bucket, objectKey, contentType string) (*model.Media, error)
	GetByID(ctx context.Context, id string) (*model.Media, error)
	GetByObjectKey(ctx context.Context, objectKey string) (*model.Media, error)
	GetByPostID(ctx context.Context, postID string) ([]model.Media, error)
	GetByPostIDs(ctx context.Context, postIDs []string) (map[string][]model.Media, error)
	LinkToPost(ctx context.Context, objectKeys []string, postID, userID string) error
	Delete(ctx context.Context, id, userID string) (*model.Media, error)
}

type mediaRepo struct {
	pool *pgxpool.Pool
}

func NewMediaRepository(pool *pgxpool.Pool) MediaRepository {
	return &mediaRepo{pool: pool}
}

func (r *mediaRepo) Create(ctx context.Context, userID string, postID *string, bucket, objectKey, contentType string) (*model.Media, error) {
	m := &model.Media{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO media (user_id, post_id, bucket, object_key, content_type)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, post_id, bucket, object_key, content_type, position, created_at`,
		userID, postID, bucket, objectKey, contentType,
	).Scan(&m.ID, &m.UserID, &m.PostID, &m.Bucket, &m.ObjectKey, &m.ContentType, &m.Position, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *mediaRepo) GetByID(ctx context.Context, id string) (*model.Media, error) {
	m := &model.Media{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, post_id, bucket, object_key, content_type, position, created_at
		 FROM media WHERE id = $1`, id,
	).Scan(&m.ID, &m.UserID, &m.PostID, &m.Bucket, &m.ObjectKey, &m.ContentType, &m.Position, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

func (r *mediaRepo) GetByObjectKey(ctx context.Context, objectKey string) (*model.Media, error) {
	m := &model.Media{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, post_id, bucket, object_key, content_type, position, created_at
		 FROM media WHERE object_key = $1`, objectKey,
	).Scan(&m.ID, &m.UserID, &m.PostID, &m.Bucket, &m.ObjectKey, &m.ContentType, &m.Position, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

func (r *mediaRepo) GetByPostID(ctx context.Context, postID string) ([]model.Media, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, post_id, bucket, object_key, content_type, position, created_at
		 FROM media WHERE post_id = $1
		 ORDER BY position ASC`, postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.Media
	for rows.Next() {
		var m model.Media
		if err := rows.Scan(&m.ID, &m.UserID, &m.PostID, &m.Bucket, &m.ObjectKey, &m.ContentType, &m.Position, &m.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, nil
}

func (r *mediaRepo) GetByPostIDs(ctx context.Context, postIDs []string) (map[string][]model.Media, error) {
	if len(postIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, post_id, bucket, object_key, content_type, position, created_at
		 FROM media WHERE post_id = ANY($1)
		 ORDER BY post_id, position ASC`, postIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]model.Media)
	for rows.Next() {
		var m model.Media
		if err := rows.Scan(&m.ID, &m.UserID, &m.PostID, &m.Bucket, &m.ObjectKey, &m.ContentType, &m.Position, &m.CreatedAt); err != nil {
			return nil, err
		}
		if m.PostID != nil {
			result[*m.PostID] = append(result[*m.PostID], m)
		}
	}
	return result, nil
}

func (r *mediaRepo) LinkToPost(ctx context.Context, objectKeys []string, postID, userID string) error {
	if len(objectKeys) > 10 {
		return errors.New("max 10 media per post")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Unlink existing media from this post
	_, err = tx.Exec(ctx, `UPDATE media SET post_id = NULL, position = 0 WHERE post_id = $1`, postID)
	if err != nil {
		return err
	}

	// Link new media in order
	for i, key := range objectKeys {
		tag, err := tx.Exec(ctx,
			`UPDATE media SET post_id = $1, position = $2 WHERE object_key = $3 AND user_id = $4`,
			postID, i, key, userID,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errors.New("media not found or not owned: " + key)
		}
	}

	return tx.Commit(ctx)
}

func (r *mediaRepo) Delete(ctx context.Context, id, userID string) (*model.Media, error) {
	m := &model.Media{}
	err := r.pool.QueryRow(ctx,
		`DELETE FROM media WHERE id = $1 AND user_id = $2
		 RETURNING id, user_id, post_id, bucket, object_key, content_type, position, created_at`,
		id, userID,
	).Scan(&m.ID, &m.UserID, &m.PostID, &m.Bucket, &m.ObjectKey, &m.ContentType, &m.Position, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			_ = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM media WHERE id = $1)`, id).Scan(&exists)
			if exists {
				return nil, ErrForbidden
			}
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}
