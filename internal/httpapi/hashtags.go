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

	trends, err := s.hashtags.Trend(r.Context(), 10)
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
