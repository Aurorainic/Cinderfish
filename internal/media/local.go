package media

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage keeps files on the local filesystem, laid out by content
// hash so identical uploads share one object.
type LocalStorage struct {
	root string
	// baseURL is the path prefix the server serves the files under.
	baseURL string
}

// NewLocalStorage returns a LocalStorage rooted at root.
func NewLocalStorage(root, baseURL string) (*LocalStorage, error) {
	if root == "" {
		root = "./data/files"
	}
	if baseURL == "" {
		baseURL = "/media"
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("media: create local root: %w", err)
	}
	return &LocalStorage{root: root, baseURL: baseURL}, nil
}

// Backend names the implementation.
func (s *LocalStorage) Backend() string { return "local" }

// Put writes the content under key.
func (s *LocalStorage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (string, error) {
	path := s.path(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("media: create directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("media: create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return "", fmt.Errorf("media: write file: %w", err)
	}
	return s.URL(key), nil
}

// Get opens the file stored under key.
func (s *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	f, err := os.Open(s.path(key))
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("media: open file: %w", err)
	}
	return f, nil
}

// Delete removes the file stored under key.
func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	if err := os.Remove(s.path(key)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("media: delete file: %w", err)
	}
	return nil
}

// URL returns the public URL of a key.
func (s *LocalStorage) URL(key string) string {
	return s.baseURL + "/" + strings.TrimPrefix(key, "/")
}

// path maps a key to a file path, refusing to escape the root.
func (s *LocalStorage) path(key string) string {
	clean := filepath.Clean("/" + key)
	return filepath.Join(s.root, clean)
}
