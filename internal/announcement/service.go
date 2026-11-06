// Package announcement stores instance-wide announcements: the banners every
// account sees on the announcements page and the admin console manages.
package announcement

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// ErrNotFound is returned when an announcement id does not exist.
var ErrNotFound = errors.New("announcement: not found")

// Announcement is one instance notice.
type Announcement struct {
	ID        string
	Title     string
	Text      string
	ImageURL  *string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt *time.Time
}

// Service reads and writes announcements.
type Service struct {
	pool *pgxpool.Pool
}

// NewService returns a Service backed by the given pool.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

const announcementColumns = `id, title, text, image_url, is_active, created_at, updated_at, expires_at`

func scanAnnouncement(row pgx.Row) (*Announcement, error) {
	var a Announcement
	err := row.Scan(&a.ID, &a.Title, &a.Text, &a.ImageURL, &a.IsActive,
		&a.CreatedAt, &a.UpdatedAt, &a.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("announcement: scan: %w", err)
	}
	return &a, nil
}

// Create inserts an active announcement.
func (s *Service) Create(ctx context.Context, title, text string, imageURL *string, expiresAt *time.Time) (*Announcement, error) {
	row := s.pool.QueryRow(ctx, `
INSERT INTO announcements (id, title, text, image_url, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING `+announcementColumns, idgen.New(), title, text, imageURL, expiresAt)
	a, err := scanAnnouncement(row)
	if err != nil {
		return nil, fmt.Errorf("announcement: create: %w", err)
	}
	return a, nil
}

// List returns announcements newest first. activeOnly applies the public
// visibility rule: active and not expired.
func (s *Service) List(ctx context.Context, activeOnly bool, limit int, untilID string) ([]*Announcement, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	conds := []string{}
	args := []any{}
	if activeOnly {
		conds = append(conds, "is_active", "(expires_at IS NULL OR expires_at > now())")
	}
	if untilID != "" {
		args = append(args, untilID)
		conds = append(conds, fmt.Sprintf("id < $%d", len(args)))
	}
	query := `SELECT ` + announcementColumns + ` FROM announcements`
	if len(conds) > 0 {
		query += ` WHERE ` + strings.Join(conds, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT %d", limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("announcement: list: %w", err)
	}
	defer rows.Close()

	var out []*Announcement
	for rows.Next() {
		a, err := scanAnnouncement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Get loads one announcement by id.
func (s *Service) Get(ctx context.Context, id string) (*Announcement, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+announcementColumns+` FROM announcements WHERE id = $1`, id)
	return scanAnnouncement(row)
}

// Delete removes an announcement, reporting whether it existed.
func (s *Service) Delete(ctx context.Context, id string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM announcements WHERE id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("announcement: delete: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// MarkRead records that the user has seen an announcement. Revising an
// announcement does not clear the mark; a new notice is a new id.
func (s *Service) MarkRead(ctx context.Context, userID, announcementID string) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO announcement_reads (user_id, announcement_id) VALUES ($1, $2)
ON CONFLICT (user_id, announcement_id) DO NOTHING
`, userID, announcementID)
	if err != nil {
		return fmt.Errorf("announcement: mark read: %w", err)
	}
	return nil
}

// ReadIDs returns which of the given announcement ids the user has read.
func (s *Service) ReadIDs(ctx context.Context, userID string, ids []string) (map[string]bool, error) {
	out := make(map[string]bool, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT announcement_id FROM announcement_reads
WHERE user_id = $1 AND announcement_id = ANY($2)
`, userID, ids)
	if err != nil {
		return nil, fmt.Errorf("announcement: read ids: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("announcement: scan read id: %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

// UnreadCount counts active, unexpired announcements the user has not read.
func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
SELECT count(*) FROM announcements a
WHERE a.is_active = TRUE
  AND (a.expires_at IS NULL OR a.expires_at > now())
  AND NOT EXISTS (
      SELECT 1 FROM announcement_reads r
      WHERE r.user_id = $1 AND r.announcement_id = a.id
  )
`, userID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("announcement: unread count: %w", err)
	}
	return n, nil
}

// UpdateParams carries the editable announcement fields. A nil pointer keeps
// the current value; an empty ImageURL clears it.
type UpdateParams struct {
	Title     *string
	Text      *string
	ImageURL  *string
	IsActive  *bool
	ExpiresAt *time.Time
	SetExpiry bool
}

// Update rewrites an announcement. It returns ErrNotFound for an unknown id.
func (s *Service) Update(ctx context.Context, id string, p UpdateParams) (*Announcement, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Title != nil {
		current.Title = *p.Title
	}
	if p.Text != nil {
		current.Text = *p.Text
	}
	if p.ImageURL != nil {
		if *p.ImageURL == "" {
			current.ImageURL = nil
		} else {
			current.ImageURL = p.ImageURL
		}
	}
	if p.IsActive != nil {
		current.IsActive = *p.IsActive
	}
	if p.SetExpiry {
		current.ExpiresAt = p.ExpiresAt
	}

	row := s.pool.QueryRow(ctx, `
UPDATE announcements
SET title = $2, text = $3, image_url = $4, is_active = $5, expires_at = $6, updated_at = now()
WHERE id = $1
RETURNING `+announcementColumns+`
`, id, current.Title, current.Text, current.ImageURL, current.IsActive, current.ExpiresAt)
	return scanAnnouncement(row)
}
