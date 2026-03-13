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
		 RETURNING id, user_id, post_id, bucket, object_key, content_type, created_at`,
		userID, postID, bucket, objectKey, contentType,
	).Scan(&m.ID, &m.UserID, &m.PostID, &m.Bucket, &m.ObjectKey, &m.ContentType, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *mediaRepo) GetByID(ctx context.Context, id string) (*model.Media, error) {
	m := &model.Media{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, post_id, bucket, object_key, content_type, created_at
		 FROM media WHERE id = $1`, id,
	).Scan(&m.ID, &m.UserID, &m.PostID, &m.Bucket, &m.ObjectKey, &m.ContentType, &m.CreatedAt)
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
		`SELECT id, user_id, post_id, bucket, object_key, content_type, created_at
		 FROM media WHERE object_key = $1`, objectKey,
	).Scan(&m.ID, &m.UserID, &m.PostID, &m.Bucket, &m.ObjectKey, &m.ContentType, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

func (r *mediaRepo) Delete(ctx context.Context, id, userID string) (*model.Media, error) {
	m := &model.Media{}
	err := r.pool.QueryRow(ctx,
		`DELETE FROM media WHERE id = $1 AND user_id = $2
		 RETURNING id, user_id, post_id, bucket, object_key, content_type, created_at`,
		id, userID,
	).Scan(&m.ID, &m.UserID, &m.PostID, &m.Bucket, &m.ObjectKey, &m.ContentType, &m.CreatedAt)
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
