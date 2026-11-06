package httpapi

import (
	"net/http"
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
