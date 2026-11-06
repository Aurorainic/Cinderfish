package note

import (
	"context"
	"fmt"
	"time"
)

// FeaturedWindow bounds how far back notes/featured looks. Upstream ranks the
// whole table through Redis; a personal instance has no such index, and a
// window keeps the query a scan of recent rows instead of everything ever
// written.
const FeaturedWindow = 30 * 24 * time.Hour

// Featured returns public local notes ranked by engagement, newest first on
// ties. Renotes are skipped: they carry no text of their own, and featuring
// one would credit the wrong author.
//
// The score is renotes*2 + replies + reactions, the same three signals
// upstream weights, so a note people quote or answer outranks one that was
// only liked.
func (s *Service) Featured(ctx context.Context, limit int, untilID, sinceID string) ([]*Note, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	since := time.Now().Add(-FeaturedWindow)

	rows, err := s.pool.Query(ctx, noteSelect+`
 WHERE n.deleted_at IS NULL
   AND n.user_host IS NULL
   AND n.visibility = 'public'
   AND n.renote_id IS NULL
   AND n.created_at >= $1
   AND ($2 = '' OR n.id < $2)
   AND ($3 = '' OR n.id > $3)
 ORDER BY (n.renote_count * 2 + n.replies_count
           + (SELECT count(*) FROM note_reactions r WHERE r.note_id = n.id)) DESC,
          n.id DESC
 LIMIT $4`, since, untilID, sinceID, limit)
	if err != nil {
		return nil, fmt.Errorf("note: featured: %w", err)
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
