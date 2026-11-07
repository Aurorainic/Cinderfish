package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/idgen"
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
	ThumbnailURL   string
	IsSensitive    bool
	FolderID       *string
	Comment        string
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
func (s *DriveService) Save(ctx context.Context, userID, name string, r io.Reader, contentType string, folderID *string) (*File, error) {
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
	tmp, err := os.CreateTemp("", "cinderfish-upload-*")
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
		FolderID:       folderID,
		Name:           name,
		ContentType:    contentType,
		Size:           n,
		MD5:            digest,
		StorageKey:     key,
		StorageBackend: s.storage.Backend(),
		CreatedAt:      time.Now(),
		URL:            url,
	}

	// A thumbnail is best effort: an image that fails to decode is still a
	// valid upload, it just has no preview.
	if err := s.storeThumbnail(ctx, tmp, f); err != nil {
		log.Printf("media: thumbnail for %s: %v", f.ID, err)
	}

	if _, err := s.pool.Exec(ctx, `
INSERT INTO drive_files (id, user_id, name, content_type, size, md5, storage_key, storage_backend, thumbnail_key, folder_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
`, f.ID, f.UserID, f.Name, f.ContentType, f.Size, f.MD5, f.StorageKey, f.StorageBackend, f.ThumbnailKey, f.FolderID); err != nil {
		return nil, fmt.Errorf("media: insert file: %w", err)
	}
	return f, nil
}

// storeThumbnail renders and stores a preview for image uploads, filling the
// file's thumbnail fields. src must be rewound by the caller before the call;
// it is left at the end afterwards.
func (s *DriveService) storeThumbnail(ctx context.Context, src io.ReadSeeker, f *File) error {
	if !CanThumbnail(f.ContentType) {
		return nil
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("media: rewind for thumbnail: %w", err)
	}
	// Reading at most the upload limit bounds memory: Save already refused
	// anything larger.
	data, err := io.ReadAll(io.LimitReader(src, s.maxSize))
	if err != nil {
		return fmt.Errorf("media: read for thumbnail: %w", err)
	}
	thumb, ext, err := GenerateThumbnail(data, f.ContentType)
	if err != nil {
		return err
	}
	if thumb == nil {
		return nil
	}
	thumbKey := ThumbnailKeyFor(f.StorageKey, ext)
	thumbURL, err := s.storage.Put(ctx, thumbKey, bytes.NewReader(thumb), int64(len(thumb)),
		thumbnailContentType(ext))
	if err != nil {
		return err
	}
	key := thumbKey
	f.ThumbnailKey = &key
	f.ThumbnailURL = thumbURL
	return nil
}

// thumbnailURL resolves the stored thumbnail key to a URL, or empty when the
// file has none.
func (s *DriveService) thumbnailURL(f *File) string {
	if f.ThumbnailKey == nil || *f.ThumbnailKey == "" {
		return ""
	}
	return s.storage.URL(*f.ThumbnailKey)
}

// thumbnailContentType maps the thumbnail extension back to a media type.
func thumbnailContentType(ext string) string {
	if ext == ".png" {
		return "image/png"
	}
	return "image/jpeg"
}

// Get loads a file's metadata.
func (s *DriveService) Get(ctx context.Context, id string) (*File, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, COALESCE(user_id, ''), name, content_type, size, md5, storage_key, storage_backend,
       thumbnail_key, is_sensitive, folder_id, COALESCE(comment, ''), created_at
FROM drive_files WHERE id = $1
`, id)

	var f File
	err := row.Scan(&f.ID, &f.UserID, &f.Name, &f.ContentType, &f.Size, &f.MD5,
		&f.StorageKey, &f.StorageBackend, &f.ThumbnailKey, &f.IsSensitive, &f.FolderID, &f.Comment, &f.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("media: scan file: %w", err)
	}
	f.URL = s.storage.URL(f.StorageKey)
	f.ThumbnailURL = s.thumbnailURL(&f)
	return &f, nil
}

// GetMany loads the files with the given ids. Files that do not exist
// are omitted from the result; the order is not guaranteed.
func (s *DriveService) GetMany(ctx context.Context, ids []string) ([]*File, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(user_id, ''), name, content_type, size, md5, storage_key, storage_backend,
       thumbnail_key, is_sensitive, folder_id, COALESCE(comment, ''), created_at
FROM drive_files WHERE id = ANY($1)
`, ids)
	if err != nil {
		return nil, fmt.Errorf("media: list files by id: %w", err)
	}
	defer rows.Close()

	var out []*File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.UserID, &f.Name, &f.ContentType, &f.Size, &f.MD5,
			&f.StorageKey, &f.StorageBackend, &f.ThumbnailKey, &f.IsSensitive, &f.FolderID, &f.Comment, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("media: scan file list: %w", err)
		}
		f.URL = s.storage.URL(f.StorageKey)
		f.ThumbnailURL = s.thumbnailURL(&f)
		out = append(out, &f)
	}
	return out, rows.Err()
}

