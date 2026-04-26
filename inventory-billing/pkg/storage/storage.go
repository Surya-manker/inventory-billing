// Package storage provides a provider-agnostic interface for persisting binary
// assets (invoice PDFs, attachments). Swap LocalStorage for S3Storage in
// production by changing the single line in cmd/worker/main.go.
package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Storage is the interface every backend must satisfy.
type Storage interface {
	// Put writes data under key. ContentType is stored as metadata on S3 but
	// ignored by LocalStorage.
	Put(ctx context.Context, key, contentType string, data []byte) error
	// Get retrieves the raw bytes for key.
	Get(ctx context.Context, key string) ([]byte, error)
	// PublicURL returns the URL a client should use to download key.
	PublicURL(key string) string
}

// ── local filesystem implementation ──────────────────────────────────────────

// LocalStorage stores files under a directory on the local filesystem.
// Suitable for development and single-server deployments.
type LocalStorage struct {
	baseDir string // e.g. "/var/data/assets"
	baseURL string // e.g. "http://localhost:8080/assets"
}

func NewLocalStorage(baseDir, baseURL string) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("storage: create base dir: %w", err)
	}
	return &LocalStorage{baseDir: baseDir, baseURL: baseURL}, nil
}

func (s *LocalStorage) Put(_ context.Context, key, _ string, data []byte) error {
	path := filepath.Join(s.baseDir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("storage: mkdir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("storage: write file %s: %w", path, err)
	}
	return nil
}

func (s *LocalStorage) Get(_ context.Context, key string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(s.baseDir, filepath.FromSlash(key)))
	if err != nil {
		return nil, fmt.Errorf("storage: read file %s: %w", key, err)
	}
	return data, nil
}

func (s *LocalStorage) PublicURL(key string) string {
	return s.baseURL + "/" + key
}

// ── S3 stub (wire up github.com/aws/aws-sdk-go-v2/service/s3 here) ───────────

// S3Config holds the credentials and bucket for an S3-compatible store.
type S3Config struct {
	Endpoint  string // empty → AWS, or MinIO URL
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	BaseURL   string // CDN or presigned URL prefix
}

// NewS3Storage returns a Storage backed by AWS S3 / MinIO.
// Requires: github.com/aws/aws-sdk-go-v2/service/s3
//
// Placeholder — implement when moving to multi-server deployment.
//
//nolint:ireturn
func NewS3Storage(_ S3Config) Storage {
	panic("S3Storage not yet implemented — use LocalStorage for now")
}
