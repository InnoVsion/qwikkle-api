package uploads

import (
	"context"
	"time"
)

type UploadStatus string

const (
	UploadStatusPending   UploadStatus = "pending"
	UploadStatusCompleted UploadStatus = "completed"
)

type Upload struct {
	ID          string
	Provider    string
	StorageKey  string
	DownloadURL *string
	FileName    string
	FileSize    int64
	MimeType    string
	Status      UploadStatus
	CreatedAt   time.Time
	CompletedAt *time.Time
}

type Repository interface {
	Create(ctx context.Context, provider string, storageKey string, downloadURL *string, fileName string, fileSize int64, mimeType string) (*Upload, error)
	MarkCompleted(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*Upload, error)
}
