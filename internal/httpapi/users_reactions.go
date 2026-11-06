package httpapi

import (
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/note"
)

// Error ids for users/reactions, copied from Misskey.
const (
	errIDReactionsNotPublic = "673a7dd2-6924-1093-e0c0-e68456ceae5c"
	errIDIsRemoteUser       = "6b95fa98-8cf9-2350-e284-f0ffdb54a805"
)

// handleUsersReactions implements POST /api/users/reactions: the reactions one
// account made, packed with the note each belongs to (the profile's reaction
// tab).
func (s *Server) handleUsersReactions(w http.ResponseWriter, r *http.Request) {
	if s.noteReactions == nil || s.notes == nil || s.accounts == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		UserID    string `json:"userId"`
		Limit     int    `json:"limit"`
		SinceID   string `json:"sinceId"`
		UntilID   string `json:"untilId"`
		SinceDate *int64 `json:"sinceDate"`
		UntilDate *int64 `json:"untilDate"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.UserID == "" {
		writeAPIError(w, invalidParamError("userId is required"))
		return
	}

	info, _ := s.authenticate(r.Context(), r, bodyAsMap(r))
	viewerID := ""
	if info != nil {
		viewerID = info.User.ID
	}

	target, err := s.accounts.GetByID(r.Context(), body.UserID)
	if err != nil || target == nil {
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDNoSuchUser, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	// Misskey only serves this for local accounts (remote reactions are not
	// stored in a queryable form on the home instance).
	if target.Host != nil {
		writeAPIError(w, &APIError{
			Message: "Currently unavailable to display reactions of remote users.",
			Code:    "IS_REMOTE_USER",
			ID:      errIDIsRemoteUser, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	prof := loadProfile(r.Context(), s, target.ID)
	if viewerID != target.ID && !prof.PublicReactions {
		writeAPIError(w, &APIError{
			Message: "Reactions of the user is not public.", Code: "REACTIONS_NOT_PUBLIC",
			ID: errIDReactionsNotPublic, Kind: "client", Status: http.StatusForbidden,
		})
		return
	}

	q := note.ReactionQuery{
		Limit:   body.Limit,
		SinceID: body.SinceID,
		UntilID: body.UntilID,
	}
	if body.SinceDate != nil {
		t := time.UnixMilli(*body.SinceDate)
		q.SinceDate = &t
	}
	if body.UntilDate != nil {
		t := time.UnixMilli(*body.UntilDate)
		q.UntilDate = &t
	}

	list, err := s.noteReactions.ListByUser(r.Context(), target.ID, q)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]any, 0, len(list))
	for _, reaction := range list {
		n, err := s.notes.Get(r.Context(), reaction.NoteID)
		if err != nil || n == nil {
			continue
		}
		visible, err := s.notes.VisibleTo(r.Context(), n, viewerID)
		if err != nil || !visible {
			continue
		}
		out = append(out, map[string]any{
			"id":        reaction.ID,
			"createdAt": reaction.CreatedAt.UTC().Format(time.RFC3339Nano),
			"type":      reaction.Emoji,
			"user":      packUserLite(target, "", prof),
			"note":      packNote(n, s.notePackContext(r.Context(), n, viewerID)),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
