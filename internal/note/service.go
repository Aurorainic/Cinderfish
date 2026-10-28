// Package note implements MilnFish's note (post) model: creation with
// the four visibility levels, the local-only flag, replies, renotes,
// reactions and polls.
//
// Visibility semantics follow misskey-dev/misskey:
//   - public: visible to everyone, appears in every timeline;
//   - home: visible to everyone but only in followers' home timelines;
//   - followers: visible to the author's followers and to users the
//     note replies to;
//   - specified: visible only to the listed users.
package note

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"milnfish/internal/idgen"
)

// Visibility levels.
const (
	VisibilityPublic    = "public"
	VisibilityHome      = "home"
	VisibilityFollowers = "followers"
	VisibilitySpecified = "specified"
)

// MaxTextLength matches Misskey's MAX_NOTE_TEXT_LENGTH.
const MaxTextLength = 3000

// Note is a stored note.
type Note struct {
	ID                 string
	UserID             string
	UserHost           *string
	Visibility         string
	LocalOnly          bool
	CW                 *string
	Text               *string
	PollID             *string
	ReplyID            *string
	ReplyUserID        *string
	RenoteID           *string
	RenoteUserID       *string
	RenoteUserHost     *string
	Name               *string
	URL                *string
	URI                *string
	Tags               []string
	Mentions           []string
	Emojis             []string
	FileIDs            []string
	RenoteCount        int
	RepliesCount       int
	ClippedCount       int
	IsSensitive        bool
	ReactionAcceptance *string
	CreatedAt          time.Time
	DeletedAt          *time.Time
}

// Poll is the poll attached to a note.
type Poll struct {
	ID        string
	Choices   []string
	Multiple  bool
	ExpiresAt *time.Time
}

// CreateParams carries the inputs of a note creation.
type CreateParams struct {
	UserID         string
	Visibility     string
	LocalOnly      bool
	CW             *string
	Text           *string
	FileIDs        []string
	VisibleUserIDs []string
	ReplyID        *string
	RenoteID       *string
	Poll           *PollParams
}

// PollParams describes a poll to create.
type PollParams struct {
	Choices   []string
	Multiple  bool
	ExpiresAt *time.Time
}

// Service provides note operations.
type Service struct {
	pool *pgxpool.Pool
}

// NewService returns a Service backed by the given pool.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Errors returned by the service.
var (
	ErrNotFound                 = errors.New("note: note not found")
	ErrNoSuchReplyTarget        = errors.New("note: no such reply target")
	ErrNoSuchRenoteTarget       = errors.New("note: no such renote target")
	ErrCannotRenoteToPureRenote = errors.New("note: cannot renote to a pure renote")
	ErrCannotReplyToPureRenote  = errors.New("note: cannot reply to a pure renote")
	ErrEmptyNote                = errors.New("note: a note needs text, files or a poll")
	ErrTextTooLong              = errors.New("note: text is too long")
	ErrInvalidVisibility        = errors.New("note: invalid visibility")
	ErrPollExpired              = errors.New("note: poll is already expired")
)

