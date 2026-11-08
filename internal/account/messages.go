package account

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// MessagingMessage is one direct message, the legacy messaging/* family
// (fork-preserved contract; see docs/HANDOVER.md for the provenance
// note). Groups and federation are explicit deferrals of the first pass.
type MessagingMessage struct {
	ID          string
	UserID      string
	RecipientID string
	Text        *string
	FileID      *string
	IsRead      bool
	CreatedAt   time.Time
}

// Sentinels the handlers map onto Misskey's error codes.
var (
	ErrNoSuchMessage      = errors.New("account: no such message")
	ErrRecipientIsYourself = errors.New("account: recipient is yourself")
	ErrMessageNoContent   = errors.New("account: message needs text or a file")
	ErrMessageFileNotFound = errors.New("account: no such file")
	ErrRecipientBlockedYou = errors.New("account: recipient blocks the caller")
	ErrMessageTextTooLong  = errors.New("account: message text too long")
)

// The legacy text cap is the messaging paramDef's 3000.
const messagingTextMax = 3000

// CreateMessage stores a DM from senderID to recipientID. The sender must
// differ from the recipient, blocked senders are refused, and either text
// or an owned file is required.
func (s *Service) CreateMessage(ctx context.Context, senderID, recipientID string, text *string, fileID *string) (*MessagingMessage, error) {
	if senderID == recipientID {
		return nil, ErrRecipientIsYourself
	}
	content := text != nil && strings.TrimSpace(*text) != ""
	if !content && fileID == nil {
		return nil, ErrMessageNoContent
	}
	if text != nil && len([]rune(*text)) > messagingTextMax {
		return nil, ErrMessageTextTooLong
	}
	if fileID != nil && *fileID != "" {
		var owner string
		err := s.pool.QueryRow(ctx,
			`SELECT user_id FROM drive_files WHERE id = $1`, *fileID).Scan(&owner)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && owner != senderID) {
			return nil, ErrMessageFileNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("account: message file lookup: %w", err)
		}
	} else {
		fileID = nil
	}

	// YOU_HAVE_BEEN_BLOCKED fires before the insert, as upstream's
	// messages/create checks the block relation first.
	blocked, err := s.IsBlocked(ctx, recipientID, senderID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrRecipientBlockedYou
	}

	m := &MessagingMessage{
		ID:          idgen.New(),
		UserID:      senderID,
		RecipientID: recipientID,
		Text:        text,
		FileID:      fileID,
		CreatedAt:   time.Now(),
	}
	if _, err := s.pool.Exec(ctx, `
INSERT INTO messaging_messages (id, user_id, recipient_id, text, file_id)
VALUES ($1, $2, $3, $4, $5)
`, m.ID, m.UserID, m.RecipientID, m.Text, m.FileID); err != nil {
		return nil, fmt.Errorf("account: create message: %w", err)
	}
	return m, nil
}

// MessagingThreadQuery pages one conversation by message id.
type MessagingThreadQuery struct {
	Limit   int
	SinceID string
	UntilID string
	// MarkAsRead mirrors the messages paramDef default true: opening the
	// conversation reads it.
	MarkAsRead bool
	ReaderID   string
}

// ThreadMessages returns the messages between two users, newest first
// unless a sinceId is given without untilId — the conversation query the
// messaging/messages endpoint runs.
func (s *Service) ThreadMessages(ctx context.Context, a, b string, q MessagingThreadQuery) ([]*MessagingMessage, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	var sql strings.Builder
	sql.WriteString(`
SELECT id, user_id, recipient_id, text, file_id, is_read, created_at
FROM messaging_messages
WHERE (user_id = $1 AND recipient_id = $2) OR (user_id = $2 AND recipient_id = $1)`)
	args := []any{a, b}
	if q.SinceID != "" {
		args = append(args, q.SinceID)
		fmt.Fprintf(&sql, " AND id > $%d", len(args))
	}
	if q.UntilID != "" {
		args = append(args, q.UntilID)
		fmt.Fprintf(&sql, " AND id < $%d", len(args))
	}
	order := "DESC"
	if q.SinceID != "" && q.UntilID == "" {
		order = "ASC"
	}
	sql.WriteString(" ORDER BY id " + order + " LIMIT " + strconv.Itoa(limit))

	messages, err := s.queryMessages(ctx, sql.String(), args...)
	if err != nil {
		return nil, err
	}
	if q.MarkAsRead && q.ReaderID != "" {
		// Reading the thread flips the recipient-side rows, the ones the
		// reader has not sent themselves.
		for _, m := range messages {
			if m.RecipientID == q.ReaderID && !m.IsRead {
				if err := s.MarkMessageRead(ctx, q.ReaderID, m.ID); err != nil {
					return nil, err
				}
				m.IsRead = true
			}
		}
	}
	return messages, nil
}

