package storage

import (
	"context"
	"strconv"
	"time"
)

type PresignResult struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Expires time.Time         `json:"expires"`
}

type Presigner interface {
	PresignPutObject(ctx context.Context, bucket string, key string, contentType string, contentLength int64, expiry time.Duration) (PresignResult, error)
}

type NotConfiguredError struct{}

func (e NotConfiguredError) Error() string { return "storage not configured" }

func NewNoopPresigner() Presigner {
	return noopPresigner{}
}

type noopPresigner struct{}

func (p noopPresigner) PresignPutObject(ctx context.Context, bucket string, key string, contentType string, contentLength int64, expiry time.Duration) (PresignResult, error) {
	return PresignResult{}, NotConfiguredError{}
}

func DefaultHeaders(contentType string, contentLength int64) map[string]string {
	h := map[string]string{}
	if contentType != "" {
		h["Content-Type"] = contentType
	}
	if contentLength > 0 {
		h["Content-Length"] = itoa64(contentLength)
	}
	return h
}

func itoa64(n int64) string {
	return strconv.FormatInt(n, 10)
}
