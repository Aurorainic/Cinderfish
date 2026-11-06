package note

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// MentionsParams is the filter set of `notes/mentions`.
type MentionsParams struct {
	ViewerID   string
	Following  bool
	Limit      int
	SinceID    string
	UntilID    string
	Visibility string
}

// Mentions returns notes that mention the viewer. `mentions` stores local user
// ids, so jsonb's `?` operator is a direct membership test.
//
// With Following set, only notes from accounts the viewer follows (plus the
// viewer's own notes) are returned, matching Misskey's "following" tab.
func (s *TimelineService) Mentions(ctx context.Context, p MentionsParams) ([]*Note, error) {
	if p.ViewerID == "" {
		return nil, fmt.Errorf("note: mentions requires a viewer")
	}
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}

	q := &queryBuilder{}
	q.add("n.mentions ? %s", p.ViewerID)
	q.addNoteVisibility(p.ViewerID)
	q.addViewerRelationFilters(p.ViewerID)

	if p.Following {
		me := q.viewerArg(p.ViewerID)
		q.conds = append(q.conds, fmt.Sprintf(
			"(n.user_id = %s OR EXISTS (SELECT 1 FROM followings f WHERE f.follower_id = %s AND f.followee_id = n.user_id))",
			me, me))
	}
	if p.Visibility != "" {
		q.add("n.visibility = %s", p.Visibility)
	}
	q.addPageCursors(p.SinceID, p.UntilID)

	query := noteSelect + ` WHERE ` + strings.Join(q.conds, " AND ") +
		` AND n.deleted_at IS NULL ORDER BY n.id ` + paginationOrder(p.SinceID, p.UntilID) +
		` LIMIT ` + strconv.Itoa(p.Limit)

	return s.scanNotes(ctx, query, q.args...)
}
