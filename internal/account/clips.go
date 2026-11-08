package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// Clip is one named collection of notes, owned by a local account. Public
// clips can be read and favourited by anyone; only the owner edits them
// or changes their membership.
type Clip struct {
	ID            string
	UserID        string
	Name          string
	IsPublic      bool
	Description   *string
	LastClippedAt *time.Time
	CreatedAt     time.Time
}

// Caps Misskey keeps in role policies (RoleService.ts L102-103): ten clips
// per user, 200 notes per clip. Cinderfish has no roles, so the defaults
// are fixed.
const (
	clipsPerUserLimit = 10
	clipNotesLimit    = 200
	clipNameMax       = 100
	clipDescMax       = 2048
)

// Sentinel errors the handlers map onto Misskey's error codes.
var (
	ErrClipNotFound         = errors.New("account: clip not found")
	ErrTooManyClips         = errors.New("account: too many clips")
	ErrTooManyClipNotes     = errors.New("account: too many notes in the clip")
	ErrAlreadyClipped       = errors.New("account: note already in the clip")
	ErrClipNameInvalid      = errors.New("account: clip name invalid")
	ErrClipDescInvalid      = errors.New("account: clip description invalid")
	ErrClipAlreadyFavorited = errors.New("account: clip already favorited")
	ErrClipNotFavorited     = errors.New("account: clip not favorited")
	ErrClipNoteMissing      = errors.New("account: note not in the clip")
	ErrClipNoteNotFound     = errors.New("account: no such note")
)

// CreateClip makes a clip owned by userID. An empty description string is
// stored as NULL, matching upstream's coercion.
func (s *Service) CreateClip(ctx context.Context, userID, name string, isPublic bool, description *string) (*Clip, error) {
	if name == "" || len([]rune(name)) > clipNameMax {
		return nil, ErrClipNameInvalid
	}
	var desc *string
	if description != nil {
		trimmed := strings.TrimSpace(*description)
		if len([]rune(trimmed)) > clipDescMax {
			return nil, ErrClipDescInvalid
		}
		if trimmed != "" {
			desc = &trimmed
		}
	}
	var n int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM clips WHERE user_id = $1`, userID).Scan(&n); err != nil {
		return nil, fmt.Errorf("account: count clips: %w", err)
	}
	if n >= clipsPerUserLimit {
		return nil, ErrTooManyClips
	}
	c := &Clip{ID: idgen.New(), UserID: userID, Name: name, IsPublic: isPublic, Description: desc, CreatedAt: time.Now()}
	if _, err := s.pool.Exec(ctx, `
INSERT INTO clips (id, user_id, name, is_public, description) VALUES ($1, $2, $3, $4, $5)
`, c.ID, c.UserID, c.Name, c.IsPublic, desc); err != nil {
		return nil, fmt.Errorf("account: create clip: %w", err)
	}
	return c, nil
}

// GetClip loads one clip. Misskey's show/notes lookup: the owner always
// reads their own; everyone else only public clips. A private clip read
// by anyone else reports not-found.
func (s *Service) GetClip(ctx context.Context, clipID, viewerID string) (*Clip, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, user_id, name, is_public, description, last_clipped_at, created_at
FROM clips WHERE id = $1
`, clipID)
	var c Clip
	err := row.Scan(&c.ID, &c.UserID, &c.Name, &c.IsPublic, &c.Description, &c.LastClippedAt, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrClipNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account: get clip: %w", err)
	}
	if c.UserID == viewerID {
		return &c, nil
	}
	if !c.IsPublic {
		return nil, ErrClipNotFound
	}
	return &c, nil
}

