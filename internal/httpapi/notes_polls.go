package httpapi

import (
	"errors"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/note"
)

// Misskey error ids for notes/polls/vote
// (packages/backend/src/server/api/endpoints/notes/polls/vote.ts).
const (
	errIDPollNoSuchNote = "ecafbd2e-c283-4d6d-aecb-1a0a33b75396"
	errIDNoPoll         = "5f979967-52d9-4314-a911-1c673727f92f"
	errIDAlreadyVoted   = "0963fc77-efac-419b-9424-b391608dc6d8"
	errIDAlreadyExpired = "1022a357-b085-4054-9083-8f8de358337e"
)

// handleNotesPollsVote implements POST /api/notes/polls/vote.
//
// A single-choice poll replaces nothing: a second vote is refused with
// ALREADY_VOTED. A multiple-choice poll accepts one vote per distinct choice.
func (s *Server) handleNotesPollsVote(w http.ResponseWriter, r *http.Request) {
	if s.notes == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		NoteID string `json:"noteId"`
		Choice *int   `json:"choice"`
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
	if body.NoteID == "" || body.Choice == nil {
		writeAPIError(w, invalidParamError("noteId and choice are required"))
		return
	}

	// The note must exist and be readable; an unreadable note is answered
	// like a missing one, as elsewhere in this API.
	n, err := s.notes.Get(r.Context(), body.NoteID)
	if err != nil {
		writePollNoSuchNote(w)
		return
	}
	visible, err := s.notes.VisibleTo(r.Context(), n, info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if !visible {
		writePollNoSuchNote(w)
		return
	}

	if err := s.notes.Vote(r.Context(), body.NoteID, info.User.ID, *body.Choice); err != nil {
		switch {
		case errors.Is(err, note.ErrNoPoll):
			writeAPIError(w, &APIError{
				Message: "The note does not attach a poll.", Code: "NO_POLL",
				ID: errIDNoPoll, Kind: "client", Status: http.StatusBadRequest,
			})
		case errors.Is(err, note.ErrAlreadyVoted):
			writeAPIError(w, &APIError{
				Message: "You have already voted.", Code: "ALREADY_VOTED",
				ID: errIDAlreadyVoted, Kind: "client", Status: http.StatusBadRequest,
			})
		case errors.Is(err, note.ErrPollExpired):
			writeAPIError(w, &APIError{
				Message: "The poll is already expired.", Code: "ALREADY_EXPIRED",
				ID: errIDAlreadyExpired, Kind: "client", Status: http.StatusBadRequest,
			})
		case errors.Is(err, note.ErrChoiceOutOfRange):
			writeAPIError(w, invalidParamError("choice is out of range"))
		default:
			writeInternalServerError(w)
		}
		return
	}

	// Everyone with the note open sees the new counts; the voter's own client
	// already applied them optimistically, so a duplicate is harmless.
	s.publishNotePollVoted(n, info.User.ID, *body.Choice)

	writeJSON(w, http.StatusOK, struct{}{})
}

func writePollNoSuchNote(w http.ResponseWriter) {
	writeAPIError(w, &APIError{
		Message: "No such note.", Code: "NO_SUCH_NOTE",
		ID: errIDPollNoSuchNote, Kind: "client", Status: http.StatusNotFound,
	})
}
