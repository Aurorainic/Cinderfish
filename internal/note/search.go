package note

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// SearchParams is the filter set of `notes/search`. The zero value of Host
// means "any instance"; a pointer to the empty string means "local only";
// a non-empty value names one remote host.
type SearchParams struct {
	Query     string
	ViewerID  string
	Limit     int
	Offset    int
	SinceID   string
	UntilID   string
	Host      *string
	UserID    string
	WithFiles bool
}

// SearchByTagParams is the filter set of `notes/search-by-tag`.
type SearchByTagParams struct {
	Tag       string
	ViewerID  string
	Limit     int
	SinceID   string
	UntilID   string
	WithFiles bool
}

// escapeLike escapes the three characters LIKE treats specially. The
// queries below rely on PostgreSQL's default backslash escape.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}

// Search answers `notes/search`. Without a full-text index this is the same
// ILIKE fallback Misskey uses when no search engine is configured: the query
// matches the note body only, and visibility, mutes, blocks and muted hosts
// apply exactly as they do on the timelines.
func (s *TimelineService) Search(ctx context.Context, p SearchParams) ([]*Note, error) {
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Offset < 0 {
		p.Offset = 0
	}

	q := &queryBuilder{}
	q.add("n.text ILIKE %s", "%"+escapeLike(p.Query)+"%")
	q.addNoteVisibility(p.ViewerID)
	q.addViewerRelationFilters(p.ViewerID)

	if p.UserID != "" {
		q.add("n.user_id = %s", p.UserID)
	}
	if p.Host != nil {
		if *p.Host == "" {
			q.conds = append(q.conds, "n.user_host IS NULL")
		} else {
			q.add("n.user_host = %s", *p.Host)
		}
	}
	if p.WithFiles {
		q.conds = append(q.conds, "EXISTS (SELECT 1 FROM drive_files df WHERE df.id = ANY(n.file_ids))")
	}
	q.addPageCursors(p.SinceID, p.UntilID)

	query := noteSelect + ` WHERE ` + strings.Join(q.conds, " AND ") +
		` AND n.deleted_at IS NULL ORDER BY n.id ` + paginationOrder(p.SinceID, p.UntilID) +
		` LIMIT ` + strconv.Itoa(p.Limit)
	if p.SinceID == "" && p.UntilID == "" && p.Offset > 0 {
		query += ` OFFSET ` + strconv.Itoa(p.Offset)
	}

	return s.scanNotes(ctx, query, q.args...)
}

// SearchByTag answers `notes/search-by-tag` for the single-tag form. Note
// creation keeps the author's casing in `notes.tags` (deduplication is case
// insensitive), so the match folds both sides.
func (s *TimelineService) SearchByTag(ctx context.Context, p SearchByTagParams) ([]*Note, error) {
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}

	q := &queryBuilder{}
	// Tags keep the author's casing in `notes.tags`, so compare case-folded
	// values instead of using jsonb's case-sensitive `?` operator.
	q.add("EXISTS (SELECT 1 FROM jsonb_array_elements_text(n.tags) t WHERE lower(t) = lower(%s))", p.Tag)
	q.addNoteVisibility(p.ViewerID)
	q.addViewerRelationFilters(p.ViewerID)
	if p.WithFiles {
		q.conds = append(q.conds, "EXISTS (SELECT 1 FROM drive_files df WHERE df.id = ANY(n.file_ids))")
	}
	q.addPageCursors(p.SinceID, p.UntilID)

	query := noteSelect + ` WHERE ` + strings.Join(q.conds, " AND ") +
		` AND n.deleted_at IS NULL ORDER BY n.id ` + paginationOrder(p.SinceID, p.UntilID) +
		` LIMIT ` + strconv.Itoa(p.Limit)

	return s.scanNotes(ctx, query, q.args...)
}

// scanNotes runs a note query and collects the rows.
func (s *TimelineService) scanNotes(ctx context.Context, query string, args ...any) ([]*Note, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("note: search query: %w", err)
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
