package httpapi

import (
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/note"
)

// errIDNoSuchNoteReactions is Misskey's NO_SUCH_NOTE id for notes/reactions
// (packages/backend/src/server/api/endpoints/notes/reactions.ts).
const errIDNoSuchNoteReactions = "263fff3d-d0e1-4af4-bea7-8408059b451a"

// handleNotesReactions implements POST /api/notes/reactions: who reacted to a
// note, with the Misskey NoteReaction shape {id, createdAt, type, user}.
//
// The endpoint is readable without a credential, like upstream, but a note
// the viewer may not read answers NO_SUCH_NOTE instead of exposing its
// reaction list.
func (s *Server) handleNotesReactions(w http.ResponseWriter, r *http.Request) {
	if s.noteReactions == nil || s.notes == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		NoteID    string  `json:"noteId"`
		Type      *string `json:"type"`
		Limit     int     `json:"limit"`
		SinceID   string  `json:"sinceId"`
		UntilID   string  `json:"untilId"`
		SinceDate *int64  `json:"sinceDate"`
		UntilDate *int64  `json:"untilDate"`
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

	n, err := s.notes.Get(r.Context(), body.NoteID)
	if err != nil {
		writeNoSuchNote(w)
		return
	}
	visible, err := s.notes.VisibleTo(r.Context(), n, viewerID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if !visible {
		writeNoSuchNote(w)
		return
	}

	q := note.ReactionQuery{
		Limit:   body.Limit,
		SinceID: body.SinceID,
		UntilID: body.UntilID,
	}
	if body.Type != nil {
		q.Type = *body.Type
	}
	if body.SinceDate != nil {
		t := time.UnixMilli(*body.SinceDate)
		q.SinceDate = &t
	}
	if body.UntilDate != nil {
		t := time.UnixMilli(*body.UntilDate)
		q.UntilDate = &t
	}

	list, err := s.noteReactions.List(r.Context(), body.NoteID, q)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]any, 0, len(list))
	for _, reaction := range list {
		user, err := s.accounts.GetByID(r.Context(), reaction.UserID)
		if err != nil || user == nil {
			// A reaction by a user that no longer exists has no UserLite to
			// carry; skip it rather than emitting a null user.
			continue
		}
		out = append(out, map[string]any{
			"id":        reaction.ID,
			"createdAt": reaction.CreatedAt.UTC().Format(time.RFC3339Nano),
			"type":      reaction.Emoji,
			"user":      packUserLite(user, "", loadProfile(r.Context(), s, user.ID)),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// writeNoSuchNote answers Misskey's NO_SUCH_NOTE for the endpoints that use
// the notes/reactions error id.
func writeNoSuchNote(w http.ResponseWriter) {
	writeAPIError(w, &APIError{
		Message: "No such note.", Code: "NO_SUCH_NOTE",
		ID: errIDNoSuchNoteReactions, Kind: "client", Status: http.StatusNotFound,
	})
}
