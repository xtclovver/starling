package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usedcvnt/microtwitter/user-svc/internal/model"
)

type NotificationRepository interface {
	Create(ctx context.Context, userID, actorID, nType string, postID, commentID *string) (*model.Notification, error)
	GetByUser(ctx context.Context, userID, cursor string, limit int) ([]model.Notification, string, bool, error)
	GetUnreadCount(ctx context.Context, userID string) (int32, error)
	MarkRead(ctx context.Context, id, userID string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type notifRepo struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) NotificationRepository {
	return &notifRepo{pool: pool}
}

func (r *notifRepo) Create(ctx context.Context, userID, actorID, nType string, postID, commentID *string) (*model.Notification, error) {
	n := &model.Notification{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO notifications (user_id, actor_id, type, post_id, comment_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, actor_id, type, post_id, comment_id, read, created_at`,
		userID, actorID, nType, postID, commentID,
	).Scan(&n.ID, &n.UserID, &n.ActorID, &n.Type, &n.PostID, &n.CommentID, &n.Read, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (r *notifRepo) GetByUser(ctx context.Context, userID, cursor string, limit int) ([]model.Notification, string, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args := []any{userID, limit + 1}
	q := `SELECT id, user_id, actor_id, type, post_id, comment_id, read, created_at
		  FROM notifications
		  WHERE user_id = $1`

	if cursor != "" {
		q += ` AND (created_at, id) < (
			(SELECT created_at FROM notifications WHERE id = $3),
			$3
		)`
		args = append(args, cursor)
	}

	q += ` ORDER BY created_at DESC, id DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	var notifications []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.ActorID, &n.Type, &n.PostID, &n.CommentID, &n.Read, &n.CreatedAt); err != nil {
			return nil, "", false, err
		}
		notifications = append(notifications, n)
	}

	hasMore := len(notifications) > limit
	if hasMore {
		notifications = notifications[:limit]
	}

	var nextCursor string
	if hasMore {
		nextCursor = notifications[limit-1].ID
	}

	return notifications, nextCursor, hasMore, nil
}

func (r *notifRepo) GetUnreadCount(ctx context.Context, userID string) (int32, error) {
	var count int32
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = FALSE`,
		userID,
	).Scan(&count)
	return count, err
}

func (r *notifRepo) MarkRead(ctx context.Context, id, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read = TRUE WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	return err
}

func (r *notifRepo) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read = TRUE WHERE user_id = $1 AND read = FALSE`,
		userID,
	)
	return err
}
