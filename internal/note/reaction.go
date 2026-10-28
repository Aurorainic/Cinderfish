package note

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Reaction is a reaction on a note.
type Reaction struct {
	ID        string
	NoteID    string
	UserID    string
	Emoji     string
	CreatedAt time.Time
}

// ReactionService records and summarises reactions.
type ReactionService struct {
	pool *pgxpool.Pool
}

// NewReactionService returns a ReactionService backed by the given pool.
func NewReactionService(pool *pgxpool.Pool) *ReactionService {
	return &ReactionService{pool: pool}
}

// Add records a reaction. Reacting with the same emoji twice is a no-op.
func (s *ReactionService) Add(ctx context.Context, noteID, userID, emoji string) error {
	ct, err := s.pool.Exec(ctx, `
INSERT INTO note_reactions (id, note_id, user_id, emoji)
VALUES ($1, $2, $3, $4)
ON CONFLICT (note_id, user_id, emoji) DO NOTHING
`, newID(), noteID, userID, emoji)
	if err != nil {
		return fmt.Errorf("note: add reaction: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil
	}
	return nil
}

// Remove withdraws a reaction.
func (s *ReactionService) Remove(ctx context.Context, noteID, userID, emoji string) error {
	_, err := s.pool.Exec(ctx, `
DELETE FROM note_reactions WHERE note_id = $1 AND user_id = $2 AND emoji = $3
`, noteID, userID, emoji)
	if err != nil {
		return fmt.Errorf("note: remove reaction: %w", err)
	}
	return nil
}

// Counts returns the number of reactions per emoji for a note.
func (s *ReactionService) Counts(ctx context.Context, noteID string) (map[string]int, error) {
	rows, err := s.pool.Query(ctx, `
SELECT emoji, count(*) FROM note_reactions WHERE note_id = $1 GROUP BY emoji
`, noteID)
	if err != nil {
		return nil, fmt.Errorf("note: reaction counts: %w", err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var emoji string
		var n int
		if err := rows.Scan(&emoji, &n); err != nil {
			return nil, fmt.Errorf("note: scan reaction counts: %w", err)
		}
		counts[emoji] = n
	}
	return counts, rows.Err()
}

// MyReaction returns the reaction a user left on a note, or an empty
// string when there is none.
func (s *ReactionService) MyReaction(ctx context.Context, noteID, userID string) (string, error) {
	var emoji string
	err := s.pool.QueryRow(ctx, `
SELECT emoji FROM note_reactions WHERE note_id = $1 AND user_id = $2 LIMIT 1
`, noteID, userID).Scan(&emoji)
	if err != nil {
		return "", nil
	}
	return emoji, nil
}

// TotalCount returns the number of reactions on a note.
func (s *ReactionService) TotalCount(ctx context.Context, noteID string) (int, error) {
	var n int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM note_reactions WHERE note_id = $1`, noteID).Scan(&n); err != nil {
		return 0, fmt.Errorf("note: reaction total: %w", err)
	}
	return n, nil
}

// NotificationService creates and reads notifications.
type NotificationService struct {
	pool *pgxpool.Pool
}

// NewNotificationService returns a NotificationService backed by the pool.
func NewNotificationService(pool *pgxpool.Pool) *NotificationService {
	return &NotificationService{pool: pool}
}

// Notification is a stored notification.
type Notification struct {
	ID          string
	RecipientID string
	Type        string
	NotifierID  *string
	NoteID      *string
	Reaction    *string
	IsRead      bool
	CreatedAt   time.Time
}

// Create records a notification.
func (s *NotificationService) Create(ctx context.Context, recipientID, notifType string, notifierID, noteID *string, reaction *string) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO notifications (id, recipient_id, type, notifier_id, note_id, reaction)
VALUES ($1, $2, $3, $4, $5, $6)
`, newID(), recipientID, notifType, notifierID, noteID, reaction)
	if err != nil {
		return fmt.Errorf("note: create notification: %w", err)
	}
	return nil
}

// List returns a user's notifications, newest first.
func (s *NotificationService) List(ctx context.Context, recipientID string, limit int, sinceID, untilID string, excludeTypes []string) ([]*Notification, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	query := `
SELECT id, recipient_id, type, notifier_id, note_id, reaction, is_read, created_at
FROM notifications WHERE recipient_id = $1`
	args := []any{recipientID}

	if len(excludeTypes) > 0 {
		query += ` AND NOT (type = ANY($2))`
		args = append(args, excludeTypes)
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
		return nil, fmt.Errorf("note: list notifications: %w", err)
	}
	defer rows.Close()

	var out []*Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.RecipientID, &n.Type, &n.NotifierID, &n.NoteID, &n.Reaction, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("note: scan notification: %w", err)
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}

// MarkAllRead marks every notification of a user as read.
func (s *NotificationService) MarkAllRead(ctx context.Context, recipientID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE notifications SET is_read = TRUE WHERE recipient_id = $1`, recipientID)
	if err != nil {
		return fmt.Errorf("note: mark notifications read: %w", err)
	}
	return nil
}

// UnreadCount returns the number of unread notifications.
func (s *NotificationService) UnreadCount(ctx context.Context, recipientID string) (int, error) {
	var n int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM notifications WHERE recipient_id = $1 AND is_read = FALSE`,
		recipientID).Scan(&n); err != nil {
		return 0, fmt.Errorf("note: unread count: %w", err)
	}
	return n, nil
}