// HistoryMessages answers messaging/history: the newest message per
// conversation (either direction), newest conversation first.
func (s *Service) HistoryMessages(ctx context.Context, userID string, limit int) ([]*MessagingMessage, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	// DISTINCT ON collapses each peer pair to its latest message.
	rows, err := s.pool.Query(ctx, `
SELECT DISTINCT ON (peer) id, user_id, recipient_id, text, file_id, is_read, created_at
FROM (
    SELECT id, user_id, recipient_id, text, file_id, is_read, created_at,
           CASE WHEN user_id = $1 THEN recipient_id ELSE user_id END AS peer
    FROM messaging_messages
    WHERE user_id = $1 OR recipient_id = $1
) c
ORDER BY peer, id DESC
LIMIT $2
`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("account: message history: %w", err)
	}
	return scanMessages(rows)
}

func scanMessages(rows pgx.Rows) ([]*MessagingMessage, error) {
	defer rows.Close()
	out := []*MessagingMessage{}
	for rows.Next() {
		var m MessagingMessage
		if err := rows.Scan(&m.ID, &m.UserID, &m.RecipientID, &m.Text, &m.FileID, &m.IsRead, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("account: scan message: %w", err)
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (s *Service) queryMessages(ctx context.Context, sql string, args ...any) ([]*MessagingMessage, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("account: messages query: %w", err)
	}
	return scanMessages(rows)
}

// GetMessage loads one message the viewer participates in; any other
// message is not-found, so ids stay unenumerable through the error.
func (s *Service) GetMessage(ctx context.Context, messageID, viewerID string) (*MessagingMessage, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, user_id, recipient_id, text, file_id, is_read, created_at
FROM messaging_messages WHERE id = $1
`, messageID)
	m, err := scanOneMessage(row)
	if err != nil {
		return nil, err
	}
	if m.UserID != viewerID && m.RecipientID != viewerID {
		return nil, ErrNoSuchMessage
	}
	return m, nil
}

func scanOneMessage(row pgx.Row) (*MessagingMessage, error) {
	var m MessagingMessage
	err := row.Scan(&m.ID, &m.UserID, &m.RecipientID, &m.Text, &m.FileID, &m.IsRead, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoSuchMessage
	}
	if err != nil {
		return nil, fmt.Errorf("account: scan message: %w", err)
	}
	return &m, nil
}

// MarkMessageRead flips one recipient-side row. The caller verifies the
// reader is the recipient; a non-participant never reaches the update.
func (s *Service) MarkMessageRead(ctx context.Context, readerID, messageID string) error {
	ct, err := s.pool.Exec(ctx, `
UPDATE messaging_messages SET is_read = true
WHERE id = $1 AND recipient_id = $2
`, messageID, readerID)
	if err != nil {
		return fmt.Errorf("account: read message: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNoSuchMessage
	}
	return nil
}

// MarkAllMessagesRead flips every unread message the account received —
// i/read-all-messaging-messages. Returns the ids that changed, for the
// read event.
func (s *Service) MarkAllMessagesRead(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id FROM messaging_messages WHERE recipient_id = $1 AND is_read = false
`, userID)
	if err != nil {
		return nil, fmt.Errorf("account: unread messages: %w", err)
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		if _, err := s.pool.Exec(ctx, `
UPDATE messaging_messages SET is_read = true
WHERE recipient_id = $1 AND is_read = false
`, userID); err != nil {
			return nil, fmt.Errorf("account: read all messages: %w", err)
		}
	}
	return ids, nil
}

// HasUnreadMessages reports whether the account has any unread DM, the
// MeDetailed.hasUnreadMessagingMessage flag.
func (s *Service) HasUnreadMessages(ctx context.Context, userID string) (bool, error) {
	var has bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM messaging_messages WHERE recipient_id = $1 AND is_read = false)
`, userID).Scan(&has)
	if err != nil {
		return false, fmt.Errorf("account: unread messages check: %w", err)
	}
	return has, nil
}

// DeleteMessage hard-deletes one message, the legacy delete semantics
// (no tombstone — the row goes and the stream event carries the id).
func (s *Service) DeleteMessage(ctx context.Context, messageID string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM messaging_messages WHERE id = $1`, messageID)
	if err != nil {
		return fmt.Errorf("account: delete message: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNoSuchMessage
	}
	return nil
}
