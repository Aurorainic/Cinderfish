package note

import (
	"context"
	"fmt"
)

// SearchHashtags returns hashtags that contain query, most used first.
//
// Upstream reads its Redis-backed tag index; Cinderfish derives the answer
// from the notes it stores, so only tags on visible, undeleted notes are
// found. An empty query returns the most used tags overall, which is what the
// composer's suggestion box asks for.
func (s *HashtagService) SearchHashtags(ctx context.Context, query string, limit, offset int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	pattern := "%" + escapeLike(query) + "%"
	rows, err := s.pool.Query(ctx, `
SELECT lower(tag) AS tag, count(*) AS uses
FROM notes n, jsonb_array_elements_text(n.tags) AS tag
WHERE n.deleted_at IS NULL
  AND n.visibility IN ('public', 'home')
  AND lower(tag) LIKE lower($1)
GROUP BY 1
ORDER BY uses DESC, tag ASC
LIMIT $2 OFFSET $3
`, pattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("note: search hashtags: %w", err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var tag string
		var uses int
		if err := rows.Scan(&tag, &uses); err != nil {
			return nil, fmt.Errorf("note: scan hashtag: %w", err)
		}
		out = append(out, tag)
	}
	return out, rows.Err()
}
