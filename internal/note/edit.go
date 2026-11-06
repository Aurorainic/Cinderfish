package note

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// ErrNotNoteAuthor is returned when someone tries to edit another user's note.
var ErrNotNoteAuthor = errors.New("note: not the note author")

// EditParams carries the editable fields of a note. A nil pointer leaves the
// value unchanged; a pointer to an empty string clears text or CW.
type EditParams struct {
	Text       *string
	CW         *string
	Visibility string
	LocalOnly  *bool
}

// Edit updates a local note and records the previous revision. The tags and
// local mentions are recomputed from the new body, matching what a create
// would derive from it. Only text, CW, visibility and localOnly are editable;
// attachments and polls keep their values.
func (s *Service) Edit(ctx context.Context, userID, noteID string, p EditParams) (*Note, error) {
	n, err := s.Get(ctx, noteID)
	if err != nil {
		return nil, err
	}
	if n.UserID != userID {
		return nil, ErrNotNoteAuthor
	}

	visibility := n.Visibility
	if p.Visibility != "" {
		switch p.Visibility {
		case VisibilityPublic, VisibilityHome, VisibilityFollowers, VisibilitySpecified:
			visibility = p.Visibility
		default:
			return nil, ErrInvalidVisibility
		}
	}

	text := n.Text
	if p.Text != nil {
		trimmed := strings.TrimSpace(*p.Text)
		if len(trimmed) > MaxTextLength {
			return nil, ErrTextTooLong
		}
		if trimmed == "" {
			text = nil
		} else {
			text = &trimmed
		}
	}
	cw := n.CW
	if p.CW != nil {
		trimmed := strings.TrimSpace(*p.CW)
		if trimmed == "" {
			cw = nil
		} else {
			cw = &trimmed
		}
	}
	localOnly := n.LocalOnly
	if p.LocalOnly != nil {
		localOnly = *p.LocalOnly
	}

	tags := []string{}
	mentions := []string{}
	if text != nil {
		tags = ExtractHashtags(*text)
		for _, m := range ParseMentions(*text) {
			if m.Host != "" {
				continue
			}
			var id string
			if err := s.pool.QueryRow(ctx, `
SELECT id FROM users WHERE username_lower = lower($1) AND host IS NULL
`, m.Username).Scan(&id); err == nil {
				mentions = append(mentions, id)
			}
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("note: begin edit: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
UPDATE notes
SET text = $2, cw = $3, visibility = $4, local_only = $5,
    tags = $6, mentions = $7, emojis = '[]'::jsonb, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
`, noteID, text, cw, visibility, localOnly, tags, mentions); err != nil {
		return nil, fmt.Errorf("note: update note: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO note_edits (id, note_id, text, cw, file_ids) VALUES ($1, $2, $3, $4, $5)
`, idgen.New(), noteID, text, cw, n.FileIDs); err != nil {
		return nil, fmt.Errorf("note: record edit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("note: commit edit: %w", err)
	}
	return s.Get(ctx, noteID)
}

// NoteEdit is one saved revision returned by notes/history.
type NoteEdit struct {
	ID        string
	NoteID    string
	Text      *string
	CW        *string
	FileIDs   []string
	CreatedAt time.Time
}

// Edits returns the revisions of a note the user owns, newest first.
func (s *Service) Edits(ctx context.Context, userID, noteID string, limit int) ([]NoteEdit, error) {
	var owner string
	if err := s.pool.QueryRow(ctx, `SELECT user_id FROM notes WHERE id = $1`, noteID).Scan(&owner); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("note: load note owner: %w", err)
	}
	if owner != userID {
		return nil, ErrNotNoteAuthor
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, note_id, text, cw, file_ids, created_at
FROM note_edits WHERE note_id = $1 ORDER BY id DESC LIMIT $2
`, noteID, limit)
	if err != nil {
		return nil, fmt.Errorf("note: list edits: %w", err)
	}
	defer rows.Close()

	out := make([]NoteEdit, 0, limit)
	for rows.Next() {
		var e NoteEdit
		if err := rows.Scan(&e.ID, &e.NoteID, &e.Text, &e.CW, &e.FileIDs, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("note: scan edit: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
