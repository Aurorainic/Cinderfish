package note

import (
	"context"
	"errors"
)

// IsPureRenote mirrors Misskey's isRenote(note) && !isQuote(note)
// (packages/backend/src/misc/is-renote.ts): a note carries a renote target
// and none of its own content — text, CW, a reply pointer, a poll or files.
// A note WITH any of those is a quote (isQuote), which stays replyable and
// renotable: NoteCreateService.ts rejects `isRenote && !isQuote` targets
// only (L335 for renotes, L384 for replies). The stored row must be loaded
// with its poll id and file ids (noteSelect) for the classification to be
// complete.
func (n *Note) IsPureRenote() bool {
	if n.RenoteID == nil {
		return false
	}
	if n.Text != nil || n.CW != nil || n.PollID != nil || n.ReplyID != nil || len(n.FileIDs) > 0 {
		return false
	}
	return true
}

// IsQuote reports whether the note renotes another one and adds content of
// its own (Misskey's isQuote in packages/backend/src/misc/is-renote.ts).
// It selects the "quote" notification type over "renote".
func (n *Note) IsQuote() bool {
	return n.RenoteID != nil && !n.IsPureRenote()
}

// PureRenoteOf returns the newest pure renote userID published of noteID, or
// nil when they have not renoted it. A quote is not a renote in this sense: it
// carries the user's own words, so "un-renote" must not delete it.
func (s *Service) PureRenoteOf(ctx context.Context, userID, noteID string) (*Note, error) {
	row := s.pool.QueryRow(ctx, noteSelect+`
 WHERE n.user_id = $1 AND n.renote_id = $2 AND n.deleted_at IS NULL
   AND n.text IS NULL AND n.cw IS NULL AND n.reply_id IS NULL
   AND coalesce(array_length(n.file_ids, 1), 0) = 0
   AND NOT EXISTS (SELECT 1 FROM polls WHERE polls.id = n.id)
 ORDER BY n.id DESC LIMIT 1`, userID, noteID)
	n, err := scanNote(row)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return n, nil
}
