package note

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TimelineKind selects which notes a timeline contains.
type TimelineKind string

const (
	// TimelineHome holds the notes of the accounts the viewer follows.
	TimelineHome TimelineKind = "home"
	// TimelineLocal holds the public notes of local accounts.
	TimelineLocal TimelineKind = "local"
	// TimelineSocial holds both home and local notes.
	TimelineSocial TimelineKind = "social"
	// TimelineGlobal holds the public notes of every known account.
	TimelineGlobal TimelineKind = "global"
)

// TimelineParams carries the query parameters of a timeline request.
type TimelineParams struct {
	Kind        TimelineKind
	ViewerID    string
	Limit       int
	SinceID     string
	UntilID     string
	WithFiles   bool
	WithRenotes bool
	WithReplies bool
	// IncludeMyRenotes, IncludeRenotedMyNotes and IncludeLocalRenotes
	// default to true, matching Misskey's defaults.
	IncludeMyRenotes      bool
	IncludeRenotedMyNotes bool
	IncludeLocalRenotes   bool
}

// TimelineService answers timeline queries.
type TimelineService struct {
	pool *pgxpool.Pool
}

// NewTimelineService returns a TimelineService backed by the given pool.
func NewTimelineService(pool *pgxpool.Pool) *TimelineService {
	return &TimelineService{pool: pool}
}

// queryBuilder collects conditions and their arguments in order.
type queryBuilder struct {
	conds []string
	args  []any
}

// add appends a condition. Any %s in the condition is replaced with a
// fresh parameter placeholder, in order.
func (q *queryBuilder) add(cond string, vals ...any) {
	placeholders := make([]any, len(vals))
	for i, v := range vals {
		q.args = append(q.args, v)
		placeholders[i] = fmt.Sprintf("$%d", len(q.args))
	}
	q.conds = append(q.conds, fmt.Sprintf(cond, placeholders...))
}

// viewerArg adds the viewer id as a parameter and returns its
// placeholder. It must only be called once per distinct value.
func (q *queryBuilder) viewerArg(viewerID string) string {
	q.args = append(q.args, viewerID)
	return fmt.Sprintf("$%d", len(q.args))
}

// addNoteVisibility adds Misskey's generateVisibilityQuery
// (packages/backend/src/core/QueryService.ts L270-315): public and home
// notes are readable by everyone, a followers-only note by its author,
// the author's followers and the user it answers, and a specified note
// by its author and its audience. An empty viewerID is the anonymous
// visitor, who only sees public and home notes.
func (q *queryBuilder) addNoteVisibility(viewerID string) {
	if viewerID == "" {
		q.conds = append(q.conds, "n.visibility IN ('public','home')")
		return
	}
	me := q.viewerArg(viewerID)
	q.conds = append(q.conds, fmt.Sprintf(`(
		n.visibility IN ('public','home')
		OR n.user_id = %s
		OR EXISTS (SELECT 1 FROM note_visibilities nv WHERE nv.note_id = n.id AND nv.visible_user_id = %s)
		OR EXISTS (SELECT 1 FROM jsonb_array_elements_text(n.mentions) m WHERE m = %s)
		OR (n.visibility = 'followers' AND (
			EXISTS (SELECT 1 FROM followings f WHERE f.follower_id = %s AND f.followee_id = n.user_id)
			OR n.reply_user_id = %s
		))
	)`, me, me, me, me, me))
}

// addPageCursors appends the id cursors of a page request.
func (q *queryBuilder) addPageCursors(sinceID, untilID string) {
	if sinceID != "" {
		q.add("n.id > %s", sinceID)
	}
	if untilID != "" {
		q.add("n.id < %s", untilID)
	}
}

// paginationOrder mirrors Misskey's makePaginationQuery
// (packages/backend/src/core/QueryService.ts L54-62): a sinceId without
// untilId pages upward in ascending order, the direction a "load newer"
// cursor expects; every other combination is newest first.
func paginationOrder(sinceID, untilID string) string {
	if sinceID != "" && untilID == "" {
		return "ASC"
	}
	return "DESC"
}

