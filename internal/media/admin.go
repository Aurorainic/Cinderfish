package media

import (
	"context"
	"fmt"
	"strings"
)

// AdminDriveParams is the filter set of `admin/drive/files`.
type AdminDriveParams struct {
	Limit      int
	SinceID    string
	UntilID    string
	UserID     *string
	TypePrefix string
	Origin     string
	Hostname   *string
}

// ListAll lists files across every account, newest first. The caller is
// expected to have checked the admin credential; the service does not.
func (s *DriveService) ListAll(ctx context.Context, p AdminDriveParams) ([]*File, error) {
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}

	conds := []string{}
	args := []any{}

	if p.UserID != nil && *p.UserID != "" {
		args = append(args, *p.UserID)
		conds = append(conds, fmt.Sprintf("f.user_id = $%d", len(args)))
	} else {
		switch p.Origin {
		case "", "combined", "local":
			if p.Origin == "local" {
				conds = append(conds, "u.host IS NULL")
			}
		case "remote":
			conds = append(conds, "u.host IS NOT NULL")
		default:
			return nil, fmt.Errorf("media: unknown origin %q", p.Origin)
		}
		if p.Hostname != nil && *p.Hostname != "" {
			args = append(args, strings.ToLower(*p.Hostname))
			conds = append(conds, fmt.Sprintf("u.host = $%d", len(args)))
		}
	}

	if p.TypePrefix != "" {
		args = append(args, escapeLike(p.TypePrefix)+"%")
		conds = append(conds, fmt.Sprintf("f.content_type LIKE $%d", len(args)))
	}
	if p.SinceID != "" {
		args = append(args, p.SinceID)
		conds = append(conds, fmt.Sprintf("f.id > $%d", len(args)))
	}
	if p.UntilID != "" {
		args = append(args, p.UntilID)
		conds = append(conds, fmt.Sprintf("f.id < $%d", len(args)))
	}

	query := `
SELECT f.id, f.user_id, f.name, f.content_type, f.size, f.md5, f.storage_key,
       f.storage_backend, f.thumbnail_key, f.is_sensitive, f.folder_id, f.created_at
FROM drive_files f JOIN users u ON u.id = f.user_id`
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY f.id DESC LIMIT %d", p.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("media: admin list files: %w", err)
	}
	defer rows.Close()

	var out []*File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.UserID, &f.Name, &f.ContentType, &f.Size, &f.MD5,
			&f.StorageKey, &f.StorageBackend, &f.ThumbnailKey, &f.IsSensitive, &f.FolderID, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("media: scan admin file list: %w", err)
		}
		f.URL = s.storage.URL(f.StorageKey)
		out = append(out, &f)
	}
	return out, rows.Err()
}

// escapeLike escapes the characters LIKE treats specially.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}
