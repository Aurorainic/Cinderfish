package httpapi

import (
	"net/http"
	"time"
)

// chartsUserNotes is Misskey's per-user notes chart: parallel arrays, oldest
// first, one entry per span bucket.
//
// Misskey fills these from a background aggregation table. Cinderfish derives
// the same shape from the notes rows, so the numbers are exact at read time;
// `total` is the cumulative count (created minus deleted) and `inc`/`dec` are
// the created and deleted notes inside each bucket.
type chartsUserNotes struct {
	Total []int `json:"total"`
	Inc   []int `json:"inc"`
	Dec   []int `json:"dec"`
	Diffs struct {
		Normal   []int `json:"normal"`
		Reply    []int `json:"reply"`
		Renote   []int `json:"renote"`
		WithFile []int `json:"withFile"`
	} `json:"diffs"`
}

// handleChartsUserNotes implements POST /api/charts/user/notes.
func (s *Server) handleChartsUserNotes(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		Span   string `json:"span"`
		Limit  int    `json:"limit"`
		Offset *int64 `json:"offset"`
		UserID string `json:"userId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.Span != "day" && body.Span != "hour" {
		writeAPIError(w, invalidParamError("span must be day or hour"))
		return
	}
	if body.UserID == "" {
		writeAPIError(w, invalidParamError("userId is required"))
		return
	}
	limit := body.Limit
	if limit <= 0 {
		limit = 30
	}
	if limit > 500 {
		limit = 500
	}

	unit := "day"
	if body.Span == "hour" {
		unit = "hour"
	}
	end := time.Now().UTC()
	if body.Offset != nil {
		end = time.UnixMilli(*body.Offset).UTC()
	}
	// Buckets are UTC-aligned and the last one is the bucket containing end.
	last := truncateBucket(end, unit)
	first := last.Add(-time.Duration(limit-1) * bucketDuration(unit))
	until := last.Add(bucketDuration(unit))

	bucketKey := func(t time.Time) string {
		return truncateBucket(t, unit).Format("2006-01-02T15")
	}
	created := map[string][4]int{}
	rows, err := s.pool.Query(r.Context(), `
SELECT date_trunc('`+unit+`', created_at AT TIME ZONE 'UTC') AS bucket,
       count(*) FILTER (WHERE renote_id IS NULL AND reply_id IS NULL),
       count(*) FILTER (WHERE reply_id IS NOT NULL),
       count(*) FILTER (WHERE renote_id IS NOT NULL),
       count(*) FILTER (WHERE cardinality(file_ids) > 0)
FROM notes
WHERE user_id = $1 AND created_at >= $2 AND created_at < $3
GROUP BY bucket
`, body.UserID, first, until)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	for rows.Next() {
		var bucket time.Time
		var normal, reply, renote, withFile int
		if err := rows.Scan(&bucket, &normal, &reply, &renote, &withFile); err != nil {
			rows.Close()
			writeInternalServerError(w)
			return
		}
		created[bucketKey(bucket)] = [4]int{normal, reply, renote, withFile}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		writeInternalServerError(w)
		return
	}
	rows.Close()

	deleted := map[string]int{}
	rows, err = s.pool.Query(r.Context(), `
SELECT date_trunc('`+unit+`', deleted_at AT TIME ZONE 'UTC') AS bucket, count(*)
FROM notes
WHERE user_id = $1 AND deleted_at >= $2 AND deleted_at < $3
GROUP BY bucket
`, body.UserID, first, until)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	for rows.Next() {
		var bucket time.Time
		var n int
		if err := rows.Scan(&bucket, &n); err != nil {
			rows.Close()
			writeInternalServerError(w)
			return
		}
		deleted[bucketKey(bucket)] = n
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		writeInternalServerError(w)
		return
	}
	rows.Close()

	// The cumulative total starts from everything before the window so the
	// first bucket is not mistaken for the account's whole history.
	var base int
	if err := s.pool.QueryRow(r.Context(), `
SELECT count(*) FILTER (WHERE deleted_at IS NULL) - count(*) FILTER (WHERE deleted_at IS NOT NULL)
FROM notes WHERE user_id = $1 AND created_at < $2
`, body.UserID, first).Scan(&base); err != nil {
		writeInternalServerError(w)
		return
	}

	out := chartsUserNotes{
		Total: make([]int, 0, limit),
		Inc:   make([]int, 0, limit),
		Dec:   make([]int, 0, limit),
	}
	out.Diffs.Normal = make([]int, 0, limit)
	out.Diffs.Reply = make([]int, 0, limit)
	out.Diffs.Renote = make([]int, 0, limit)
	out.Diffs.WithFile = make([]int, 0, limit)

	total := base
	for i := 0; i < limit; i++ {
		bucket := first.Add(time.Duration(i) * bucketDuration(unit))
		key := bucketKey(bucket)
		c := created[key]
		inc := c[0] + c[1] + c[2]
		dec := deleted[key]
		total += inc - dec
		out.Total = append(out.Total, total)
		out.Inc = append(out.Inc, inc)
		out.Dec = append(out.Dec, dec)
		out.Diffs.Normal = append(out.Diffs.Normal, c[0])
		out.Diffs.Reply = append(out.Diffs.Reply, c[1])
		out.Diffs.Renote = append(out.Diffs.Renote, c[2])
		out.Diffs.WithFile = append(out.Diffs.WithFile, c[3])
	}
	writeJSON(w, http.StatusOK, out)
}

func bucketDuration(unit string) time.Duration {
	if unit == "hour" {
		return time.Hour
	}
	return 24 * time.Hour
}

func truncateBucket(t time.Time, unit string) time.Time {
	t = t.UTC()
	if unit == "hour" {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
