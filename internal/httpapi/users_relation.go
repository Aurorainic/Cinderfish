package httpapi

import (
	"encoding/json"
	"net/http"
)

// handleUsersRelation implements POST /api/users/relation: the follow, block
// and mute flags between the caller and one user or a batch of users.
//
// The response mirrors the request: a single id yields one object, an array
// yields an array of the same order.
func (s *Server) handleUsersRelation(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		UserID json.RawMessage `json:"userId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	var ids []string
	batch := false
	if len(body.UserID) > 0 {
		if err := json.Unmarshal(body.UserID, &ids); err != nil {
			var single string
			if err := json.Unmarshal(body.UserID, &single); err != nil {
				writeAPIError(w, invalidParamError("userId must be an id or an array of ids"))
				return
			}
			ids = []string{single}
		} else {
			batch = true
		}
	}
	if len(ids) == 0 {
		writeAPIError(w, invalidParamError("userId is required"))
		return
	}

	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}

	viewer := info.User
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		rel := map[string]any{
			"id":                             id,
			"isFollowing":                    false,
			"isFollowed":                     false,
			"hasPendingFollowRequestFromYou": false,
			"hasPendingFollowRequestToYou":   false,
			"isBlocking":                     false,
			"isBlocked":                      false,
			"isMuted":                        false,
			"isRenoteMuted":                  false,
		}
		if id != viewer.ID {
			if s.federationStore != nil {
				rel["isFollowing"] = s.federationStore.HasFollowing(r.Context(), viewer.ID, id)
				rel["isFollowed"] = s.federationStore.HasFollowing(r.Context(), id, viewer.ID)
				rel["hasPendingFollowRequestFromYou"] = s.federationStore.HasPendingFollowRequest(r.Context(), viewer.ID, id)
				rel["hasPendingFollowRequestToYou"] = s.federationStore.HasPendingFollowRequest(r.Context(), id, viewer.ID)
			}
			if muted, err := s.accounts.IsMuted(r.Context(), viewer.ID, id); err == nil {
				rel["isMuted"] = muted
			}
			if blocking, err := s.accounts.IsBlocked(r.Context(), viewer.ID, id); err == nil {
				rel["isBlocking"] = blocking
			}
			if blocked, err := s.accounts.IsBlocked(r.Context(), id, viewer.ID); err == nil {
				rel["isBlocked"] = blocked
			}
			// isRenoteMuted stays false: the renote-mute table does not exist.
		}
		out = append(out, rel)
	}

	if batch {
		writeJSON(w, http.StatusOK, out)
		return
	}
	if len(out) == 0 {
		writeAPIError(w, invalidParamError("no such user"))
		return
	}
	writeJSON(w, http.StatusOK, out[0])
}

// handleUsersFrequentlyReplied implements POST
// /api/users/get-frequently-replied-users: who a user replies to most, with the
// reply count as the weight.
func (s *Server) handleUsersFrequentlyReplied(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil || s.pool == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		UserID string `json:"userId"`
		Limit  int    `json:"limit"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.UserID == "" {
		writeAPIError(w, invalidParamError("userId is required"))
		return
	}
	limit := body.Limit
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	rows, err := s.pool.Query(r.Context(), `
SELECT n.reply_user_id, count(*) AS weight
FROM notes n
WHERE n.user_id = $1 AND n.reply_user_id IS NOT NULL AND n.deleted_at IS NULL
GROUP BY n.reply_user_id
ORDER BY weight DESC, n.reply_user_id ASC
LIMIT $2
`, body.UserID, limit)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	defer rows.Close()

	type replyStat struct {
		userID string
		weight int
	}
	var stats []replyStat
	for rows.Next() {
		var st replyStat
		if err := rows.Scan(&st.userID, &st.weight); err != nil {
			writeInternalServerError(w)
			return
		}
		stats = append(stats, st)
	}

	out := make([]any, 0, len(stats))
	for _, st := range stats {
		user, err := s.accounts.GetByID(r.Context(), st.userID)
		if err != nil || user == nil {
			continue
		}
		packed := packUserDetailed(user, user.CreatedAt, false, loadProfile(r.Context(), s, user.ID))
		s.packMovedTo(r.Context(), user, packed)
		out = append(out, map[string]any{"user": packed, "weight": st.weight})
	}
	writeJSON(w, http.StatusOK, out)
}
