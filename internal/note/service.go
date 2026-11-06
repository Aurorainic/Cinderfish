// Package note implements CinderFish's note (post) model: creation with
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
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/idgen"
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
	ID             string
	UserID         string
	UserHost       *string
	Visibility     string
	LocalOnly      bool
	CW             *string
	Text           *string
	PollID         *string
	ReplyID        *string
	ReplyUserID    *string
	RenoteID       *string
	RenoteUserID   *string
	RenoteUserHost *string
	Name           *string
	URL            *string
	URI            *string
	Tags           []string
	Mentions       []string
	// RemoteMentions holds user@host acct strings for mentioned remote
	// accounts; Mentions stays the local user ids.
	RemoteMentions     []string
	Emojis             []Emoji
	FileIDs            []string
	RenoteCount        int
	RepliesCount       int
	ClippedCount       int
	IsSensitive        bool
	ReactionAcceptance *string
	CreatedAt          time.Time
	UpdatedAt          *time.Time
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
	// Mirrors Misskey's cannotRenoteDueToVisibility: a followers-only note
	// can only be renoted by its author, and a specified note cannot be
	// renoted at all (core/NoteCreateService.ts L352-356).
	ErrCannotRenoteDueToVisibility = errors.New("note: renote target visibility forbids the renote")
	// Mirrors Misskey's cannotReplyToInvisibleNote: the reply target must
	// be readable by the replier (core/entities/NoteEntityService.ts
	// isVisibleForMe).
	ErrCannotReplyToInvisibleNote = errors.New("note: reply target is not visible to the replier")
	// Mirrors Misskey's
	// cannotReplyToSpecifiedVisibilityNoteWithExtendedVisibility: a
	// specified note may only be answered with another specified note, so
	// its audience is never widened (core/NoteCreateService.ts L388-390).
	ErrCannotWidenSpecifiedReply = errors.New("note: a specified note can only be answered by a specified note")
	ErrEmptyNote                 = errors.New("note: a note needs text, files or a poll")
	ErrTextTooLong               = errors.New("note: text is too long")
	ErrInvalidVisibility         = errors.New("note: invalid visibility")
	ErrPollExpired               = errors.New("note: poll is already expired")
	ErrNoPoll                    = errors.New("note: the note has no poll")
	ErrAlreadyVoted              = errors.New("note: already voted in this poll")
	ErrChoiceOutOfRange          = errors.New("note: poll choice is out of range")
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
		if parent.IsPureRenote() {
			return nil, ErrCannotReplyToPureRenote
		}
		// Misskey's order: existence, pure renote, visibility, then the
		// specified-audience rule (core/NoteCreateService.ts L374-406).
		visible, err := noteVisibleToTx(ctx, tx, parent, p.UserID)
		if err != nil {
			return nil, err
		}
		if !visible {
			return nil, ErrCannotReplyToInvisibleNote
		}
		if parent.Visibility == VisibilitySpecified && p.Visibility != VisibilitySpecified {
			return nil, ErrCannotWidenSpecifiedReply
		}
		// A reply never widens its target's audience either
		// (core/NoteCreateService.ts L544-573): a public reply to a home
		// note becomes home, a public or home reply to a followers note
		// becomes followers, and a local-only target keeps the reply
		// local.
		switch parent.Visibility {
		case VisibilityHome:
			if p.Visibility == VisibilityPublic {
				p.Visibility = VisibilityHome
			}
		case VisibilityFollowers:
			if p.Visibility == VisibilityPublic || p.Visibility == VisibilityHome {
				p.Visibility = VisibilityFollowers
			}
		}
		if parent.LocalOnly {
			p.LocalOnly = true
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
		if target.IsPureRenote() {
			return nil, ErrCannotRenoteToPureRenote
		}
		if target.Visibility == VisibilityFollowers && target.UserID != p.UserID {
			return nil, ErrCannotRenoteDueToVisibility
		}
		if target.Visibility == VisibilitySpecified {
			return nil, ErrCannotRenoteDueToVisibility
		}
		// A renote never widens its target's audience
		// (core/NoteCreateService.ts L499-530): home stays at most home,
		// an own followers note stays at most followers, and a local-only
		// note keeps a renote local.
		switch target.Visibility {
		case VisibilityHome:
			if p.Visibility == VisibilityPublic {
				p.Visibility = VisibilityHome
			}
		case VisibilityFollowers:
			if p.Visibility == VisibilityPublic || p.Visibility == VisibilityHome {
				p.Visibility = VisibilityFollowers
			}
		}
		if target.LocalOnly {
			p.LocalOnly = true
		}
		renoteUserID = &target.UserID
		renoteUserHost = target.UserHost
	}

	// Resolve the users the text mentions. Local ones go into mentions[] as
	// ids, the way Misskey's column works; remote ones are kept as user@host in
	// remote_mentions so rendering and delivery can resolve them.
	mentions := []string{}
	remoteMentions := []string{}
	if p.Text != nil {
		for _, m := range ParseMentions(*p.Text) {
			mention := m
			if mention.Host != "" {
				remoteMentions = appendUnique(remoteMentions, strings.ToLower(mention.Username+"@"+mention.Host))
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
	tags := []string{}
	if p.Text != nil {
		tags = ExtractHashtags(*p.Text)
	}

	n := &Note{
		ID:             idgen.New(),
		UserID:         p.UserID,
		Visibility:     p.Visibility,
		LocalOnly:      p.LocalOnly,
		CW:             p.CW,
		Text:           p.Text,
		FileIDs:        p.FileIDs,
		ReplyID:        p.ReplyID,
		ReplyUserID:    replyUserID,
		RenoteID:       p.RenoteID,
		RenoteUserID:   renoteUserID,
		RenoteUserHost: renoteUserHost,
		Tags:           tags,
		Mentions:       mentions,
		RemoteMentions: remoteMentions,
		Emojis:         []Emoji{},
		CreatedAt:      now,
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO notes (id, user_id, visibility, local_only, cw, text, file_ids,
                   reply_id, reply_user_id, renote_id, renote_user_id, renote_user_host,
                   tags, mentions, remote_mentions, emojis, created_at)
VALUES ($1,$2,$3,$4,$5,$6,
        COALESCE($7::text[], '{}'::text[]),
        $8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
`, n.ID, n.UserID, n.Visibility, n.LocalOnly, n.CW, n.Text, n.FileIDs,
		n.ReplyID, n.ReplyUserID, n.RenoteID, n.RenoteUserID, n.RenoteUserHost,
		mustJSON(n.Tags), mustJSON(n.Mentions), mustJSON(n.RemoteMentions), mustJSON(n.Emojis), n.CreatedAt); err != nil {
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

// noteVisibleToTx reports whether userID may read n, mirroring Misskey's
// NoteEntityService.isVisibleForMe
// (core/entities/NoteEntityService.ts L308-359): a specified note is
// visible to its author and its audience, a followers note to its author,
// the user it replies to, the users it mentions and its author's
// followers.
func noteVisibleToTx(ctx context.Context, tx pgx.Tx, n *Note, userID string) (bool, error) {
	switch n.Visibility {
	case VisibilitySpecified:
		if userID == n.UserID {
			return true, nil
		}
		var addressed bool
		if err := tx.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM note_visibilities WHERE note_id = $1 AND visible_user_id = $2)
`, n.ID, userID).Scan(&addressed); err != nil {
			return false, fmt.Errorf("note: check specified audience: %w", err)
		}
		return addressed, nil

	case VisibilityFollowers:
		if userID == n.UserID {
			return true, nil
		}
		if n.ReplyUserID != nil && *n.ReplyUserID == userID {
			return true, nil
		}
		for _, mentioned := range n.Mentions {
			if mentioned == userID {
				return true, nil
			}
		}
		var following bool
		if err := tx.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM followings WHERE followee_id = $1 AND follower_id = $2)
`, n.UserID, userID).Scan(&following); err != nil {
			return false, fmt.Errorf("note: check follow: %w", err)
		}
		if following {
			return true, nil
		}
		// Two remote users' follow relationship is unknowable here, so
		// upstream assumes it exists (NoteEntityService.ts L347-354).
		if n.UserHost == nil || *n.UserHost == "" {
			return false, nil
		}
		var viewerHost *string
		if err := tx.QueryRow(ctx, `SELECT host FROM users WHERE id = $1`, userID).Scan(&viewerHost); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return false, nil
			}
			return false, fmt.Errorf("note: check viewer host: %w", err)
		}
		return viewerHost != nil && *viewerHost != "", nil

	default:
		return true, nil
	}
}

// noteSelect is the shared column list for loading notes. The table is
// aliased as n so it can be used in the timeline queries.
const noteSelect = `
SELECT n.id, n.user_id, n.user_host, n.visibility, n.local_only, n.cw, n.text, n.file_ids,
       (SELECT id FROM polls WHERE polls.id = n.id) AS poll_id,
       n.reply_id, n.reply_user_id, n.renote_id, n.renote_user_id, n.renote_user_host,
       n.name, n.url, n.uri, n.tags, n.mentions, n.remote_mentions, n.emojis,
       n.renote_count, n.replies_count, n.clipped_count, n.is_sensitive, n.reaction_acceptance,
       n.created_at, n.updated_at, n.deleted_at
FROM notes n`

func scanNote(row pgx.Row) (*Note, error) {
	var n Note
	var tags, mentions, remoteMentions, emojis []byte
	err := row.Scan(&n.ID, &n.UserID, &n.UserHost, &n.Visibility, &n.LocalOnly, &n.CW, &n.Text,
		&n.FileIDs, &n.PollID, &n.ReplyID, &n.ReplyUserID, &n.RenoteID, &n.RenoteUserID, &n.RenoteUserHost,
		&n.Name, &n.URL, &n.URI, &tags, &mentions, &remoteMentions, &emojis,
		&n.RenoteCount, &n.RepliesCount, &n.ClippedCount, &n.IsSensitive, &n.ReactionAcceptance,
		&n.CreatedAt, &n.UpdatedAt, &n.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("note: scan: %w", err)
	}
	n.Tags = decodeStrings(tags)
	n.Mentions = decodeStrings(mentions)
	n.RemoteMentions = decodeStrings(remoteMentions)
	n.Emojis = decodeEmojis(emojis)
	if n.FileIDs == nil {
		n.FileIDs = []string{}
	}
	return &n, nil
}

// appendUnique appends s when absent.
func appendUnique(list []string, s string) []string {
	for _, v := range list {
		if v == s {
			return list
		}
	}
	return append(list, s)
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

// VisibleTo reports whether viewerID may read the note. An empty
// viewerID is the anonymous viewer, who only sees public and home notes.
func (s *Service) VisibleTo(ctx context.Context, n *Note, viewerID string) (bool, error) {
	if n.Visibility == VisibilityPublic || n.Visibility == VisibilityHome {
		return true, nil
	}
	if viewerID == "" {
		return false, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("note: begin visibility check: %w", err)
	}
	defer tx.Rollback(ctx)
	visible, err := noteVisibleToTx(ctx, tx, n, viewerID)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("note: commit visibility check: %w", err)
	}
	return visible, nil
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

// checkVoteAllowed enforces the duplicate-vote rules before anything is
// written.
func (s *Service) checkVoteAllowed(ctx context.Context, poll *Poll, noteID, userID string, choice int) error {
	existing, err := s.votedChoices(ctx, noteID, userID)
	if err != nil {
		return err
	}
	if len(existing) == 0 {
		return nil
	}
	if !poll.Multiple || existing[choice] {
		return ErrAlreadyVoted
	}
	return nil
}

// MyVotes returns every choice the user voted for in a poll; a single-choice
// poll has at most one. The packer uses it so multiple-choice polls show all
// the viewer's ticks.
func (s *Service) MyVotes(ctx context.Context, noteID, userID string) ([]int, error) {
	set, err := s.votedChoices(ctx, noteID, userID)
	if err != nil {
		return nil, err
	}
	out := make([]int, 0, len(set))
	for choice := range set {
		out = append(out, choice)
	}
	return out, nil
}

// votedChoices returns the set of choices the user already voted for.
func (s *Service) votedChoices(ctx context.Context, noteID, userID string) (map[int]bool, error) {
	rows, err := s.pool.Query(ctx, `
SELECT choice FROM poll_votes WHERE note_id = $1 AND user_id = $2
`, noteID, userID)
	if err != nil {
		return nil, fmt.Errorf("note: voted choices: %w", err)
	}
	defer rows.Close()

	out := map[int]bool{}
	for rows.Next() {
		var choice int
		if err := rows.Scan(&choice); err != nil {
			return nil, fmt.Errorf("note: scan voted choice: %w", err)
		}
		out[choice] = true
	}
	return out, rows.Err()
}

// Vote records a vote in a poll.
func (s *Service) Vote(ctx context.Context, noteID, userID string, choice int) error {
	poll, err := s.Poll(ctx, noteID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNoPoll
		}
		return err
	}
	if poll.ExpiresAt != nil && poll.ExpiresAt.Before(time.Now()) {
		return ErrPollExpired
	}
	if choice < 0 || choice >= len(poll.Choices) {
		return ErrChoiceOutOfRange
	}

	// Misskey rejects a second vote in a single-choice poll, and the same
	// choice twice in a multiple-choice one; only a different choice in a
	// multiple poll replaces nothing and simply adds a row.
	if err := s.checkVoteAllowed(ctx, poll, noteID, userID, choice); err != nil {
		return err
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

// Emoji is one custom emoji a note references: the shortcode without colons
// and the image URL a renderer should use.
type Emoji struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func decodeEmojis(b []byte) []Emoji {
	if len(b) == 0 {
		return []Emoji{}
	}
	var out []Emoji
	if err := json.Unmarshal(b, &out); err != nil {
		return []Emoji{}
	}
	if out == nil {
		return []Emoji{}
	}
	return out
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
