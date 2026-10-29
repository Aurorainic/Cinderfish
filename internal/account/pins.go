package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Errors returned by the pin operations.
var (
	// ErrNoSuchPinTarget is returned when the note to pin does not
	// exist or is deleted.
	ErrNoSuchPinTarget = errors.New("account: no such note to pin")
	// ErrNotOwnNote is returned when the note to pin is not owned by
	// the user.
	ErrNotOwnNote = errors.New("account: note is not owned by the user")
)

// PinnedNoteIDs returns the ids of the notes the user pinned, in the
// stored order (most recently pinned first).
func (s *Service) PinnedNoteIDs(ctx context.Context, userID string) ([]string, error) {
	var ids []string
	err := s.pool.QueryRow(ctx, `
SELECT pinned_note_ids FROM user_profiles WHERE user_id = $1
`, userID).Scan(&ids)
	if errors.Is(err, pgx.ErrNoRows) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("account: pinned notes: %w", err)
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}

// PinNote pins a note at the front of the user's pin list. Only the
// note's author may pin it; pinning a note that is already pinned is a
// no-op.
func (s *Service) PinNote(ctx context.Context, userID, noteID string) error {
	var owner string
	err := s.pool.QueryRow(ctx, `
SELECT user_id FROM notes WHERE id = $1 AND deleted_at IS NULL
`, noteID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoSuchPinTarget
	}
	if err != nil {
		return fmt.Errorf("account: pin note: %w", err)
	}
	if owner != userID {
		return ErrNotOwnNote
	}

	if _, err := s.pool.Exec(ctx, `
UPDATE user_profiles
SET pinned_note_ids = CASE WHEN $2::text = ANY(pinned_note_ids)
    THEN pinned_note_ids
    ELSE array_prepend($2::text, pinned_note_ids) END
WHERE user_id = $1
`, userID, noteID); err != nil {
		return fmt.Errorf("account: pin note: %w", err)
	}
	return nil
}

// UnpinNote removes a note from the user's pin list. Unpinning a note
// that is not pinned is a no-op.
func (s *Service) UnpinNote(ctx context.Context, userID, noteID string) error {
	if _, err := s.pool.Exec(ctx, `
UPDATE user_profiles
SET pinned_note_ids = array_remove(pinned_note_ids, $2::text)
WHERE user_id = $1
`, userID, noteID); err != nil {
		return fmt.Errorf("account: unpin note: %w", err)
	}
	return nil
}
