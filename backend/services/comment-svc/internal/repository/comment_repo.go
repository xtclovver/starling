package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usedcvnt/microtwitter/comment-svc/internal/model"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrForbidden    = errors.New("forbidden")
	ErrMaxDepth     = errors.New("maximum nesting depth exceeded")
	ErrPostNotFound = errors.New("post not found")
)

type CommentRepository interface {
	Create(ctx context.Context, postID, userID string, parentID *string, content string) (*model.Comment, error)
	GetTree(ctx context.Context, postID, cursor string, limit int) ([]model.Comment, string, error)
	SoftDelete(ctx context.Context, commentID, userID string) error
	IncrementPostComments(ctx context.Context, postID string) error
	DecrementPostComments(ctx context.Context, postID string) error
}

type commentRepo struct {
	pool *pgxpool.Pool
}

func NewCommentRepository(pool *pgxpool.Pool) CommentRepository {
	return &commentRepo{pool: pool}
}

func (r *commentRepo) Create(ctx context.Context, postID, userID string, parentID *string, content string) (*model.Comment, error) {
	// Check post exists
	var postExists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1 AND deleted_at IS NULL)`, postID).Scan(&postExists); err != nil {
		return nil, err
	}
	if !postExists {
		return nil, ErrPostNotFound
	}

	var depth int32
	if parentID != nil && *parentID != "" {
		var parentDepth int32
		err := r.pool.QueryRow(ctx,
			`SELECT depth FROM comments WHERE id = $1 AND deleted_at IS NULL`, *parentID,
		).Scan(&parentDepth)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		depth = parentDepth + 1
		if depth > 5 {
			return nil, ErrMaxDepth
		}
	} else {
		parentID = nil
	}

	c := &model.Comment{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO comments (post_id, user_id, parent_id, content, depth)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, post_id, user_id, parent_id, content, likes_count, depth, created_at, updated_at`,
		postID, userID, parentID, content, depth,
	).Scan(&c.ID, &c.PostID, &c.UserID, &c.ParentID, &c.Content, &c.LikesCount, &c.Depth, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *commentRepo) GetTree(ctx context.Context, postID, cursor string, limit int) ([]model.Comment, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// Get root comments with pagination
	args := []any{postID, limit + 1}
	q := `SELECT id, post_id, user_id, parent_id, content, likes_count, depth, created_at, updated_at, deleted_at
		  FROM comments
		  WHERE post_id = $1 AND parent_id IS NULL AND deleted_at IS NULL`

	if cursor != "" {
		q += ` AND (created_at, id) > (
			(SELECT created_at FROM comments WHERE id = $3),
			$3
		)`
		args = append(args, cursor)
	}

	q += ` ORDER BY created_at ASC, id ASC LIMIT $2`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var roots []model.Comment
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.ParentID, &c.Content, &c.LikesCount, &c.Depth, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt); err != nil {
			return nil, "", err
		}
		roots = append(roots, c)
	}

	var nextCursor string
	if len(roots) > limit {
		roots = roots[:limit]
		nextCursor = roots[limit-1].ID
	}

	if len(roots) == 0 {
		return roots, "", nil
	}

	// Load all children via recursive CTE
	rootIDs := make([]string, len(roots))
	for i, r := range roots {
		rootIDs[i] = r.ID
	}

	childRows, err := r.pool.Query(ctx, `
		WITH RECURSIVE tree AS (
			SELECT id, post_id, user_id, parent_id, content, likes_count, depth, created_at, updated_at, deleted_at
			FROM comments
			WHERE parent_id = ANY($1)
			UNION ALL
			SELECT c.id, c.post_id, c.user_id, c.parent_id, c.content, c.likes_count, c.depth, c.created_at, c.updated_at, c.deleted_at
			FROM comments c
			INNER JOIN tree t ON c.parent_id = t.id
			WHERE c.depth <= 5
		)
		SELECT id, post_id, user_id, parent_id, content, likes_count, depth, created_at, updated_at, deleted_at
		FROM tree
		ORDER BY depth ASC, created_at ASC`, rootIDs)
	if err != nil {
		return nil, "", err
	}
	defer childRows.Close()

	var children []model.Comment
	for childRows.Next() {
		var c model.Comment
		if err := childRows.Scan(&c.ID, &c.PostID, &c.UserID, &c.ParentID, &c.Content, &c.LikesCount, &c.Depth, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt); err != nil {
			return nil, "", err
		}
		children = append(children, c)
	}

	// Build tree
	all := append(roots, children...)
	result := buildTree(roots, all)

	return result, nextCursor, nil
}

func (r *commentRepo) SoftDelete(ctx context.Context, commentID, userID string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE comments SET deleted_at = NOW() WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		commentID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		_ = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1 AND deleted_at IS NULL)`, commentID).Scan(&exists)
		if exists {
			return ErrForbidden
		}
		return ErrNotFound
	}
	return nil
}

func (r *commentRepo) IncrementPostComments(ctx context.Context, postID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE posts SET comments_count = comments_count + 1 WHERE id = $1`, postID)
	return err
}

func (r *commentRepo) DecrementPostComments(ctx context.Context, postID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE posts SET comments_count = GREATEST(comments_count - 1, 0) WHERE id = $1`, postID)
	return err
}

func buildTree(roots []model.Comment, all []model.Comment) []model.Comment {
	byID := make(map[string]*model.Comment)
	for i := range roots {
		roots[i].Children = nil
		byID[roots[i].ID] = &roots[i]
	}

	for i := range all {
		if _, ok := byID[all[i].ID]; !ok {
			all[i].Children = nil
			byID[all[i].ID] = &all[i]
		}
	}

	// Attach children to parents
	for i := range all {
		c := &all[i]
		if c.ParentID != nil && *c.ParentID != "" {
			if parent, ok := byID[*c.ParentID]; ok {
				parent.Children = append(parent.Children, byID[c.ID])
			}
		}
	}

	// Handle deleted: replace content if has children, skip if no children
	result := make([]model.Comment, 0, len(roots))
	for _, r := range roots {
		if filtered := filterDeleted(byID[r.ID]); filtered != nil {
			result = append(result, *filtered)
		}
	}

	return result
}

func filterDeleted(c *model.Comment) *model.Comment {
	if c == nil {
		return nil
	}

	filtered := make([]*model.Comment, 0)
	for _, child := range c.Children {
		if f := filterDeleted(child); f != nil {
			filtered = append(filtered, f)
		}
	}
	c.Children = filtered

	if c.DeletedAt != nil {
		if len(c.Children) > 0 {
			c.Content = "[удалено]"
			return c
		}
		return nil
	}

	return c
}
