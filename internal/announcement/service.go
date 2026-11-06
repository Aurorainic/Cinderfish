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
