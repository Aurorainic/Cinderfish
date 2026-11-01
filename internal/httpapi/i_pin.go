package httpapi

import (
	"errors"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// errIDPinLimitExceeded is Cinderfish's id for the pin-limit error,
// styled after Misskey's error ids.
const errIDPinLimitExceeded = "4e1b0f8e-9a2d-4c7e-b3a1-6f5d8c2b7a94"

// handleIPin implements POST /api/i/pin: it adds a note owned by the
// authenticated user to their pinned notes, at most MaxPinnedNotes at a
// time. The response is the packed MeDetailed user, like i/update.
func (s *Server) handleIPin(w http.ResponseWriter, r *http.Request) {
	s.pinNote(w, r, true)
}

// handleIUnpin implements POST /api/i/unpin: it removes a note from the
// authenticated user's pinned notes. Unpinning a note that is not
// pinned is a no-op; unknown and foreign notes answer NO_SUCH_NOTE like
// Misskey, so a note's existence is not leaked.
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
		// Misskey maps both the unknown and the foreign note to
		// NO_SUCH_NOTE, so neither the existence nor the ownership of
		// a note is revealed.
		case errors.Is(err, account.ErrNoSuchPinTarget), errors.Is(err, account.ErrNotOwnNote):
			writeAPIError(w, &APIError{
				Message: "No such note.", Code: "NO_SUCH_NOTE",
				ID: errIDNoSuchNote, Kind: "client", Status: http.StatusNotFound,
			})
		case errors.Is(err, account.ErrPinLimitExceeded):
			writeAPIError(w, &APIError{
				Message: "You cannot pin any more notes.", Code: "PIN_LIMIT_EXCEEDED",
				ID: errIDPinLimitExceeded, Kind: "client", Status: http.StatusBadRequest,
			})
		default:
			writeInternalServerError(w)
		}
		return
	}

	packed := packMeDetailed(info.User, info.User.CreatedAt, info.Token == nil, loadProfile(r.Context(), s, info.User.ID))
	s.applyUserCounts(r.Context(), info.User.ID, packed)
	s.applyUnreadNotifications(r.Context(), info.User.ID, packed)
	// The pin state is the one field this response changes, so it must
	// carry the fresh list: the web session replaces itself with the
	// response, and an empty pinnedNoteIds would blank the footer's
	// pinned/unpinned label until the next /api/i.
	s.applyPinnedNotes(r.Context(), info.User.ID, info.User.ID, packed)
	writeJSON(w, http.StatusOK, packed)
}
