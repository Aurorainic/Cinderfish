package media

import (
	"context"
	"io"
	"sync"
)

// SwitchableStorage wraps a Storage and lets the active backend be replaced
// at runtime (the admin console edits object storage without a restart).
// The wrapper is handed to the drive service once at startup, so swapping
// the inner backend does not need the service rebuilt.
type SwitchableStorage struct {
	mu    sync.RWMutex
	inner Storage
}

// NewSwitchableStorage wraps an initial backend.
func NewSwitchableStorage(inner Storage) *SwitchableStorage {
	return &SwitchableStorage{inner: inner}
}

// Switch replaces the active backend. Requests in flight finish on the old
// one; the new backend serves everything after the swap. Existing files are
// not migrated — the operator moves them with cinderfish-media-migrate.
func (s *SwitchableStorage) Switch(inner Storage) {
	s.mu.Lock()
	s.inner = inner
	s.mu.Unlock()
}

// current returns the active backend under a read lock.
func (s *SwitchableStorage) current() Storage {
	s.mu.RLock()
	inner := s.inner
	s.mu.RUnlock()
	return inner
}

func (s *SwitchableStorage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (string, error) {
	return s.current().Put(ctx, key, r, size, contentType)
}

func (s *SwitchableStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.current().Get(ctx, key)
}

func (s *SwitchableStorage) Delete(ctx context.Context, key string) error {
	return s.current().Delete(ctx, key)
}

func (s *SwitchableStorage) URL(key string) string {
	return s.current().URL(key)
}

func (s *SwitchableStorage) Backend() string {
	return s.current().Backend()
}