// Create stores a new note.
func (s *Service) Create(ctx context.Context, p CreateParams) (*Note, error) {
	if p.Visibility == "" {
		p.Visibility = VisibilityPublic
	}
	switch p.Visibility {
	case VisibilityPublic, VisibilityHome, VisibilityFollowers, VisibilitySpecified:
	default:
		return nil, ErrInvalidVisibility
	}
	// A pure renote carries no content of its own, so text is optional
	// when a renote target is given.
	if p.Text == nil && len(p.FileIDs) == 0 && p.Poll == nil && p.RenoteID == nil {
		return nil, ErrEmptyNote
	}
	if p.Text != nil && len(*p.Text) > MaxTextLength {
		return nil, ErrTextTooLong
	}
	if p.Visibility == VisibilitySpecified && len(p.VisibleUserIDs) == 0 {
		return nil, ErrInvalidVisibility
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("note: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var replyUserID *string
	if p.ReplyID != nil {
		parent, err := getNoteTx(ctx, tx, *p.ReplyID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrNoSuchReplyTarget
			}
			return nil, err
		}
		if parent.RenoteID != nil {
			return nil, ErrCannotReplyToPureRenote
		}
		replyUserID = &parent.UserID
	}

	var renoteUserID *string
	var renoteUserHost *string
	if p.RenoteID != nil {
		target, err := getNoteTx(ctx, tx, *p.RenoteID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrNoSuchRenoteTarget
			}
			return nil, err
		}
		if target.RenoteID != nil {
			return nil, ErrCannotRenoteToPureRenote
		}
		renoteUserID = &target.UserID
		renoteUserHost = target.UserHost
	}

	// Resolve the local users the text mentions. Remote mentions are
	// kept for federation addressing but do not enter mentions[], which
	// holds local user ids the way Misskey's column does.
	mentions := []string{}
	if p.Text != nil {
		for _, m := range ParseMentions(*p.Text) {
			mention := m
			if mention.Host != "" {
				continue
			}
			var id string
			if err := tx.QueryRow(ctx, `
SELECT id FROM users WHERE username_lower = lower($1) AND host IS NULL
			`, mention.Username).Scan(&id); err == nil {
				mentions = append(mentions, id)
			}
		}
	}

	now := time.Now()
	if p.FileIDs == nil {
		// The column is NOT NULL, so an empty array is needed rather
		// than a nil slice.
		p.FileIDs = []string{}
	}
	n := &Note{
		ID:             idgen.New(),
		UserID:         p.UserID,
		Visibility:     p.Visibility,
		LocalOnly:      p.LocalOnly,
		CW:             p.CW,
		Text:           p.Text,
		ReplyID:        p.ReplyID,
		ReplyUserID:    replyUserID,
		RenoteID:       p.RenoteID,
		RenoteUserID:   renoteUserID,
		RenoteUserHost: renoteUserHost,
		Tags:           []string{},
		Mentions:       mentions,
		Emojis:         []string{},
		CreatedAt:      now,
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO notes (id, user_id, visibility, local_only, cw, text, file_ids,
                   reply_id, reply_user_id, renote_id, renote_user_id, renote_user_host,
                   tags, mentions, emojis, created_at)
VALUES ($1,$2,$3,$4,$5,$6,
        COALESCE($7::text[], '{}'::text[]),
        $8,$9,$10,$11,$12,$13,$14,$15,$16)
`, n.ID, n.UserID, n.Visibility, n.LocalOnly, n.CW, n.Text, n.FileIDs,
		n.ReplyID, n.ReplyUserID, n.RenoteID, n.RenoteUserID, n.RenoteUserHost,
		mustJSON(n.Tags), mustJSON(n.Mentions), mustJSON(n.Emojis), n.CreatedAt); err != nil {
		return nil, fmt.Errorf("note: insert: %w", err)
	}

	for _, uid := range p.VisibleUserIDs {
		if _, err := tx.Exec(ctx, `
INSERT INTO note_visibilities (note_id, visible_user_id) VALUES ($1,$2)
ON CONFLICT DO NOTHING
`, n.ID, uid); err != nil {
			return nil, fmt.Errorf("note: insert visibility: %w", err)
		}
	}

	for _, fileID := range p.FileIDs {
		if _, err := tx.Exec(ctx, `
UPDATE drive_files SET id = id WHERE id = $1
`, fileID); err != nil {
			return nil, fmt.Errorf("note: check file: %w", err)
		}
	}

	if p.Poll != nil {
		// The poll shares the note's id, which the schema enforces.
		n.PollID = &n.ID
		if _, err := tx.Exec(ctx, `
INSERT INTO polls (id, choices, multiple, expires_at) VALUES ($1,$2,$3,$4)
`, n.ID, mustJSON(p.Poll.Choices), p.Poll.Multiple, p.Poll.ExpiresAt); err != nil {
			return nil, fmt.Errorf("note: insert poll: %w", err)
		}
	}

	// Keep the denormalised counters in step.
	if p.ReplyID != nil {
		if _, err := tx.Exec(ctx, `UPDATE notes SET replies_count = replies_count + 1 WHERE id = $1`, *p.ReplyID); err != nil {
			return nil, fmt.Errorf("note: bump replies: %w", err)
		}
	}
	if p.RenoteID != nil {
		if _, err := tx.Exec(ctx, `UPDATE notes SET renote_count = renote_count + 1 WHERE id = $1`, *p.RenoteID); err != nil {
			return nil, fmt.Errorf("note: bump renotes: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.Get(ctx, n.ID)
}

// Get loads a note by id.
func (s *Service) Get(ctx context.Context, id string) (*Note, error) {
	row := s.pool.QueryRow(ctx, noteSelect+` WHERE n.id = $1 AND n.deleted_at IS NULL`, id)
	return scanNote(row)
}

func getNoteTx(ctx context.Context, tx pgx.Tx, id string) (*Note, error) {
	row := tx.QueryRow(ctx, noteSelect+` WHERE n.id = $1 AND n.deleted_at IS NULL`, id)
	return scanNote(row)
}

// noteSelect is the shared column list for loading notes. The table is
// aliased as n so it can be used in the timeline queries.
const noteSelect = `
SELECT n.id, n.user_id, n.user_host, n.visibility, n.local_only, n.cw, n.text, n.file_ids,
       (SELECT id FROM polls WHERE polls.id = n.id) AS poll_id,
       n.reply_id, n.reply_user_id, n.renote_id, n.renote_user_id, n.renote_user_host,
       n.name, n.url, n.uri, n.tags, n.mentions, n.emojis,
       n.renote_count, n.replies_count, n.clipped_count, n.is_sensitive, n.reaction_acceptance,
       n.created_at, n.deleted_at
FROM notes n`

func scanNote(row pgx.Row) (*Note, error) {
	var n Note
	var tags, mentions, emojis []byte
	err := row.Scan(&n.ID, &n.UserID, &n.UserHost, &n.Visibility, &n.LocalOnly, &n.CW, &n.Text,
		&n.FileIDs, &n.PollID, &n.ReplyID, &n.ReplyUserID, &n.RenoteID, &n.RenoteUserID, &n.RenoteUserHost,
		&n.Name, &n.URL, &n.URI, &tags, &mentions, &emojis,
		&n.RenoteCount, &n.RepliesCount, &n.ClippedCount, &n.IsSensitive, &n.ReactionAcceptance,
		&n.CreatedAt, &n.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("note: scan: %w", err)
	}
	n.Tags = decodeStrings(tags)
	n.Mentions = decodeStrings(mentions)
	n.Emojis = decodeStrings(emojis)
	if n.FileIDs == nil {
		n.FileIDs = []string{}
	}
	return &n, nil
}

// VisibleUserIDs returns the users a specified-visibility note is
// addressed to.
func (s *Service) VisibleUserIDs(ctx context.Context, noteID string) []string {
	rows, err := s.pool.Query(ctx, `
SELECT visible_user_id FROM note_visibilities WHERE note_id = $1
`, noteID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return out
		}
		out = append(out, id)
	}
	return out
}

// Delete soft deletes a note owned by the given user.
func (s *Service) Delete(ctx context.Context, id, userID string) error {
	ct, err := s.pool.Exec(ctx, `
UPDATE notes SET deleted_at = now() WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
`, id, userID)
	if err != nil {
		return fmt.Errorf("note: delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Poll loads the poll of a note.
func (s *Service) Poll(ctx context.Context, noteID string) (*Poll, error) {
	var p Poll
	var choices []byte
	err := s.pool.QueryRow(ctx, `SELECT id, choices, multiple, expires_at FROM polls WHERE id = $1`, noteID).
		Scan(&p.ID, &choices, &p.Multiple, &p.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("note: scan poll: %w", err)
	}
	p.Choices = decodeStrings(choices)
	return &p, nil
}

// VoteCounts returns the number of votes per choice of a poll.
func (s *Service) VoteCounts(ctx context.Context, noteID string) (map[int]int, error) {
	rows, err := s.pool.Query(ctx, `
SELECT choice, count(*) FROM poll_votes WHERE note_id = $1 GROUP BY choice
`, noteID)
	if err != nil {
		return nil, fmt.Errorf("note: vote counts: %w", err)
	}
	defer rows.Close()

	counts := map[int]int{}
	for rows.Next() {
		var choice, n int
		if err := rows.Scan(&choice, &n); err != nil {
			return nil, fmt.Errorf("note: scan vote counts: %w", err)
		}
		counts[choice] = n
	}
	return counts, rows.Err()
}

// MyVote returns the choice a user voted for, or -1.
func (s *Service) MyVote(ctx context.Context, noteID, userID string) (int, error) {
	var choice int
	err := s.pool.QueryRow(ctx, `
SELECT choice FROM poll_votes WHERE note_id = $1 AND user_id = $2 LIMIT 1
`, noteID, userID).Scan(&choice)
	if errors.Is(err, pgx.ErrNoRows) {
		return -1, nil
	}
	if err != nil {
		return -1, fmt.Errorf("note: my vote: %w", err)
	}
	return choice, nil
}

// Vote records a vote in a poll.
func (s *Service) Vote(ctx context.Context, noteID, userID string, choice int) error {
	poll, err := s.Poll(ctx, noteID)
	if err != nil {
		return err
	}
	if poll.ExpiresAt != nil && poll.ExpiresAt.Before(time.Now()) {
		return ErrPollExpired
	}
	if choice < 0 || choice >= len(poll.Choices) {
		return errors.New("note: choice is out of range")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("note: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if !poll.Multiple {
		if _, err := tx.Exec(ctx, `DELETE FROM poll_votes WHERE note_id = $1 AND user_id = $2`, noteID, userID); err != nil {
			return fmt.Errorf("note: clear votes: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO poll_votes (note_id, user_id, choice) VALUES ($1,$2,$3)
ON CONFLICT DO NOTHING
`, noteID, userID, choice); err != nil {
		return fmt.Errorf("note: insert vote: %w", err)
	}
	return tx.Commit(ctx)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("[]")
	}
	return b
}

func decodeStrings(b []byte) []string {
	if len(b) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return []string{}
	}
	if out == nil {
		return []string{}
	}
	return out
}

// newID returns a fresh identifier for a reaction or notification.
func newID() string {
	return idgen.New()
}
