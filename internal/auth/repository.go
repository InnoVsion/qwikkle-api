package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"qwikkle-api/internal/db"
	"qwikkle-api/internal/types"
)

var (
	ErrIdentityTaken      = errors.New("identity already registered")
	ErrInvalidCredentials = errors.New("invalid qkId or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrSessionNotFound    = errors.New("session not found")
)

type Repository interface {
	CreateUser(ctx context.Context, in CreateUserInput) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByQKID(ctx context.Context, qkID string) (*User, error)
	UpdateUserProfile(ctx context.Context, userID string, in UpdateUserProfileInput) (*User, error)
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

type CreateUserInput struct {
	QKID              string
	Email             *string
	PasswordHash      string
	Role              string
	Status            types.AccountStatus
	FirstName         *string
	LastName          *string
	Phone             *string
	AvatarURL         *string
	Gender            *string
	DateOfBirth       *time.Time
	Country           *string
	Interests         []string
	AvatarStorageKey  *string
	AvatarDownloadURL *string
}

type UpdateUserProfileInput struct {
	Email             *string
	FirstName         *string
	LastName          *string
	Phone             *string
	AvatarURL         *string
	Gender            *string
	DateOfBirth       *time.Time
	Country           *string
	Interests         *[]string
	AvatarStorageKey  *string
	AvatarDownloadURL *string
}

func (r *postgresRepo) CreateUser(ctx context.Context, in CreateUserInput) (*User, error) {
	interests := in.Interests
	if interests == nil {
		interests = []string{}
	}

	const q = `
		INSERT INTO users (
			qk_id,
			email,
			password_hash,
			role,
			status,
			first_name,
			last_name,
			phone,
			avatar_url,
			gender,
			date_of_birth,
			country,
			interests,
			avatar_storage_key,
			avatar_download_url
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			NULLIF($6, ''),
			NULLIF($7, ''),
			NULLIF($8, ''),
			NULLIF($9, ''),
			NULLIF($10, ''),
			$11,
			NULLIF($12, ''),
			$13,
			NULLIF($14, ''),
			NULLIF($15, '')
		)
		RETURNING
			id,
			qk_id,
			email,
			password_hash,
			role,
			status,
			first_name,
			last_name,
			phone,
			avatar_url,
			gender,
			date_of_birth,
			country,
			interests,
			avatar_storage_key,
			avatar_download_url,
			organization_id,
			created_at,
			last_login_at
	`

	var u User
	err := r.pool.QueryRow(
		ctx,
		q,
		in.QKID,
		in.Email,
		in.PasswordHash,
		in.Role,
		in.Status,
		in.FirstName,
		in.LastName,
		in.Phone,
		in.AvatarURL,
		in.Gender,
		in.DateOfBirth,
		in.Country,
		interests,
		in.AvatarStorageKey,
		in.AvatarDownloadURL,
	).Scan(
		&u.ID,
		&u.QKID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.Status,
		&u.FirstName,
		&u.LastName,
		&u.Phone,
		&u.AvatarURL,
		&u.Gender,
		&u.DateOfBirth,
		&u.Country,
		&u.Interests,
		&u.AvatarStorageKey,
		&u.AvatarDownloadURL,
		&u.OrganizationID,
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
		SELECT
			id,
			qk_id,
			email,
			password_hash,
			role,
			status,
			first_name,
			last_name,
			phone,
			avatar_url,
			gender,
			date_of_birth,
			country,
			interests,
			avatar_storage_key,
			avatar_download_url,
			organization_id,
			created_at,
			last_login_at
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
		&u.FirstName,
		&u.LastName,
		&u.Phone,
		&u.AvatarURL,
		&u.Gender,
		&u.DateOfBirth,
		&u.Country,
		&u.Interests,
		&u.AvatarStorageKey,
		&u.AvatarDownloadURL,
		&u.OrganizationID,
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
		SELECT
			id,
			qk_id,
			email,
			password_hash,
			role,
			status,
			first_name,
			last_name,
			phone,
			avatar_url,
			gender,
			date_of_birth,
			country,
			interests,
			avatar_storage_key,
			avatar_download_url,
			organization_id,
			created_at,
			last_login_at
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
		&u.FirstName,
		&u.LastName,
		&u.Phone,
		&u.AvatarURL,
		&u.Gender,
		&u.DateOfBirth,
		&u.Country,
		&u.Interests,
		&u.AvatarStorageKey,
		&u.AvatarDownloadURL,
		&u.OrganizationID,
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

func (r *postgresRepo) UpdateUserProfile(ctx context.Context, userID string, in UpdateUserProfileInput) (*User, error) {
	const q = `
		UPDATE users
		SET
			email = COALESCE($2, email),
			first_name = COALESCE(NULLIF($3, ''), first_name),
			last_name = COALESCE(NULLIF($4, ''), last_name),
			phone = COALESCE(NULLIF($5, ''), phone),
			avatar_url = COALESCE(NULLIF($6, ''), avatar_url),
			gender = COALESCE(NULLIF($7, ''), gender),
			date_of_birth = COALESCE($8, date_of_birth),
			country = COALESCE(NULLIF($9, ''), country),
			interests = COALESCE($10, interests),
			avatar_storage_key = COALESCE(NULLIF($11, ''), avatar_storage_key),
			avatar_download_url = COALESCE(NULLIF($12, ''), avatar_download_url)
		WHERE id = $1
	`
	_, err := r.pool.Exec(
		ctx,
		q,
		userID,
		in.Email,
		in.FirstName,
		in.LastName,
		in.Phone,
		in.AvatarURL,
		in.Gender,
		in.DateOfBirth,
		in.Country,
		in.Interests,
		in.AvatarStorageKey,
		in.AvatarDownloadURL,
	)
	if err != nil {
		return nil, err
	}
	return r.GetUserByID(ctx, userID)
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
