package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"qwikkle-api/internal/db"
)

var (
	ErrIdentityTaken      = errors.New("identity already registered")
	ErrInvalidCredentials = errors.New("invalid qkId or password")
	ErrUserNotFound       = errors.New("user not found")
)

type Repository interface {
	CreateUser(ctx context.Context, qkID string, email *string, passwordHash string, role string) (*User, error)
	GetUserByQKID(ctx context.Context, qkID string) (*User, error)
}

type postgresRepo struct {
	pool *db.Pool
}

func NewPostgresRepository(pool *db.Pool) Repository {
	return &postgresRepo{pool: pool}
}

func (r *postgresRepo) CreateUser(ctx context.Context, qkID string, email *string, passwordHash string, role string) (*User, error) {
	const q = `
		INSERT INTO users (qk_id, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, qk_id, email, password_hash, role, status, created_at, last_login_at
	`

	var u User
	err := r.pool.QueryRow(ctx, q, qkID, email, passwordHash, role).Scan(
		&u.ID,
		&u.QKID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.Status,
		&u.CreatedAt,
		&u.LastLoginAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.ConstraintName == "users_email_key" || pgErr.ConstraintName == "users_qk_id_key" {
				return nil, ErrIdentityTaken
			}
		}
		return nil, err
	}
	return &u, nil
}

func (r *postgresRepo) GetUserByQKID(ctx context.Context, qkID string) (*User, error) {
	const q = `
		SELECT id, qk_id, email, password_hash, role, status, created_at, last_login_at
		FROM users
		WHERE qk_id = $1
	`

	var u User
	err := r.pool.QueryRow(ctx, q, qkID).Scan(
		&u.ID,
		&u.QKID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.Status,
		&u.CreatedAt,
		&u.LastLoginAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}