// OwnClip loads a clip that must belong to ownerID — the lookup every
// mutating endpoint does. A clip owned by someone else reports not-found,
// so private clip ids are not enumerable through the error.
func (s *Service) OwnClip(ctx context.Context, clipID, ownerID string) (*Clip, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, user_id, name, is_public, description, last_clipped_at, created_at
FROM clips WHERE id = $1
`, clipID)
	var c Clip
	err := row.Scan(&c.ID, &c.UserID, &c.Name, &c.IsPublic, &c.Description, &c.LastClippedAt, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrClipNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account: own clip: %w", err)
	}
	if c.UserID != ownerID {
		return nil, ErrClipNotFound
	}
	return &c, nil
}

// ListClipsOf returns one account's clips, newest first unless a sinceId
// is given without untilId. With onlyPublic it narrows to public clips —
// the users/clips branch, which carries the same pagination shape as
// clips/list upstream.
func (s *Service) ListClipsOf(ctx context.Context, userID string, onlyPublic bool, q ClipQuery) ([]*Clip, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	var sql strings.Builder
	sql.WriteString(`
SELECT id, user_id, name, is_public, description, last_clipped_at, created_at
FROM clips WHERE user_id = $1`)
	if onlyPublic {
		sql.WriteString(` AND is_public`)
	}
	args := []any{userID}
	if q.SinceID != "" {
		args = append(args, q.SinceID)
		fmt.Fprintf(&sql, " AND id > $%d", len(args))
	}
	if q.UntilID != "" {
		args = append(args, q.UntilID)
		fmt.Fprintf(&sql, " AND id < $%d", len(args))
	}
	if q.SinceDate != nil {
		args = append(args, *q.SinceDate)
		fmt.Fprintf(&sql, " AND created_at >= $%d", len(args))
	}
	if q.UntilDate != nil {
		args = append(args, *q.UntilDate)
		fmt.Fprintf(&sql, " AND created_at <= $%d", len(args))
	}
	order := "DESC"
	if q.SinceID != "" && q.UntilID == "" {
		order = "ASC"
	}
	sql.WriteString(" ORDER BY id " + order + " LIMIT " + fmt.Sprint(limit))
	return s.queryClips(ctx, sql.String(), args...)
}

func (s *Service) queryClips(ctx context.Context, sql string, args ...any) ([]*Clip, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("account: list clips: %w", err)
	}
	defer rows.Close()

	out := []*Clip{}
	for rows.Next() {
		var c Clip
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.IsPublic, &c.Description, &c.LastClippedAt, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("account: scan clip: %w", err)
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// ClipQuery pages a list request by clip id.
type ClipQuery struct {
	Limit   int
	SinceID string
	UntilID string
	SinceDate *time.Time
	UntilDate *time.Time
}

// ListClips pages the caller's clips, newest first unless a sinceId is
// given without untilId.
func (s *Service) ListClips(ctx context.Context, userID string, q ClipQuery) ([]*Clip, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var sql strings.Builder
	sql.WriteString(`
SELECT id, user_id, name, is_public, description, last_clipped_at, created_at
FROM clips WHERE user_id = $1`)
	args := []any{userID}
	if q.SinceID != "" {
		args = append(args, q.SinceID)
		fmt.Fprintf(&sql, " AND id > $%d", len(args))
	}
	if q.UntilID != "" {
		args = append(args, q.UntilID)
		fmt.Fprintf(&sql, " AND id < $%d", len(args))
	}
	if q.SinceDate != nil {
		args = append(args, *q.SinceDate)
		fmt.Fprintf(&sql, " AND created_at >= $%d", len(args))
	}
	if q.UntilDate != nil {
		args = append(args, *q.UntilDate)
		fmt.Fprintf(&sql, " AND created_at <= $%d", len(args))
	}
	order := "DESC"
	if q.SinceID != "" && q.UntilID == "" {
		order = "ASC"
	}
	sql.WriteString(" ORDER BY id " + order + " LIMIT " + fmt.Sprint(limit))
	return s.queryClips(ctx, sql.String(), args...)
}

// ListFavoriteClips returns the clips userID favourited. Upstream's
// my-favorites carries no pagination and no explicit order; the join
// order follows the favourite row ids here.
func (s *Service) ListFavoriteClips(ctx context.Context, userID string) ([]*Clip, error) {
	return s.queryClips(ctx, `
SELECT c.id, c.user_id, c.name, c.is_public, c.description, c.last_clipped_at, c.created_at
FROM clip_favorites f
JOIN clips c ON c.id = f.clip_id
WHERE f.user_id = $1
ORDER BY f.id DESC
`, userID)
}

// UpdateClip renames a clip or flips its visibility/description. A nil
// pointer leaves the field unchanged; an explicit empty description
// clears it, matching upstream's "" → NULL coercion. The returned clip
// carries the post-update row.
func (s *Service) UpdateClip(ctx context.Context, clipID, ownerID string, name *string, isPublic *bool, description *string) (*Clip, error) {
	c, err := s.OwnClip(ctx, clipID, ownerID)
	if err != nil {
		return nil, err
	}
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" || len([]rune(trimmed)) > clipNameMax {
			return nil, ErrClipNameInvalid
		}
		if _, err := s.pool.Exec(ctx, `UPDATE clips SET name = $1 WHERE id = $2`, trimmed, c.ID); err != nil {
			return nil, fmt.Errorf("account: update clip name: %w", err)
		}
		c.Name = trimmed
	}
	if isPublic != nil {
		if _, err := s.pool.Exec(ctx, `UPDATE clips SET is_public = $1 WHERE id = $2`, *isPublic, c.ID); err != nil {
			return nil, fmt.Errorf("account: update clip visibility: %w", err)
		}
		c.IsPublic = *isPublic
	}
	if description != nil {
		trimmed := strings.TrimSpace(*description)
		if len([]rune(trimmed)) > clipDescMax {
			return nil, ErrClipNameInvalid
		}
		var desc *string
		if trimmed != "" {
			desc = &trimmed
		}
		if _, err := s.pool.Exec(ctx, `UPDATE clips SET description = $1 WHERE id = $2`, desc, c.ID); err != nil {
			return nil, fmt.Errorf("account: update clip description: %w", err)
		}
		c.Description = desc
	}
	return c, nil
}

// DeleteClip removes a clip and, by cascade, its note memberships and
// favourites.
func (s *Service) DeleteClip(ctx context.Context, clipID, ownerID string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM clips WHERE id = $1 AND user_id = $2`, clipID, ownerID)
	if err != nil {
		return fmt.Errorf("account: delete clip: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrClipNotFound
	}
	return nil
}

