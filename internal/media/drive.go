package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"milnfish/internal/idgen"
)

// DefaultMaxFileSize is the upload limit used when none is configured.
const DefaultMaxFileSize = 20 * 1024 * 1024

// File is the metadata of a stored file.
type File struct {
	ID             string
	UserID         string
	Name           string
	ContentType    string
	Size           int64
	MD5            string
	StorageKey     string
	StorageBackend string
	ThumbnailKey   *string
	IsSensitive    bool
	CreatedAt      time.Time
	URL            string
}

// DriveService manages the file metadata and delegates the bytes to a
// Storage backend.
type DriveService struct {
	pool    *pgxpool.Pool
	storage Storage
	maxSize int64
}

// NewDriveService returns a DriveService using the given storage.
func NewDriveService(pool *pgxpool.Pool, storage Storage) *DriveService {
	return &DriveService{pool: pool, storage: storage, maxSize: DefaultMaxFileSize}
}

// SetMaxFileSize changes the upload limit.
func (s *DriveService) SetMaxFileSize(n int64) {
	if n > 0 {
		s.maxSize = n
	}
}

// Save stores an uploaded file.
//
// The content is hashed while it is written, so identical uploads of the
// same bytes reuse one stored object.
func (s *DriveService) Save(ctx context.Context, userID, name string, r io.Reader, contentType string) (*File, error) {
	if name == "" {
		name = "file"
	}
	if contentType == "" {
		contentType = mime.TypeByExtension(path.Ext(name))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	// The upload is buffered in a temporary file so the hash is known
	// before the final storage key is chosen.
	tmp, err := os.CreateTemp("", "milnfish-upload-*")
	if err != nil {
		return nil, fmt.Errorf("media: create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(r, s.maxSize+1))
	if err != nil {
		return nil, fmt.Errorf("media: read upload: %w", err)
	}
	if n > s.maxSize {
		return nil, ErrTooLarge
	}

	digest := hex.EncodeToString(hash.Sum(nil))
	key := path.Join(digest[:2], digest[2:4], digest)

	// Re-read the temporary file into storage.
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("media: rewind upload: %w", err)
	}
	url, err := s.storage.Put(ctx, key, tmp, n, contentType)
	if err != nil {
		return nil, err
	}

	f := &File{
		ID:             idgen.New(),
		UserID:         userID,
		Name:           name,
		ContentType:    contentType,
		Size:           n,
		MD5:            digest,
		StorageKey:     key,
		StorageBackend: s.storage.Backend(),
		CreatedAt:      time.Now(),
		URL:            url,
	}

	if _, err := s.pool.Exec(ctx, `
INSERT INTO drive_files (id, user_id, name, content_type, size, md5, storage_key, storage_backend)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
`, f.ID, f.UserID, f.Name, f.ContentType, f.Size, f.MD5, f.StorageKey, f.StorageBackend); err != nil {
		return nil, fmt.Errorf("media: insert file: %w", err)
	}
	return f, nil
}

// Get loads a file's metadata.
func (s *DriveService) Get(ctx context.Context, id string) (*File, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, user_id, name, content_type, size, md5, storage_key, storage_backend,
       thumbnail_key, is_sensitive, created_at
FROM drive_files WHERE id = $1
`, id)

	var f File
	err := row.Scan(&f.ID, &f.UserID, &f.Name, &f.ContentType, &f.Size, &f.MD5,
		&f.StorageKey, &f.StorageBackend, &f.ThumbnailKey, &f.IsSensitive, &f.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("media: scan file: %w", err)
	}
	f.URL = s.storage.URL(f.StorageKey)
	return &f, nil
}

// List returns a user's files, newest first.
func (s *DriveService) List(ctx context.Context, userID string, limit int, sinceID, untilID string) ([]*File, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	query := `
SELECT id, user_id, name, content_type, size, md5, storage_key, storage_backend,
       thumbnail_key, is_sensitive, created_at
FROM drive_files WHERE user_id = $1`
	args := []any{userID}
	if sinceID != "" {
		args = append(args, sinceID)
		query += fmt.Sprintf(" AND id > $%d", len(args))
	}
	if untilID != "" {
		args = append(args, untilID)
		query += fmt.Sprintf(" AND id < $%d", len(args))
	}
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT %d", limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("media: list files: %w", err)
	}
	defer rows.Close()

	var out []*File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.UserID, &f.Name, &f.ContentType, &f.Size, &f.MD5,
			&f.StorageKey, &f.StorageBackend, &f.ThumbnailKey, &f.IsSensitive, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("media: scan file list: %w", err)
		}
		f.URL = s.storage.URL(f.StorageKey)
		out = append(out, &f)
	}
	return out, rows.Err()
}

// Delete removes a file and its stored object.
func (s *DriveService) Delete(ctx context.Context, id, userID string) error {
	f, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if f.UserID != userID {
		return ErrNotFound
	}
	if _, err := s.pool.Exec(ctx, `DELETE FROM drive_files WHERE id = $1`, id); err != nil {
		return fmt.Errorf("media: delete file row: %w", err)
	}
	if err := s.storage.Delete(ctx, f.StorageKey); err != nil {
		return err
	}
	return nil
}

// Open opens the stored content of a file.
func (s *DriveService) Open(ctx context.Context, id string) (io.ReadCloser, *File, error) {
	f, err := s.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	rc, err := s.storage.Get(ctx, f.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return rc, f, nil
}

// SanitizeName strips path components from an uploaded file name.
func SanitizeName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	name = strings.TrimSpace(name)
	// path.Base leaves "." and ".." untouched.
	if name == "" || name == "." || name == ".." || name == "/" {
		return "file"
	}
	return name
}
