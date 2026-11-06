package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/note"
)

// Error ids from Firefish's notes/edit.
const (
	errIDNeedsEditID    = "d697edc8-8c73-4de8-bded-35fd198b79e5"
	errIDYouNotAuthor   = "c6e61685-411d-43d0-b90a-a448d2539001"
	errIDNoteEditNoNote = "eef6c173-3010-4a23-8674-7c4fcaeba719"
)

// handleNotesEdit implements POST /api/notes/edit. Firefish's endpoint takes
// the whole note payload; this port edits text, CW, visibility and localOnly.
// Attachments and polls keep their current values.
func (s *Server) handleNotesEdit(w http.ResponseWriter, r *http.Request) {
	if s.notes == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		EditID     string        `json:"editId"`
		Text       nullableField `json:"text"`
		CW         nullableField `json:"cw"`
		Visibility string        `json:"visibility"`
		LocalOnly  *bool         `json:"localOnly"`
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
	if body.EditID == "" {
		writeAPIError(w, &APIError{
			Message: "You need to specify `editId`.", Code: "NEEDS_EDIT_ID",
			ID: errIDNeedsEditID, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	updated, err := s.notes.Edit(r.Context(), info.User.ID, body.EditID, note.EditParams{
		Text:       body.Text.asUpdatePointer(),
		CW:         body.CW.asUpdatePointer(),
		Visibility: body.Visibility,
		LocalOnly:  body.LocalOnly,
	})
	switch {
	case errors.Is(err, note.ErrNotFound):
		writeAPIError(w, &APIError{
			Message: "No such note.", Code: "NO_SUCH_NOTE",
			ID: errIDNoteEditNoNote, Kind: "client", Status: http.StatusNotFound,
		})
		return
	case errors.Is(err, note.ErrNotNoteAuthor):
		writeAPIError(w, &APIError{
			Message: "You are not the author of this note.", Code: "YOU_ARE_NOT_THE_AUTHOR",
			ID: errIDYouNotAuthor, Kind: "client", Status: http.StatusForbidden,
		})
		return
	case errors.Is(err, note.ErrTextTooLong):
		writeAPIError(w, invalidParamError("text is too long"))
		return
	case errors.Is(err, note.ErrInvalidVisibility):
		writeAPIError(w, invalidParamError("visibility is invalid"))
		return
	case err != nil:
		writeInternalServerError(w)
		return
	}

	// Tell the remote audience the note changed; a peer that ignores status
	// updates keeps the old copy, which is why the marker is also local.
	s.DeliverUpdate(r.Context(), updated)

	writeJSON(w, http.StatusOK, map[string]any{
		"createdNote": packNote(updated, s.notePackContext(r.Context(), updated, info.User.ID)),
	})
}

// handleNotesHistory implements POST /api/notes/history.
//
// Misskey at the pinned commit has no history endpoint; this is Cinderfish's
// own route (see the coverage register). It is restricted to the note's
// author because revisions can contain text the note no longer shows.
func (s *Server) handleNotesHistory(w http.ResponseWriter, r *http.Request) {
	if s.notes == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}
	var body struct {
		NoteID string `json:"noteId"`
		Limit  int    `json:"limit"`
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

	edits, err := s.notes.Edits(r.Context(), info.User.ID, body.NoteID, body.Limit)
	if errors.Is(err, note.ErrNotFound) {
		writeAPIError(w, &APIError{
			Message: "No such note.", Code: "NO_SUCH_NOTE",
			ID: errIDNoteEditNoNote, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	if errors.Is(err, note.ErrNotNoteAuthor) {
		writeAPIError(w, &APIError{
			Message: "You are not the author of this note.", Code: "YOU_ARE_NOT_THE_AUTHOR",
			ID: errIDYouNotAuthor, Kind: "client", Status: http.StatusForbidden,
		})
		return
	}
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]map[string]any, 0, len(edits))
	for _, e := range edits {
		out = append(out, map[string]any{
			"id":        e.ID,
			"noteId":    e.NoteID,
			"text":      e.Text,
			"cw":        e.CW,
			"fileIds":   e.FileIDs,
			"createdAt": e.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
