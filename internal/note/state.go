package note

import (
	"context"
	"fmt"
)

// NoteState is the per-viewer state returned by notes/state.
type NoteState struct {
	IsFavorited   bool
	IsPinned      bool
	IsMutedThread bool
}

// State reports whether the user favorited or pinned a note. Thread muting is
// not implemented, so isMutedThread is always false.
func (s *Service) State(ctx context.Context, userID, noteID string) (NoteState, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM notes WHERE id = $1 AND deleted_at IS NULL)
`, noteID).Scan(&exists); err != nil {
		return NoteState{}, fmt.Errorf("note: check note: %w", err)
	}
	if !exists {
		return NoteState{}, ErrNotFound
	}

	var state NoteState
	if err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM note_favorites WHERE user_id = $1 AND note_id = $2)
`, userID, noteID).Scan(&state.IsFavorited); err != nil {
		return NoteState{}, fmt.Errorf("note: favorite state: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM user_profiles
    WHERE user_id = $1 AND $2::text = ANY(pinned_note_ids)
)
`, userID, noteID).Scan(&state.IsPinned); err != nil {
		return NoteState{}, fmt.Errorf("note: pin state: %w", err)
	}
	return state, nil
}
