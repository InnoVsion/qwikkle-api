package uploads

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"qwikkle-api/internal/db"
)

var ErrNotFound = errors.New("not found")

type PostgresRepository struct {
	pool *db.Pool
}

func NewPostgresRepository(pool *db.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, storageKey string, fileName string, fileSize int64, mimeType string) (*Upload, error) {
	const q = `
		INSERT INTO uploads (storage_key, file_name, file_size, mime_type)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, storage_key, file_name, file_size, mime_type, status::text, created_at, completed_at
	`

	var u Upload
	var status string
	err := r.pool.QueryRow(ctx, q, storageKey, fileName, fileSize, mimeType).Scan(
		&u.ID,
		&u.StorageKey,
		&u.FileName,
		&u.FileSize,
		&u.MimeType,
		&status,
		&u.CreatedAt,
		&u.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	u.Status = UploadStatus(status)
	return &u, nil
}

func (r *PostgresRepository) MarkCompleted(ctx context.Context, id string) error {
	const q = `
		UPDATE uploads
		SET status = 'completed'::upload_status, completed_at = NOW()
		WHERE id = $1 AND status = 'pending'::upload_status
	`
	ct, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (*Upload, error) {
	const q = `
		SELECT id::text, storage_key, file_name, file_size, mime_type, status::text, created_at, completed_at
		FROM uploads
		WHERE id = $1
	`

	var u Upload
	var status string
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID,
		&u.StorageKey,
		&u.FileName,
		&u.FileSize,
		&u.MimeType,
		&status,
		&u.CreatedAt,
		&u.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u.Status = UploadStatus(status)
	return &u, nil
}
