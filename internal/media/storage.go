// Package media stores uploaded files. Two backends implement the same
// interface: the local filesystem and any S3 compatible object store.
package media

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// Storage is the interface every media backend implements.
type Storage interface {
	// Put stores the content under key and returns the URL clients use.
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (string, error)
	// Get opens the content stored under key.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the content stored under key.
	Delete(ctx context.Context, key string) error
	// URL returns the publicly reachable URL of a key.
	URL(key string) string
	// Backend names the implementation, "local" or "s3".
	Backend() string
}

// ErrNotFound is returned when a key does not exist.
var ErrNotFound = errors.New("media: object not found")

// BuildStorage constructs the backend named by the configuration values.
// It is shared by the binary's startup path and the admin console's runtime
// object-storage update, so both interpret the settings the same way. With
// no public base URL the bucket is assumed private and URLs are presigned.
func BuildStorage(backend, localPath string, s3 S3Options) (Storage, error) {
	switch backend {
	case "s3":
		if s3.PublicBaseURL == "" {
			s3.Presign = true
		}
		return NewS3Storage(s3)
	case "", "local":
		return NewLocalStorage(localPath, "/media")
	default:
		return nil, fmt.Errorf("media: unknown backend %q", backend)
	}
}

// Errors and limits for drive/files/update.
var (
	ErrInvalidName    = errors.New("media: invalid file name")
	ErrInvalidComment = errors.New("media: comment is too long")
	ErrNoSuchFolder   = errors.New("media: no such folder")
)

const (
	// MaxFileNameLength matches Misskey's drive file name limit.
	MaxFileNameLength = 255
	// MaxCommentLength caps the drive file caption.
	MaxCommentLength = 512
)

// ErrTooLarge is returned when an upload exceeds the configured limit.
var ErrTooLarge = errors.New("media: file is too large")
