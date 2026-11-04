package note

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Conversation returns the ancestors of a note: its reply chain, nearest
// first, with the same shape as Misskey's notes/conversation (offset skips
// that many ancestors, limit caps the page).
//
// The walk stops at the first ancestor the viewer may not read. Misskey's
// endpoint does not check visibility at all, but leaking a private parent
// through a public reply would expose content this instance already refuses
// to serve from notes/show, so the stricter rule applies here.
func (s *TimelineService) Conversation(ctx context.Context, noteID, viewerID string, limit, offset int) ([]*Note, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var parentID *string
	if err := s.pool.QueryRow(ctx, `
SELECT reply_id FROM notes WHERE id = $1 AND deleted_at IS NULL
	`, noteID).Scan(&parentID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("note: conversation, load note: %w", err)
	}

	out := []*Note{}
	skipped := 0
	seen := map[string]bool{noteID: true}
	current := parentID

	for current != nil && len(out) < limit {
		if seen[*current] {
			// A malformed chain that loops back on itself would otherwise
			// keep the walk alive until the limit.
			break
		}
		seen[*current] = true

		// Visibility and existence are one query: a note the viewer may not
		// read is indistinguishable from a missing one, so either way the
		// walk stops here.
		var b queryBuilder
		b.add("n.id = %s", *current)
		b.conds = append(b.conds, "n.deleted_at IS NULL")
		b.addNoteVisibility(viewerID)
		n, err := scanNote(s.pool.QueryRow(ctx,
			noteSelect+" WHERE "+strings.Join(b.conds, " AND "), b.args...))
		if err != nil {
			if err == ErrNotFound {
				break
			}
			return nil, err
		}

		if skipped < offset {
			skipped++
		} else {
			out = append(out, n)
		}
		current = n.ReplyID
	}
	return out, nil
}

// ChildrenQuery is the paging parameter set of notes/children.
type ChildrenQuery struct {
	Limit     int
	SinceID   string
	UntilID   string
	SinceDate *time.Time
	UntilDate *time.Time
}

// Children returns the replies to a note, newest first, filtered by what the
// viewer may read. Misskey's notes/children has the same shape.
func (s *TimelineService) Children(ctx context.Context, noteID, viewerID string, q ChildrenQuery) ([]*Note, error) {
	if q.Limit <= 0 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}

	var b queryBuilder
	b.add("n.reply_id = %s", noteID)
	b.conds = append(b.conds, "n.deleted_at IS NULL")
	b.addNoteVisibility(viewerID)
	b.addPageCursors(q.SinceID, q.UntilID)
	if q.SinceDate != nil {
		b.add("n.created_at > %s", *q.SinceDate)
	}
	if q.UntilDate != nil {
		b.add("n.created_at < %s", *q.UntilDate)
	}

	query := noteSelect + " WHERE " + strings.Join(b.conds, " AND ") +
		" ORDER BY n.id " + paginationOrder(q.SinceID, q.UntilID) +
		" LIMIT " + strconv.Itoa(q.Limit)

	rows, err := s.pool.Query(ctx, query, b.args...)
	if err != nil {
		return nil, fmt.Errorf("note: children query: %w", err)
	}
	defer rows.Close()

	out := []*Note{}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
