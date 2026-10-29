package httpapi

import (
	"errors"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// handleIPin implements POST /api/i/pin: it adds a note owned by the
// authenticated user to their pinned notes. The answer is 204, matching
// Misskey's empty response.
func (s *Server) handleIPin(w http.ResponseWriter, r *http.Request) {
	s.pinNote(w, r, true)
}

// handleIUnpin implements POST /api/i/unpin: it removes a note from the
// authenticated user's pinned notes. Unpinning a note that is not pinned
// is a no-op.
func (s *Server) handleIUnpin(w http.ResponseWriter, r *http.Request) {
	s.pinNote(w, r, false)
}

func (s *Server) pinNote(w http.ResponseWriter, r *http.Request, pin bool) {
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
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	if pin {
		err = s.accounts.PinNote(r.Context(), info.User.ID, body.NoteID)
	} else {
		err = s.accounts.UnpinNote(r.Context(), info.User.ID, body.NoteID)
	}
	if err != nil {
		switch {
		case errors.Is(err, account.ErrNoSuchPinTarget):
			writeAPIError(w, &APIError{
				Message: "No such note.", Code: "NO_SUCH_NOTE",
				ID: errIDNoSuchNote, Kind: "client", Status: http.StatusNotFound,
			})
		case errors.Is(err, account.ErrNotOwnNote):
			writeAPIError(w, &APIError{
				Message: "That note is not yours.", Code: "ACCESS_DENIED",
				ID: errIDForbidden, Kind: "permission", Status: http.StatusForbidden,
			})
		default:
			writeInternalServerError(w)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
