package httpapi

import (
	"errors"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/note"
)

// handleNotesState implements POST /api/notes/state: whether the caller has
// favorited or pinned the note.
func (s *Server) handleNotesState(w http.ResponseWriter, r *http.Request) {
	if s.notes == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}
	var body struct {
		NoteID string `json:"noteId"`
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
	if body.NoteID == "" {
		writeAPIError(w, invalidParamError("noteId is required"))
		return
	}

	state, err := s.notes.State(r.Context(), info.User.ID, body.NoteID)
	if errors.Is(err, note.ErrNotFound) {
		writeAPIError(w, &APIError{
			Message: "No such note.", Code: "NO_SUCH_NOTE",
			ID: errIDNoSuchNote, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"isFavorited":   state.IsFavorited,
		"isPinned":      state.IsPinned,
		"isMutedThread": state.IsMutedThread,
	})
}
