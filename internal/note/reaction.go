package note

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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

// ReactionQuery is the paging parameter set of notes/reactions.
type ReactionQuery struct {
	// Type filters to one reaction; empty means every reaction.
	Type      string
	Limit     int
	SinceID   string
	UntilID   string
	SinceDate *time.Time
	UntilDate *time.Time
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

// List returns the reactions on a note, newest first, optionally filtered to
// one reaction type. It backs Misskey's notes/reactions ("who reacted").
func (s *ReactionService) List(ctx context.Context, noteID string, q ReactionQuery) ([]*Reaction, error) {
	return s.query(ctx, "note_id", noteID, q)
}

// ListByUser returns the reactions one account made, newest first. It backs
// users/reactions (the profile's reaction tab).
func (s *ReactionService) ListByUser(ctx context.Context, userID string, q ReactionQuery) ([]*Reaction, error) {
	return s.query(ctx, "user_id", userID, q)
}

// query pages a reaction column. The column is a fixed identifier, never
// request input.
func (s *ReactionService) query(ctx context.Context, column, value string, q ReactionQuery) ([]*Reaction, error) {
	if q.Limit <= 0 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}

	var conds []string
	var args []any
	args = append(args, value)
	conds = append(conds, fmt.Sprintf(column+" = $%d", len(args)))
	if q.Type != "" {
		args = append(args, q.Type)
		conds = append(conds, fmt.Sprintf("emoji = $%d", len(args)))
	}
	if q.SinceID != "" {
		args = append(args, q.SinceID)
		conds = append(conds, fmt.Sprintf("id > $%d", len(args)))
	}
	if q.UntilID != "" {
		args = append(args, q.UntilID)
		conds = append(conds, fmt.Sprintf("id < $%d", len(args)))
	}
	if q.SinceDate != nil {
		args = append(args, *q.SinceDate)
		conds = append(conds, fmt.Sprintf("created_at > $%d", len(args)))
	}
	if q.UntilDate != nil {
		args = append(args, *q.UntilDate)
		conds = append(conds, fmt.Sprintf("created_at < $%d", len(args)))
	}

	order := "id DESC"
	if q.SinceID != "" && q.UntilID == "" && q.UntilDate == nil {
		order = "id ASC"
	}

	query := "SELECT id, note_id, user_id, emoji, created_at FROM note_reactions WHERE " +
		strings.Join(conds, " AND ") + " ORDER BY " + order +
		" LIMIT " + strconv.Itoa(q.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("note: list reactions: %w", err)
	}
	defer rows.Close()

	out := []*Reaction{}
	for rows.Next() {
		r := &Reaction{}
		if err := rows.Scan(&r.ID, &r.NoteID, &r.UserID, &r.Emoji, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("note: scan reaction: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
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
	_, err := s.CreateReturningID(ctx, recipientID, notifType, notifierID, noteID, reaction)
	return err
}

// CreateReturningID records a notification and returns its id, which the
// streaming layer needs to publish the event.
func (s *NotificationService) CreateReturningID(ctx context.Context, recipientID, notifType string, notifierID, noteID *string, reaction *string) (string, error) {
	id := newID()
	_, err := s.pool.Exec(ctx, `
INSERT INTO notifications (id, recipient_id, type, notifier_id, note_id, reaction)
VALUES ($1, $2, $3, $4, $5, $6)
`, id, recipientID, notifType, notifierID, noteID, reaction)
	if err != nil {
		return "", fmt.Errorf("note: create notification: %w", err)
	}
	return id, nil
}

// CreateForLocal records a notification only when the recipient is a
// local user, mirroring Misskey's local-only fan-out: a reply or renote
// aimed at a remote note's author is delivered over ActivityPub instead
// (core/NoteCreateService.ts L861-892). The boolean reports whether a row
// was created, so the caller knows whether to announce it.
func (s *NotificationService) CreateForLocal(ctx context.Context, recipientID, notifType string, notifierID, noteID *string, reaction *string) (string, bool, error) {
	id := newID()
	ct, err := s.pool.Exec(ctx, `
INSERT INTO notifications (id, recipient_id, type, notifier_id, note_id, reaction)
SELECT $1, u.id, $3, $4, $5, $6 FROM users u WHERE u.id = $2 AND u.host IS NULL
`, id, recipientID, notifType, notifierID, noteID, reaction)
	if err != nil {
		return "", false, fmt.Errorf("note: create notification: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return "", false, nil
	}
	return id, true, nil
}

// Get loads one notification by id.
func (s *NotificationService) Get(ctx context.Context, id string) (*Notification, error) {
	var n Notification
	err := s.pool.QueryRow(ctx, `
SELECT id, recipient_id, type, notifier_id, note_id, reaction, is_read, created_at
FROM notifications WHERE id = $1
`, id).Scan(&n.ID, &n.RecipientID, &n.Type, &n.NotifierID, &n.NoteID, &n.Reaction, &n.IsRead, &n.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("note: scan notification: %w", err)
	}
	return &n, nil
}

// List returns a user's notifications, newest first unless a sinceId is
// given without untilId, in which case the page is the newer remainder in
// ascending order (Misskey's makePaginationQuery semantics). A nil
// includeTypes leaves the type unfiltered, while an empty non-nil slice
// matches nothing, which is what Misskey returns for an explicitly empty
// includeTypes.
func (s *NotificationService) List(ctx context.Context, recipientID string, limit int, sinceID, untilID string, includeTypes, excludeTypes []string, unreadOnly bool) ([]*Notification, error) {
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
	if unreadOnly {
		query += " AND NOT is_read"
	}

	// Misskey's getNotifications filters with include-wins semantics
	// (packages/backend/src/core/NotificationService.ts L302-306): a
	// non-empty includeTypes takes precedence and excludeTypes is ignored.
	// An EMPTY includeTypes means no filter, not "match nothing" — hence
	// the length check, not a nil check.
	if len(includeTypes) > 0 {
		args = append(args, includeTypes)
		query += fmt.Sprintf(" AND type = ANY($%d)", len(args))
	} else if len(excludeTypes) > 0 {
		args = append(args, excludeTypes)
		query += fmt.Sprintf(" AND NOT (type = ANY($%d))", len(args))
	}
	if sinceID != "" {
		args = append(args, sinceID)
		query += fmt.Sprintf(" AND id > $%d", len(args))
	}
	if untilID != "" {
		args = append(args, untilID)
		query += fmt.Sprintf(" AND id < $%d", len(args))
	}
	// Misskey's makePaginationQuery (packages/backend/src/core/QueryService.ts
	// L54-62): a sinceId without untilId pages upward in ascending order, the
	// direction a "load newer" cursor expects; everything else is newest first.
	order := "DESC"
	if sinceID != "" && untilID == "" {
		order = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY id %s LIMIT %d", order, limit)

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

// MarkRead marks the given notifications of a user as read.
func (s *NotificationService) MarkRead(ctx context.Context, recipientID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE notifications SET is_read = TRUE WHERE recipient_id = $1 AND id = ANY($2)`,
		recipientID, ids)
	if err != nil {
		return fmt.Errorf("note: mark notifications read: %w", err)
	}
	return nil
}

// MarkAllRead marks every notification of a user as read.
func (s *NotificationService) MarkAllRead(ctx context.Context, recipientID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE notifications SET is_read = TRUE WHERE recipient_id = $1`, recipientID)
	if err != nil {
		return fmt.Errorf("note: mark notifications read: %w", err)
	}
	return nil
}

// UnreadCount returns the number of unread notifications, ignoring the
// given muted types.
func (s *NotificationService) UnreadCount(ctx context.Context, recipientID string, excludeTypes []string) (int, error) {
	query := `SELECT count(*) FROM notifications WHERE recipient_id = $1 AND is_read = FALSE`
	args := []any{recipientID}
	if len(excludeTypes) > 0 {
		args = append(args, excludeTypes)
		query += fmt.Sprintf(" AND NOT (type = ANY($%d))", len(args))
	}
	var n int
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("note: unread count: %w", err)
	}
	return n, nil
}
