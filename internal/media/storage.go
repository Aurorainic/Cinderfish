// Package media stores uploaded files. Two backends implement the same
// interface: the local filesystem and any S3 compatible object store.
package media

import (
	"context"
	"errors"
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
