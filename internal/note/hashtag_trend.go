package note

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TrendBuckets is the number of daily chart points returned per hashtag.
// Misskey keeps daily hashtag statistics in Redis; Cinderfish derives the
// same numbers from the notes it already stores.
const TrendBuckets = 7

// HashtagTrend is one row of hashtags/trend.
type HashtagTrend struct {
	Tag        string
	Chart      []int
	UsersCount int
}

// HashtagService answers hashtag queries.
type HashtagService struct {
	pool *pgxpool.Pool
}

// NewHashtagService returns a HashtagService backed by the given pool.
func NewHashtagService(pool *pgxpool.Pool) *HashtagService {
	return &HashtagService{pool: pool}
}

// Trend ranks hashtags by the number of distinct local users who used them in
// the window, newest activity first on ties.
//
// The chart is one point per UTC day, oldest first, holding that day's
// distinct users; usersCount mirrors Misskey's max(chart). Notes that are
// deleted or carried only by remote actors are left out: a trend of what this
// instance's users are talking about should not be steered by inbound
// federation volume.
func (s *HashtagService) Trend(ctx context.Context, limit int) ([]*HashtagTrend, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// Buckets are whole UTC days ending today, so the window start is the
	// first of them; using "now - 7d" would leave the chart one day short.
	since := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -(TrendBuckets - 1))

	rows, err := s.pool.Query(ctx, `
WITH exploded AS (
    SELECT lower(tag) AS tag, n.user_id,
           to_char(date_trunc('day', n.created_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS day
    FROM notes n, jsonb_array_elements_text(n.tags) AS tag
    WHERE n.deleted_at IS NULL
      AND n.user_host IS NULL
      AND n.created_at >= $1
), ranked AS (
    SELECT tag, count(DISTINCT user_id) AS users
    FROM exploded GROUP BY tag ORDER BY users DESC, tag ASC LIMIT $2
)
SELECT r.tag, e.day, count(DISTINCT e.user_id)
FROM ranked r JOIN exploded e USING (tag)
GROUP BY r.tag, e.day
ORDER BY r.tag, e.day
	`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("note: hashtag trend: %w", err)
	}
	defer rows.Close()

	order := []string{}
	charts := map[string]map[string]int{}
	for rows.Next() {
		var tag string
		var day string
		var users int
		if err := rows.Scan(&tag, &day, &users); err != nil {
			return nil, fmt.Errorf("note: scan hashtag trend: %w", err)
		}
		if _, ok := charts[tag]; !ok {
			charts[tag] = map[string]int{}
			order = append(order, tag)
		}
		charts[tag][day] = users
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]*HashtagTrend, 0, len(order))
	for _, tag := range order {
		chart := make([]int, 0, TrendBuckets)
		max := 0
		for i := TrendBuckets - 1; i >= 0; i-- {
			key := since.Add(time.Duration(TrendBuckets-1-i) * 24 * time.Hour).Format("2006-01-02")
			v := charts[tag][key]
			if v > max {
				max = v
			}
			chart = append(chart, v)
		}
		out = append(out, &HashtagTrend{Tag: tag, Chart: chart, UsersCount: max})
	}

	// The SQL ranks by users already, but the per-day rows come back in tag
	// order; restore the ranking (ties by tag) before answering.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UsersCount != out[j].UsersCount {
			return out[i].UsersCount > out[j].UsersCount
		}
		return out[i].Tag < out[j].Tag
	})
	return out, nil
}
