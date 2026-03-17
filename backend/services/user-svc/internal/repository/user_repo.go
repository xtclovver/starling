package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usedcvnt/microtwitter/user-svc/internal/model"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrDuplicateEmail = errors.New("email already exists")
	ErrDuplicateUsername = errors.New("username already exists")
)

type UserRepository interface {
	Create(ctx context.Context, username, email, passwordHash string) (*model.User, error)
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, id string, fields map[string]string) (*model.User, error)
	SoftDelete(ctx context.Context, id string) error
	Search(ctx context.Context, query, cursor string, limit int) ([]model.User, string, error)
	GetByIDs(ctx context.Context, ids []string) ([]model.User, error)
}

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) Create(ctx context.Context, username, email, passwordHash string) (*model.User, error) {
	u := &model.User{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, username, email, password_hash, display_name, bio, avatar_url, created_at, updated_at`,
		username, email, passwordHash,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Bio, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "email") {
				return nil, ErrDuplicateEmail
			}
			return nil, ErrDuplicateUsername
		}
		return nil, err
	}
	return u, nil
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	u := &model.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, display_name, bio, avatar_url, created_at, updated_at
		 FROM users WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Bio, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	u := &model.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, display_name, bio, avatar_url, created_at, updated_at
		 FROM users WHERE email = $1 AND deleted_at IS NULL`, email,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Bio, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *userRepo) Update(ctx context.Context, id string, fields map[string]string) (*model.User, error) {
	if len(fields) == 0 {
		return r.GetByID(ctx, id)
	}

	setClauses := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields)+1)
	i := 1
	for col, val := range fields {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, i))
		args = append(args, val)
		i++
	}
	args = append(args, id)

	query := fmt.Sprintf(
		`UPDATE users SET %s WHERE id = $%d AND deleted_at IS NULL
		 RETURNING id, username, email, password_hash, display_name, bio, avatar_url, created_at, updated_at`,
		strings.Join(setClauses, ", "), i,
	)

	u := &model.User{}
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Bio, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *userRepo) SoftDelete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *userRepo) Search(ctx context.Context, query, cursor string, limit int) ([]model.User, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args := []any{query, limit + 1}
	q := `SELECT id, username, email, password_hash, display_name, bio, avatar_url, created_at, updated_at
		  FROM users
		  WHERE deleted_at IS NULL
		    AND (username ILIKE '%' || $1 || '%' OR display_name ILIKE '%' || $1 || '%')`

	if cursor != "" {
		q += ` AND (created_at, id) < (
			(SELECT created_at FROM users WHERE id = $3),
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

	users := make([]model.User, 0)
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Bio, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, "", err
		}
		users = append(users, u)
	}

	var nextCursor string
	if len(users) > limit {
		users = users[:limit]
		nextCursor = users[limit-1].ID
	}

	return users, nextCursor, nil
}

func (r *userRepo) GetByIDs(ctx context.Context, ids []string) ([]model.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, username, email, password_hash, display_name, bio, avatar_url, created_at, updated_at
		 FROM users WHERE id = ANY($1) AND deleted_at IS NULL`, ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]model.User, 0, len(ids))
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Bio, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}
