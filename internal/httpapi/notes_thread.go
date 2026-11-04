package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/note"
)

// handleNotesConversation implements POST /api/notes/conversation: the reply
// chain above a note, nearest ancestor first.
func (s *Server) handleNotesConversation(w http.ResponseWriter, r *http.Request) {
	if s.timeline == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		NoteID string `json:"noteId"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.NoteID == "" {
		writeAPIError(w, invalidParamError("noteId is required"))
		return
	}

	info, _ := s.authenticate(r.Context(), r, bodyAsMap(r))
	viewerID := ""
	if info != nil {
		viewerID = info.User.ID
	}

	list, err := s.timeline.Conversation(r.Context(), body.NoteID, viewerID, body.Limit, body.Offset)
	if err != nil {
		if errors.Is(err, note.ErrNotFound) {
			writeAPIError(w, &APIError{
				Message: "No such note.", Code: "NO_SUCH_NOTE",
				ID: errIDNoSuchNote, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, s.packNoteList(r, list, viewerID))
}

// handleNotesChildren implements POST /api/notes/children: the replies to a
// note, newest first.
func (s *Server) handleNotesChildren(w http.ResponseWriter, r *http.Request) {
	if s.timeline == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		NoteID    string `json:"noteId"`
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
	if body.NoteID == "" {
		writeAPIError(w, invalidParamError("noteId is required"))
		return
	}

	info, _ := s.authenticate(r.Context(), r, bodyAsMap(r))
	viewerID := ""
	if info != nil {
		viewerID = info.User.ID
	}

	q := note.ChildrenQuery{
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

	list, err := s.timeline.Children(r.Context(), body.NoteID, viewerID, q)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, s.packNoteList(r, list, viewerID))
}

// packNoteList packs a note slice page in one go.
func (s *Server) packNoteList(r *http.Request, list []*note.Note, viewerID string) []any {
	out := make([]any, 0, len(list))
	for _, n := range list {
		pc := s.notePackContext(r.Context(), n, viewerID)
		out = append(out, packNote(n, pc))
	}
	return out
}
