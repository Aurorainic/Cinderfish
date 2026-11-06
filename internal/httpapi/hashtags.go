package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

// handleHashtagsTrend implements POST /api/hashtags/trend.
//
// Upstream ranks tags from Redis-backed daily statistics; Cinderfish derives
// the same shape from the notes rows (distinct local users per tag over the
// last seven days), so the endpoint is readable without a credential like
// upstream.
func (s *Server) handleHashtagsTrend(w http.ResponseWriter, r *http.Request) {
	if s.hashtags == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	trends, err := s.hashtags.Trend(r.Context(), 10, s.liveInstanceSettings(r.Context()).HiddenTags)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]any, 0, len(trends))
	for _, t := range trends {
		out = append(out, map[string]any{
			"tag":        t.Tag,
			"chart":      t.Chart,
			"usersCount": t.UsersCount,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleHashtagsSearch implements POST /api/hashtags/search.
//
// `query` may be empty, which asks for the most used tags overall — the shape
// the composer's suggestion list wants. `sort` is accepted and ignored:
// Cinderfish only ranks by usage, and silently honouring half of upstream's
// sort modes would be worse than ranking the same way every time.
func (s *Server) handleHashtagsSearch(w http.ResponseWriter, r *http.Request) {
	if s.hashtags == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}
	var body struct {
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
		Sort   string `json:"sort"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	tags, err := s.hashtags.SearchHashtags(r.Context(), body.Query, body.Limit, body.Offset)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

// errIDNoSuchHashtag is Misskey's NO_SUCH_HASHTAG id.
const errIDNoSuchHashtag = "110ee688-193e-4a3a-9ecf-c167b2e6981e"

// hashtagCounts is the packed Hashtag entity. Misskey keeps these counters in a
// hashtags table that it updates as notes arrive; Cinderfish derives them from
// the notes it stores, so the numbers are exact rather than cached.
type hashtagCounts struct {
	Tag                       string
	MentionedUsersCount       int
	MentionedLocalUsersCount  int
	MentionedRemoteUsersCount int
}

func (h hashtagCounts) pack() map[string]any {
	return map[string]any{
		"tag":                       h.Tag,
		"mentionedUsersCount":       h.MentionedUsersCount,
		"mentionedLocalUsersCount":  h.MentionedLocalUsersCount,
		"mentionedRemoteUsersCount": h.MentionedRemoteUsersCount,
		// The attached* counters come from users listing a tag on their
		// profile, a feature Cinderfish does not model.
		"attachedUsersCount":       0,
		"attachedLocalUsersCount":  0,
		"attachedRemoteUsersCount": 0,
	}
}

// handleHashtagsShow implements POST /api/hashtags/show.
func (s *Server) handleHashtagsShow(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		Tag string `json:"tag"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	tag := normalizeHashtag(body.Tag)
	if tag == "" {
		writeAPIError(w, invalidParamError("tag is required"))
		return
	}

	counts, found, err := s.hashtagCounts(r.Context(), tag)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if !found {
		writeAPIError(w, &APIError{
			Message: "No such hashtag.", Code: "NO_SUCH_HASHTAG",
			ID: errIDNoSuchHashtag, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	writeJSON(w, http.StatusOK, counts.pack())
}

// normalizeHashtag matches Misskey's normalizeTag: strip the marker, trim and
// lowercase.
func normalizeHashtag(raw string) string {
	tag := strings.TrimSpace(raw)
	tag = strings.TrimPrefix(tag, "#")
	tag = strings.TrimPrefix(tag, "＃")
	return strings.ToLower(tag)
}

// hashtagCounts aggregates one tag over the notes that carry it.
func (s *Server) hashtagCounts(ctx context.Context, tag string) (hashtagCounts, bool, error) {
	var c hashtagCounts
	err := s.pool.QueryRow(ctx, `
SELECT count(DISTINCT n.user_id),
       count(DISTINCT n.user_id) FILTER (WHERE n.user_host IS NULL),
       count(DISTINCT n.user_id) FILTER (WHERE n.user_host IS NOT NULL)
FROM notes n, jsonb_array_elements_text(n.tags) AS tag
WHERE n.deleted_at IS NULL AND lower(tag) = $1
`, tag).Scan(&c.MentionedUsersCount, &c.MentionedLocalUsersCount, &c.MentionedRemoteUsersCount)
	if err != nil {
		return c, false, err
	}
	c.Tag = tag
	return c, c.MentionedUsersCount > 0, nil
}

// handleHashtagsList implements POST /api/hashtags/list.
//
// The sort enum covers the mentioned* and attached* counters. Cinderfish has no
// attached counter, so the attached* orders fall back to the mentioned count,
// which keeps the page stable instead of returning an arbitrary order.
func (s *Server) handleHashtagsList(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		Limit                    int    `json:"limit"`
		AttachedToUserOnly       bool   `json:"attachedToUserOnly"`
		AttachedToLocalUserOnly  bool   `json:"attachedToLocalUserOnly"`
		AttachedToRemoteUserOnly bool   `json:"attachedToRemoteUserOnly"`
		Sort                     string `json:"sort"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	limit := body.Limit
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	// An attached-only filter has an empty answer by definition here.
	if body.AttachedToUserOnly || body.AttachedToLocalUserOnly || body.AttachedToRemoteUserOnly {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	desc := true
	if strings.HasPrefix(body.Sort, "-") {
		desc = false
	}
	order := "DESC"
	if !desc {
		order = "ASC"
	}
	// The sort enum names four counters; attached* has no data, so those
	// modes fall back to the matching mentioned* count.
	column := "users"
	switch strings.TrimLeft(body.Sort, "+-") {
	case "mentionedLocalUsers", "attachedLocalUsers":
		column = "local_users"
	case "mentionedRemoteUsers", "attachedRemoteUsers":
		column = "remote_users"
	case "mentionedUsers", "attachedUsers":
		column = "users"
	}

	rows, err := s.pool.Query(r.Context(), `
SELECT lower(tag) AS tag,
       count(DISTINCT n.user_id) AS users,
       count(DISTINCT n.user_id) FILTER (WHERE n.user_host IS NULL) AS local_users,
       count(DISTINCT n.user_id) FILTER (WHERE n.user_host IS NOT NULL) AS remote_users
FROM notes n, jsonb_array_elements_text(n.tags) AS tag
WHERE n.deleted_at IS NULL
GROUP BY lower(tag)
ORDER BY `+column+` `+order+`, tag ASC
LIMIT `+strconv.Itoa(limit))
	if err != nil {
		writeInternalServerError(w)
		return
	}
	defer rows.Close()

	out := make([]any, 0, limit)
	for rows.Next() {
		var c hashtagCounts
		if err := rows.Scan(&c.Tag, &c.MentionedUsersCount, &c.MentionedLocalUsersCount, &c.MentionedRemoteUsersCount); err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, c.pack())
	}
	writeJSON(w, http.StatusOK, out)
}

// handleHashtagsUsers implements POST /api/hashtags/users.
//
// Misskey lists the users who have the tag on their profile, not the users who
// posted with it. Cinderfish does not model profile tags, so the honest answer
// is an empty list; the endpoint exists so clients do not see a 404.
func (s *Server) handleHashtagsUsers(w http.ResponseWriter, r *http.Request) {
	if err := decodeOptionalJSON(r, &struct{}{}); err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, []any{})
}
