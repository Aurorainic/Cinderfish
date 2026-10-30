package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// MaxPinnedNotes is the number of notes a user may pin.
const MaxPinnedNotes = 5

// Errors returned by the pin operations.
var (
	// ErrNoSuchPinTarget is returned when the note to pin or unpin
	// does not exist or is deleted.
	ErrNoSuchPinTarget = errors.New("account: no such note to pin")
	// ErrNotOwnNote is returned when the note to pin or unpin is not
	// owned by the user.
	ErrNotOwnNote = errors.New("account: note is not owned by the user")
	// ErrPinLimitExceeded is returned when the user already pinned the
	// maximum number of notes.
	ErrPinLimitExceeded = errors.New("account: pin limit exceeded")
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
// no-op, and the list holds at most MaxPinnedNotes notes.
//
// The read-check-write sequence runs inside one transaction that locks
// the user's profile row, so concurrent pins cannot both observe a
// short list and push it past the limit or store the same id twice.
func (s *Service) PinNote(ctx context.Context, userID, noteID string) error {
	if err := s.pinTarget(ctx, noteID, userID); err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("account: pin note: %w", err)
	}
	// Rollback after a successful commit is a no-op.
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	// A missing profile row means an empty pin list, matching
	// PinnedNoteIDs; the update below then affects no rows.
	var pinned []string
	err = tx.QueryRow(ctx, `
SELECT pinned_note_ids FROM user_profiles WHERE user_id = $1 FOR UPDATE
`, userID).Scan(&pinned)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("account: pin note: %w", err)
	}

	for _, id := range pinned {
		if id == noteID {
			return nil
		}
	}
	if len(pinned) >= MaxPinnedNotes {
		return ErrPinLimitExceeded
	}

	if _, err := tx.Exec(ctx, `
UPDATE user_profiles
SET pinned_note_ids = array_prepend($2::text, pinned_note_ids)
WHERE user_id = $1
`, userID, noteID); err != nil {
		return fmt.Errorf("account: pin note: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("account: pin note: %w", err)
	}
	return nil
}

// UnpinNote removes a note from the user's pin list. The note must exist
// and belong to the user — Misskey answers NO_SUCH_NOTE otherwise so a
// note's existence is not leaked. Unpinning an owned note that is not
// pinned is a no-op.
func (s *Service) UnpinNote(ctx context.Context, userID, noteID string) error {
	if err := s.pinTarget(ctx, noteID, userID); err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `
UPDATE user_profiles
SET pinned_note_ids = array_remove(pinned_note_ids, $2::text)
WHERE user_id = $1
`, userID, noteID); err != nil {
		return fmt.Errorf("account: unpin note: %w", err)
	}
	return nil
}

// pinTarget verifies that the note exists, is not deleted, and belongs
// to the given user.
func (s *Service) pinTarget(ctx context.Context, noteID, userID string) error {
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
	return nil
}
