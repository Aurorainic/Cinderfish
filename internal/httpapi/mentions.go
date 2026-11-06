package httpapi

import (
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/note"
)

// handleNotesMentions is POST /api/notes/mentions: notes that mention the
// caller, with Misskey's optional "following only" and visibility filters.
func (s *Server) handleNotesMentions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Following  bool   `json:"following"`
		Limit      int    `json:"limit"`
		SinceID    string `json:"sinceId"`
		UntilID    string `json:"untilId"`
		Visibility string `json:"visibility"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	if s.timeline == nil {
		writeInternalServerError(w)
		return
	}

	notes, err := s.timeline.Mentions(r.Context(), note.MentionsParams{
		ViewerID:   info.User.ID,
		Following:  body.Following,
		Limit:      body.Limit,
		SinceID:    body.SinceID,
		UntilID:    body.UntilID,
		Visibility: body.Visibility,
	})
	if err != nil {
		writeInternalServerError(w)
		return
	}
	s.writeNoteList(w, r, notes, info.User.ID)
}