// Timeline returns the notes of a timeline, newest first.
//
// The visibility rules mirror Misskey's generateVisibilityQuery: a note
// is visible to the viewer when it is public or home, or the viewer is
// the author, or the viewer is listed in the note's visible users, or
// the note mentions the viewer, or the viewer follows the author of a
// followers-only note.
func (s *TimelineService) Timeline(ctx context.Context, p TimelineParams) ([]*Note, error) {
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if !p.WithRenotes {
		p.IncludeMyRenotes = true
		p.IncludeRenotedMyNotes = true
		p.IncludeLocalRenotes = true
	}

	q := &queryBuilder{}

	switch p.Kind {
	case TimelineLocal:
		// The local timeline holds public notes of local accounts only.
		q.conds = append(q.conds, "n.visibility = 'public'", "n.user_host IS NULL")

	case TimelineGlobal:
		q.conds = append(q.conds, "n.visibility = 'public'")

	case TimelineHome:
		me := q.viewerArg(p.ViewerID)
		q.conds = append(q.conds, fmt.Sprintf(
			"(n.user_id = %s OR n.user_id IN (SELECT followee_id FROM followings WHERE follower_id = %s))", me, me))

	case TimelineSocial:
		me := q.viewerArg(p.ViewerID)
		q.conds = append(q.conds, fmt.Sprintf(
			"(n.user_id = %s OR n.user_id IN (SELECT followee_id FROM followings WHERE follower_id = %s) OR (n.visibility = 'public' AND n.user_host IS NULL))", me, me))
	default:
		return nil, fmt.Errorf("note: unknown timeline %q", p.Kind)
	}

	// Visibility filter.
	q.addNoteVisibility(p.ViewerID)

	// Replies are excluded unless asked for. A reply to a note the viewer
	// wrote is kept, matching Misskey's rule that a reply to one's own
	// note stays in the timeline.
	if !p.WithReplies {
		if p.ViewerID != "" {
			me := q.viewerArg(p.ViewerID)
			q.conds = append(q.conds, fmt.Sprintf(
				"(n.reply_id IS NULL OR EXISTS (SELECT 1 FROM notes parent WHERE parent.id = n.reply_id AND parent.user_id = %s))",
				me))
		} else {
			q.conds = append(q.conds, "n.reply_id IS NULL")
		}
	}

	if p.WithFiles {
		q.conds = append(q.conds, "EXISTS (SELECT 1 FROM drive_files df WHERE df.id = ANY(n.file_ids))")
	}

	// Cursor pagination. Note ids sort chronologically as strings.
	if p.UntilID != "" {
		q.add("n.id < %s", p.UntilID)
	}
	if p.SinceID != "" {
		q.add("n.id > %s", p.SinceID)
	}

	// Renote inclusion filters.
	if !p.IncludeMyRenotes {
		me := q.viewerArg(p.ViewerID)
		q.conds = append(q.conds, fmt.Sprintf("(n.user_id <> %s OR n.renote_id IS NULL OR n.text IS NOT NULL)", me))
	}
	if !p.IncludeRenotedMyNotes {
		me := q.viewerArg(p.ViewerID)
		q.conds = append(q.conds, fmt.Sprintf("(n.renote_user_id IS DISTINCT FROM %s OR n.renote_id IS NULL OR n.text IS NOT NULL)", me))
	}
	// When local renotes are excluded, renotes originating from a remote
	// account are dropped unless the renote carries content of its own.
	if !p.IncludeLocalRenotes {
		q.conds = append(q.conds, "(n.renote_user_host IS NULL OR n.renote_id IS NULL OR n.text IS NOT NULL)")
	}

	query := noteSelect + ` WHERE ` + strings.Join(q.conds, " AND ") +
		` AND n.deleted_at IS NULL ORDER BY n.id DESC LIMIT ` + strconv.Itoa(p.Limit)

	rows, err := s.pool.Query(ctx, query, q.args...)
	if err != nil {
		return nil, fmt.Errorf("note: timeline query: %w", err)
	}
	defer rows.Close()

	var out []*Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// SinceIDForDate returns the id of the oldest note newer than the given
// time, so clients can use sinceDate and untilDate cursors.
func (s *TimelineService) SinceIDForDate(ctx context.Context, t time.Time) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
SELECT id FROM notes WHERE created_at >= $1 AND deleted_at IS NULL ORDER BY id ASC LIMIT 1
`, t).Scan(&id)
	if err != nil {
		return "", nil
	}
	return id, nil
}

// UserNotes returns a user's publicly federable notes, newest first.
//
// This is the outbox query: public and home visibility only, local-only
// notes excluded, and cursor-bounded the same way the timeline is so
// peers see a stable order across pages. withFiles keeps only the notes
// that carry drive files; withReplies=false drops replies to other
// users while keeping replies to one's own notes.
func (s *TimelineService) UserNotes(ctx context.Context, userID string, limit int, sinceID, untilID string, withFiles, withReplies bool) ([]*Note, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	q := &queryBuilder{}
	q.add("n.user_id = %s", userID)
	q.add("n.user_host IS NULL")
	q.conds = append(q.conds, "n.deleted_at IS NULL", "n.local_only = FALSE", "n.visibility IN ('public','home')")
	if withFiles {
		q.conds = append(q.conds, "n.file_ids <> '{}'")
	}
	if !withReplies {
		q.add("(n.reply_id IS NULL OR n.reply_user_id = n.user_id)")
	}
	if sinceID != "" {
		q.add("n.id > %s", sinceID)
	}
	if untilID != "" {
		q.add("n.id < %s", untilID)
	}
	query := noteSelect + " WHERE " + strings.Join(q.conds, " AND ") +
		" ORDER BY n.id DESC LIMIT " + strconv.Itoa(limit)

	rows, err := s.pool.Query(ctx, query, q.args...)
	if err != nil {
		return nil, fmt.Errorf("note: user notes query: %w", err)
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

// Replies returns the notes that answer noteID, newest first unless a
// sinceId is given without untilId.
//
// This is Misskey's notes/replies: the same generateVisibilityQuery
// filter the timelines use, applied to notes whose reply_id is noteID.
// Its generateBaseNoteFilteringQuery only removes notes of blocked,
// muted and suspended users, which Cinderfish does not model yet, so
// the soft-delete exclusion is the only further filter. local_only notes
// stay visible, as they are in the timelines: every reader here is a
// local account.
func (s *TimelineService) Replies(ctx context.Context, noteID, viewerID string, limit int, sinceID, untilID string) ([]*Note, error) {
	return s.notesLinkedTo(ctx, "reply_id", "reply", noteID, viewerID, limit, sinceID, untilID)
}

// Renotes returns the notes whose renote_id is noteID, newest first
// unless a sinceId is given without untilId. Both pure renotes and quotes
// carry renote_id, and both are included, as upstream notes/renotes does.
func (s *TimelineService) Renotes(ctx context.Context, noteID, viewerID string, limit int, sinceID, untilID string) ([]*Note, error) {
	return s.notesLinkedTo(ctx, "renote_id", "renote", noteID, viewerID, limit, sinceID, untilID)
}

// notesLinkedTo is the shared query behind Replies and Renotes: notes
// whose column names noteID, filtered by the viewer's visibility and
// cursor-bounded the way Misskey's makePaginationQuery does.
func (s *TimelineService) notesLinkedTo(ctx context.Context, column, label, noteID, viewerID string, limit int, sinceID, untilID string) ([]*Note, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	q := &queryBuilder{}
	q.add("n."+column+" = %s", noteID)
	q.conds = append(q.conds, "n.deleted_at IS NULL")
	q.addNoteVisibility(viewerID)
	q.addPageCursors(sinceID, untilID)

	query := noteSelect + " WHERE " + strings.Join(q.conds, " AND ") +
		" ORDER BY n.id " + paginationOrder(sinceID, untilID) + " LIMIT " + strconv.Itoa(limit)

	rows, err := s.pool.Query(ctx, query, q.args...)
	if err != nil {
		return nil, fmt.Errorf("note: %s query: %w", label, err)
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
