package httpapi

import (
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// handleUsers is POST /api/users: the public user directory. It is readable by
// guests; a signed-in viewer additionally gets the follow flags for every row.
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Limit    int     `json:"limit"`
		Offset   int     `json:"offset"`
		Sort     string  `json:"sort"`
		State    string  `json:"state"`
		Origin   string  `json:"origin"`
		Hostname *string `json:"hostname"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}

	users, err := s.accounts.ListUsers(r.Context(), account.UsersParams{
		Limit:    body.Limit,
		Offset:   body.Offset,
		Sort:     body.Sort,
		State:    body.State,
		Origin:   body.Origin,
		Hostname: body.Hostname,
	})
	if err != nil {
		writeAPIError(w, invalidParamError(err.Error()))
		return
	}

	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		packed := packUserDetailed(u, u.CreatedAt, false, loadProfile(r.Context(), s, u.ID))
		s.applyUserCounts(r.Context(), u.ID, packed)
		if viewerID != "" && viewerID != u.ID && s.federationStore != nil {
			packed["isFollowing"] = s.federationStore.HasFollowing(r.Context(), viewerID, u.ID)
			packed["isFollowed"] = s.federationStore.HasFollowing(r.Context(), u.ID, viewerID)
		}
		out = append(out, packed)
	}
	writeJSON(w, http.StatusOK, out)
}
