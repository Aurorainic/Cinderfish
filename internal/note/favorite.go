package note

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Aurorainic/cinderfish/internal/idgen"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoSuchFavorite is returned when the note a caller asked to favorite
// does not exist or is deleted.
var ErrNoSuchFavorite = errors.New("note: no such note to favorite")

// Favorite is one row of note_favorites joined with its note.
type Favorite struct {
	ID        string
	NoteID    string
	CreatedAt time.Time
	Note      *Note
}

// FavoriteService stores the notes a user marked with the star action
// (Misskey's notes/favorites/* endpoints and i/favorites).
type FavoriteService struct {
	pool *pgxpool.Pool
}

// NewFavoriteService returns a favorite service over the pool.
func NewFavoriteService(pool *pgxpool.Pool) *FavoriteService {
	return &FavoriteService{pool: pool}
}

// Add favorites a note for the user. The (user, note) pair is unique, so
// favoriting twice returns the existing row instead of an error, matching
// Misskey's idempotent create. The note must exist and not be deleted.
func (s *FavoriteService) Add(ctx context.Context, userID, noteID string) (*Favorite, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM notes WHERE id = $1 AND deleted_at IS NULL)
`, noteID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("note: favorite, check note: %w", err)
	}
	if !exists {
		return nil, ErrNoSuchFavorite
	}

	if _, err := s.pool.Exec(ctx, `
INSERT INTO note_favorites (id, user_id, note_id)
VALUES ($1,$2,$3) ON CONFLICT (user_id, note_id) DO NOTHING
	`, idgen.New(), userID, noteID); err != nil {
		return nil, fmt.Errorf("note: favorite: %w", err)
	}

	var fav Favorite
	var noteIDScanned string
	err := s.pool.QueryRow(ctx, `
SELECT id, note_id, created_at FROM note_favorites
WHERE user_id = $1 AND note_id = $2
	`, userID, noteID).Scan(&fav.ID, &noteIDScanned, &fav.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("note: load favorite: %w", err)
	}
	fav.NoteID = noteIDScanned
	return &fav, nil
}

// Remove deletes the user's favorite of the note. Removing one that is not
// there is not an error: Misskey's delete endpoint is idempotent.
func (s *FavoriteService) Remove(ctx context.Context, userID, noteID string) error {
	if _, err := s.pool.Exec(ctx, `
DELETE FROM note_favorites WHERE user_id = $1 AND note_id = $2
	`, userID, noteID); err != nil {
		return fmt.Errorf("note: unfavorite: %w", err)
	}
	return nil
}

// FavoriteQuery is the paging parameter set of i/favorites.
type FavoriteQuery struct {
	Limit     int
	SinceID   string
	UntilID   string
	SinceDate *time.Time
	UntilDate *time.Time
}

// List returns the user's favorites, newest first, with each note loaded.
// A favorite whose note has been deleted is skipped so the page never
// carries a null note.
func (s *FavoriteService) List(ctx context.Context, userID string, q FavoriteQuery) ([]*Favorite, error) {
	if q.Limit <= 0 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}

	var conds []string
	var args []any
	args = append(args, userID)
	conds = append(conds, fmt.Sprintf("user_id = $%d", len(args)))
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

	// makePaginationQuery semantics: a lower bound alone pages ascending,
	// everything else newest first.
	order := "id DESC"
	if q.SinceID != "" && q.UntilID == "" && q.UntilDate == nil {
		order = "id ASC"
	}

	query := `SELECT id, note_id, created_at FROM note_favorites WHERE ` +
		strings.Join(conds, " AND ") + ` ORDER BY ` + order + ` LIMIT ` + strconv.Itoa(q.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("note: list favorites: %w", err)
	}
	defer rows.Close()

	type row struct {
		fav Favorite
	}
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.fav.ID, &r.fav.NoteID, &r.fav.CreatedAt); err != nil {
			return nil, fmt.Errorf("note: scan favorite: %w", err)
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]*Favorite, 0, len(pending))
	for _, r := range pending {
		note, err := scanNote(s.pool.QueryRow(ctx, noteSelect+` WHERE n.id = $1 AND n.deleted_at IS NULL`, r.fav.NoteID))
		if errors.Is(err, ErrNotFound) {
			// The note was deleted between the row read and this load.
			continue
		}
		if err != nil {
			return nil, err
		}
		fav := r.fav
		fav.Note = note
		out = append(out, &fav)
	}
	return out, nil
}