// List returns a user's files, newest first.
func (s *DriveService) List(ctx context.Context, userID string, limit int, sinceID, untilID string, folderID *string) ([]*File, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	query := `
SELECT id, COALESCE(user_id, ''), name, content_type, size, md5, storage_key, storage_backend,
       thumbnail_key, is_sensitive, folder_id, COALESCE(comment, ''), created_at
FROM drive_files WHERE user_id = $1`
	args := []any{userID}
	if folderID == nil {
		query += " AND folder_id IS NULL"
	} else {
		args = append(args, *folderID)
		query += fmt.Sprintf(" AND folder_id = $%d", len(args))
	}
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
			&f.StorageKey, &f.StorageBackend, &f.ThumbnailKey, &f.IsSensitive, &f.FolderID, &f.Comment, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("media: scan file list: %w", err)
		}
		f.URL = s.storage.URL(f.StorageKey)
		f.ThumbnailURL = s.thumbnailURL(&f)
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
	if f.ThumbnailKey != nil && *f.ThumbnailKey != "" {
		// A leftover thumbnail is harmless but it keeps the bytes billed, so
		// remove it too; a failure here must not fail the delete.
		if err := s.storage.Delete(ctx, *f.ThumbnailKey); err != nil {
			log.Printf("media: delete thumbnail %s: %v", *f.ThumbnailKey, err)
		}
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

// FileUpdate carries the editable metadata of a drive file. A nil pointer
// leaves that field unchanged; ClearFolder moves the file back to the root.
type FileUpdate struct {
	Name        *string
	IsSensitive *bool
	Comment     *string
	FolderID    *string
	ClearFolder bool
}

// Update rewrites a file's metadata. The file must belong to userID; a
// folderId must name one of the user's folders.
func (s *DriveService) Update(ctx context.Context, userID, fileID string, p FileUpdate) (*File, error) {
	f, err := s.Get(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if f.UserID != userID {
		return nil, ErrNotFound
	}

	var sets []string
	var args []any
	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	if p.Name != nil {
		name := strings.TrimSpace(*p.Name)
		if name == "" {
			return nil, ErrInvalidName
		}
		if len([]rune(name)) > MaxFileNameLength {
			return nil, ErrInvalidName
		}
		add("name", name)
	}
	if p.IsSensitive != nil {
		add("is_sensitive", *p.IsSensitive)
	}
	if p.Comment != nil {
		comment := strings.TrimSpace(*p.Comment)
		if len([]rune(comment)) > MaxCommentLength {
			return nil, ErrInvalidComment
		}
		if comment == "" {
			add("comment", nil)
		} else {
			add("comment", comment)
		}
	}
	if p.FolderID != nil {
		var owner string
		if err := s.pool.QueryRow(ctx, `SELECT user_id FROM drive_folders WHERE id = $1`, *p.FolderID).Scan(&owner); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNoSuchFolder
			}
			return nil, fmt.Errorf("media: check folder: %w", err)
		}
		if owner != userID {
			return nil, ErrNoSuchFolder
		}
		add("folder_id", *p.FolderID)
	} else if p.ClearFolder {
		add("folder_id", nil)
	}
	if len(sets) == 0 {
		return f, nil
	}
	args = append(args, fileID)
	if _, err := s.pool.Exec(ctx, "UPDATE drive_files SET "+strings.Join(sets, ", ")+fmt.Sprintf(" WHERE id = $%d", len(args)), args...); err != nil {
		return nil, fmt.Errorf("media: update file: %w", err)
	}
	return s.Get(ctx, fileID)
}

// Usage returns the total bytes and file count of a user's drive.
func (s *DriveService) Usage(ctx context.Context, userID string) (int64, int, error) {
	var bytes int64
	var files int
	if err := s.pool.QueryRow(ctx, `
SELECT COALESCE(sum(size), 0), count(*) FROM drive_files WHERE user_id = $1
`, userID).Scan(&bytes, &files); err != nil {
		return 0, 0, fmt.Errorf("media: drive usage: %w", err)
	}
	return bytes, files, nil
}

// MaxSize returns the configured per-file upload limit.
func (s *DriveService) MaxSize() int64 {
	return s.maxSize
}
