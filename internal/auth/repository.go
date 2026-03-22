package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"qwikkle-api/internal/db"
)

var (
	ErrIdentityTaken      = errors.New("identity already registered")
	ErrInvalidCredentials = errors.New("invalid qkId or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrSessionNotFound    = errors.New("session not found")
)

type Repository interface {
	CreateUser(ctx context.Context, qkID string, email *string, passwordHash string, role string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByQKID(ctx context.Context, qkID string) (*User, error)
	CreateSession(ctx context.Context, userID string, refreshTokenHash string, expiresAt time.Time, userAgent string, ip string) (string, error)
	GetSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (*Session, error)
	RotateSession(ctx context.Context, sessionID string, refreshTokenHash string, expiresAt time.Time) error
	RevokeSession(ctx context.Context, sessionID string) error
}

type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	CreatedAt        time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	UserAgent        *string
	IP               *string
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

func (r *postgresRepo) GetUserByID(ctx context.Context, id string) (*User, error) {
	const q = `
		SELECT id, qk_id, email, password_hash, role, status, created_at, last_login_at
		FROM users
		WHERE id = $1
	`

	var u User
	err := r.pool.QueryRow(ctx, q, id).Scan(
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

func (r *postgresRepo) CreateSession(
	ctx context.Context,
	userID string,
	refreshTokenHash string,
	expiresAt time.Time,
	userAgent string,
	ip string,
) (string, error) {
	const q = `
		INSERT INTO sessions (user_id, refresh_token_hash, expires_at, user_agent, ip)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, '')::inet)
		RETURNING id
	`

	var id string
	if err := r.pool.QueryRow(ctx, q, userID, refreshTokenHash, expiresAt, userAgent, ip).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *postgresRepo) GetSessionByRefreshTokenHash(
	ctx context.Context,
	refreshTokenHash string,
) (*Session, error) {
	const q = `
		SELECT id, user_id, refresh_token_hash, created_at, expires_at, revoked_at, user_agent, ip::text
		FROM sessions
		WHERE refresh_token_hash = $1
	`

	var s Session
	if err := r.pool.QueryRow(ctx, q, refreshTokenHash).Scan(
		&s.ID,
		&s.UserID,
		&s.RefreshTokenHash,
		&s.CreatedAt,
		&s.ExpiresAt,
		&s.RevokedAt,
		&s.UserAgent,
		&s.IP,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	return &s, nil
}

func (r *postgresRepo) RotateSession(ctx context.Context, sessionID string, refreshTokenHash string, expiresAt time.Time) error {
	const q = `
		UPDATE sessions
		SET refresh_token_hash = $1, expires_at = $2
		WHERE id = $3 AND revoked_at IS NULL
	`

	ct, err := r.pool.Exec(ctx, q, refreshTokenHash, expiresAt, sessionID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (r *postgresRepo) RevokeSession(ctx context.Context, sessionID string) error {
	const q = `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`

	ct, err := r.pool.Exec(ctx, q, sessionID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}