// AddClipNote records a note in a clip, keeping notes.clipped_count in the
// same transaction. The unique constraint answers ALREADY_CLIPPED; a
// missing note fails the foreign key and reports NO_SUCH_NOTE-shaped
// ErrClipNoteNotFound.
func (s *Service) AddClipNote(ctx context.Context, clipID, ownerID, noteID string) error {
	c, err := s.OwnClip(ctx, clipID, ownerID)
	if err != nil {
		return err
	}
	var n int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM clip_notes WHERE clip_id = $1`, c.ID).Scan(&n); err != nil {
		return fmt.Errorf("account: count clip notes: %w", err)
	}
	if n >= clipNotesLimit {
		return ErrTooManyClipNotes
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("account: add clip note: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
INSERT INTO clip_notes (id, clip_id, note_id) VALUES ($1, $2, $3)
`, idgen.New(), c.ID, noteID); err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyClipped
		}
		if isForeignKeyViolation(err) {
			return ErrClipNoteNotFound
		}
		return fmt.Errorf("account: insert clip note: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE clips SET last_clipped_at = now() WHERE id = $1
`, c.ID); err != nil {
		return fmt.Errorf("account: touch clip: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE notes SET clipped_count = clipped_count + 1 WHERE id = $1
`, noteID); err != nil {
		return fmt.Errorf("account: bump clipped count: %w", err)
	}
	return tx.Commit(ctx)
}

