package media

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Aurorainic/cinderfish/internal/idgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrFolderNotFound is returned when a folder id does not exist.
var ErrFolderNotFound = errors.New("media: no such folder")

// ErrFolderCycle is returned when moving a folder under itself or one of its
// own descendants.
var ErrFolderCycle = errors.New("media: cannot move a folder into itself")

// Folder is one row of drive_folders.
type Folder struct {
	ID        string
	Name      string
	UserID    string
	ParentID  *string
	CreatedAt time.Time
	// FoldersCount and FilesCount are the direct children, filled by the
	// listing queries so packers do not need a second round trip.
	FoldersCount int
	FilesCount   int
}

// FolderService manages a user's drive folders. It shares the pool with
// DriveService and is created from it, so a caller only wires one service.
type FolderService struct {
	pool *pgxpool.Pool
}

// NewFolderService returns a FolderService over the pool.
func NewFolderService(pool *pgxpool.Pool) *FolderService {
	return &FolderService{pool: pool}
}

// CreateFolder creates a folder under parentID (nil for the root).
func (s *FolderService) CreateFolder(ctx context.Context, userID, name string, parentID *string) (*Folder, error) {
	if name == "" {
		name = "Untitled"
	}
	if parentID != nil {
		parent, err := s.GetFolder(ctx, *parentID)
		if err != nil {
			return nil, err
		}
		if parent.UserID != userID {
			return nil, ErrFolderNotFound
		}
	}

	f := &Folder{
		ID:        idgen.New(),
		Name:      name,
		UserID:    userID,
		ParentID:  parentID,
		CreatedAt: time.Now(),
	}
	if _, err := s.pool.Exec(ctx, `
INSERT INTO drive_folders (id, user_id, name, parent_id)
VALUES ($1,$2,$3,$4)
	`, f.ID, f.UserID, f.Name, f.ParentID); err != nil {
		return nil, fmt.Errorf("media: create folder: %w", err)
	}
	return f, nil
}

// GetFolder loads one folder with its direct child counts.
func (s *FolderService) GetFolder(ctx context.Context, id string) (*Folder, error) {
	row := s.pool.QueryRow(ctx, folderSelect+` WHERE f.id = $1`, id)
	f, err := scanFolder(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFolderNotFound
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

// ListFolders returns the user's folders inside parentID (nil for the root),
// newest first.
func (s *FolderService) ListFolders(ctx context.Context, userID string, parentID *string, limit int, sinceID, untilID string) ([]*Folder, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	query := folderSelect + ` WHERE f.user_id = $1`
	args := []any{userID}
	if parentID == nil {
		query += " AND f.parent_id IS NULL"
	} else {
		args = append(args, *parentID)
		query += fmt.Sprintf(" AND f.parent_id = $%d", len(args))
	}
	if sinceID != "" {
		args = append(args, sinceID)
		query += fmt.Sprintf(" AND f.id > $%d", len(args))
	}
	if untilID != "" {
		args = append(args, untilID)
		query += fmt.Sprintf(" AND f.id < $%d", len(args))
	}
	query += fmt.Sprintf(" ORDER BY f.id DESC LIMIT %d", limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("media: list folders: %w", err)
	}
	defer rows.Close()

	out := []*Folder{}
	for rows.Next() {
		f, err := scanFolder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// UpdateFolder renames a folder and/or moves it. A move that would create a
// cycle (into itself or a descendant) is refused.
func (s *FolderService) UpdateFolder(ctx context.Context, userID, folderID string, name *string, parentID *string, moveParent bool) (*Folder, error) {
	current, err := s.GetFolder(ctx, folderID)
	if err != nil {
		return nil, err
	}
	if current.UserID != userID {
		return nil, ErrFolderNotFound
	}

	if moveParent && parentID != nil {
		if *parentID == folderID {
			return nil, ErrFolderCycle
		}
		parent, err := s.GetFolder(ctx, *parentID)
		if err != nil {
			return nil, err
		}
		if parent.UserID != userID {
			return nil, ErrFolderNotFound
		}
		// Walk up from the new parent: meeting the folder means the move
		// would put it inside itself.
		seen := map[string]bool{}
		cursor := parent
		for cursor != nil && cursor.ParentID != nil && !seen[cursor.ID] {
			seen[cursor.ID] = true
			if *cursor.ParentID == folderID {
				return nil, ErrFolderCycle
			}
			next, err := s.GetFolder(ctx, *cursor.ParentID)
			if err != nil {
				break
			}
			cursor = next
		}
	}

	var sets []string
	var args []any
	if name != nil {
		args = append(args, *name)
		sets = append(sets, fmt.Sprintf("name = $%d", len(args)))
	}
	if moveParent {
		args = append(args, parentID)
		sets = append(sets, fmt.Sprintf("parent_id = $%d", len(args)))
	}
	if len(sets) == 0 {
		return current, nil
	}
	args = append(args, folderID)
	if _, err := s.pool.Exec(ctx,
		"UPDATE drive_folders SET "+joinSets(sets)+fmt.Sprintf(" WHERE id = $%d", len(args)), args...); err != nil {
		return nil, fmt.Errorf("media: update folder: %w", err)
	}
	return s.GetFolder(ctx, folderID)
}

// DeleteFolder removes a folder, returning its files and subfolders to the
// folder's own parent (the root when it had none), exactly like Misskey.
func (s *FolderService) DeleteFolder(ctx context.Context, userID, folderID string) error {
	folder, err := s.GetFolder(ctx, folderID)
	if err != nil {
		return err
	}
	if folder.UserID != userID {
		return ErrFolderNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("media: begin delete folder: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE drive_files SET folder_id = $1 WHERE folder_id = $2`, folder.ParentID, folderID); err != nil {
		return fmt.Errorf("media: reparent files: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE drive_folders SET parent_id = $1 WHERE parent_id = $2`, folder.ParentID, folderID); err != nil {
		return fmt.Errorf("media: reparent folders: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM drive_folders WHERE id = $1`, folderID); err != nil {
		return fmt.Errorf("media: delete folder: %w", err)
	}
	return tx.Commit(ctx)
}

const folderSelect = `
SELECT f.id, f.user_id, f.name, f.parent_id, f.created_at,
       (SELECT count(*) FROM drive_folders c WHERE c.parent_id = f.id),
       (SELECT count(*) FROM drive_files d WHERE d.folder_id = f.id)
FROM drive_folders f`

func scanFolder(row pgx.Row) (*Folder, error) {
	f := &Folder{}
	if err := row.Scan(&f.ID, &f.UserID, &f.Name, &f.ParentID, &f.CreatedAt, &f.FoldersCount, &f.FilesCount); err != nil {
		return nil, err
	}
	return f, nil
}

func joinSets(sets []string) string {
	out := ""
	for i, s := range sets {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
