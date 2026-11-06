package httpapi

import "net/http"

// handleGetOnlineUsersCount implements POST /api/get-online-users-count.
//
// Misskey counts users whose lastActiveDate is inside USER_ONLINE_THRESHOLD
// (ten minutes) and allows the call without a credential. Cinderfish tracks the
// same window from authenticated requests in memory, so the count is exact for
// this process but starts from zero after a restart; a deployment that serves
// Streaming from a second process reports only the API process's activity.
func (s *Server) handleGetOnlineUsersCount(w http.ResponseWriter, r *http.Request) {
	count := 0
	if s.activity != nil {
		count = s.activity.count(onlineUserThreshold)
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": count})
}