// RemoveClipNote deletes a note's membership. A missing membership row is
// NO_SUCH_NOTE upstream (it never checks the note itself), mirrored here;
// the counter decrement rides the same transaction.
func (s *Service) RemoveClipNote(ctx context.Context, clipID, ownerID, noteID string) error {
	c, err := s.OwnClip(ctx, clipID, ownerID)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("account: remove clip note: %w", err)
	}
	defer tx.Rollback(ctx)

	ct, err := tx.Exec(ctx, `
DELETE FROM clip_notes WHERE clip_id = $1 AND note_id = $2
`, c.ID, noteID)
	if err != nil {
		return fmt.Errorf("account: delete clip note: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrClipNoteMissing
	}
	if _, err := tx.Exec(ctx, `
UPDATE clips SET last_clipped_at = now() WHERE id = $1
`, c.ID); err != nil {
		return fmt.Errorf("account: touch clip: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE notes SET clipped_count = GREATEST(clipped_count - 1, 0) WHERE id = $1
`, noteID); err != nil {
		return fmt.Errorf("account: drop clipped count: %w", err)
	}
	return tx.Commit(ctx)
}

// IsClipNoteMember reports whether noteID is in clipID.
func (s *Service) IsClipNoteMember(ctx context.Context, clipID, noteID string) (bool, error) {
	var m bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM clip_notes WHERE clip_id = $1 AND note_id = $2)
`, clipID, noteID).Scan(&m)
	if err != nil {
		return false, fmt.Errorf("account: check clip note: %w", err)
	}
	return m, nil
}

// ClipsOfNote returns the public clips containing noteID — Misskey's
// notes/clips, which never returns private clips even to their owner.
func (s *Service) ClipsOfNote(ctx context.Context, noteID string) ([]*Clip, error) {
	return s.queryClips(ctx, `
SELECT c.id, c.user_id, c.name, c.is_public, c.description, c.last_clipped_at, c.created_at
FROM clip_notes cn
JOIN clips c ON c.id = cn.clip_id
WHERE cn.note_id = $1 AND c.is_public
ORDER BY c.id DESC
`, noteID)
}

// FavoriteClip records a favourite of a visible clip. Favouriting an
// already-favourited clip is an error, as upstream's favorite is.
func (s *Service) FavoriteClip(ctx context.Context, userID, clipID string) error {
	if _, err := s.GetClip(ctx, clipID, userID); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO clip_favorites (id, user_id, clip_id) VALUES ($1, $2, $3)
`, idgen.New(), userID, clipID)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrClipAlreadyFavorited
		}
		return fmt.Errorf("account: favorite clip: %w", err)
	}
	return nil
}

// UnfavoriteClip removes a favourite. Unlike favorite, upstream does not
// re-check visibility here — only an existing favourite row matters.
func (s *Service) UnfavoriteClip(ctx context.Context, userID, clipID string) error {
	ct, err := s.pool.Exec(ctx, `
DELETE FROM clip_favorites WHERE user_id = $1 AND clip_id = $2
`, userID, clipID)
	if err != nil {
		return fmt.Errorf("account: unfavorite clip: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrClipNotFavorited
	}
	return nil
}

// CountClipFavorites returns how many users favourited a clip, for the
// packed favoritedCount, which upstream always fills.
func (s *Service) CountClipFavorites(ctx context.Context, clipID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM clip_favorites WHERE clip_id = $1`, clipID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("account: count clip favorites: %w", err)
	}
	return n, nil
}

// CountClipNotes returns how many notes a clip holds, for packClip's
// owner-only notesCount.
func (s *Service) CountClipNotes(ctx context.Context, clipID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM clip_notes cn JOIN notes n ON n.id = cn.note_id
		 WHERE cn.clip_id = $1 AND n.deleted_at IS NULL`, clipID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("account: count clip notes: %w", err)
	}
	return n, nil
}

// HasClipFavorite reports whether userID favourited clipID, for the
// packed isFavorited, which only travels when a viewer is known.
func (s *Service) HasClipFavorite(ctx context.Context, userID, clipID string) (bool, error) {
	var f bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM clip_favorites WHERE user_id = $1 AND clip_id = $2)
`, userID, clipID).Scan(&f)
	if err != nil {
		return false, fmt.Errorf("account: check clip favorite: %w", err)
	}
	return f, nil
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
