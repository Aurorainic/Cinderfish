package httpapi

import (
	"errors"
	"net/http"
	"time"

	"milnfish/internal/note"
)

// Error ids for the note endpoints, copied from Misskey.
const (
	errIDNoSuchRenoteTarget    = "b5c90186-4ab0-49c8-9bba-a1f76c282ba4"
	errIDCannotReRenote        = "fd4cc33e-2a37-48dd-99cc-9b806eb2031a"
	errIDCannotRenoteVis       = "be9529e9-fe72-4de0-ae43-0b363c4938af"
	errIDNoSuchReplyTarget     = "749ee0f6-d3da-459a-bf02-282e2da4292c"
	errIDCannotReplyInvisible  = "b98980fa-3780-406c-a935-b6d0eeee10d1"
	errIDCannotReplyPureRenote = "3ac74a84-8fd5-4bb0-870f-01804f82ce15"
	errIDNoteTooLong           = "6d51c8a2-1e1e-4f7f-b1c9-9f5a0e2d0f3b"
	errIDNoteEmpty             = "8f7d1c2e-2b1c-4a3e-9c5a-1e2d3c4b5a6f"
	errIDNoSuchNote            = "24fcbfc6-2e37-42b6-8388-c29b3861a08d"
)

// handleNotesCreate implements POST /api/notes/create.
func (s *Server) handleNotesCreate(w http.ResponseWriter, r *http.Request) {
	if s.notes == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		Visibility     string   `json:"visibility"`
		VisibleUserIDs []string `json:"visibleUserIds"`
		CW             *string  `json:"cw"`
		LocalOnly      bool     `json:"localOnly"`
		Text           *string  `json:"text"`
		FileIDs        []string `json:"fileIds"`
		MediaIDs       []string `json:"mediaIds"`
		ReplyID        *string  `json:"replyId"`
		RenoteID       *string  `json:"renoteId"`
		Poll           *struct {
			Choices      []string `json:"choices"`
			Multiple     bool     `json:"multiple"`
			ExpiresAt    *int64   `json:"expiresAt"`
			ExpiredAfter *int64   `json:"expiredAfter"`
		} `json:"poll"`
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

	fileIDs := body.FileIDs
	if len(fileIDs) == 0 {
		fileIDs = body.MediaIDs
	}

	params := note.CreateParams{
		UserID:         info.User.ID,
		Visibility:     body.Visibility,
		LocalOnly:      body.LocalOnly,
		CW:             body.CW,
		Text:           body.Text,
		FileIDs:        fileIDs,
		VisibleUserIDs: body.VisibleUserIDs,
		ReplyID:        body.ReplyID,
		RenoteID:       body.RenoteID,
	}
	if body.Poll != nil {
		poll := &note.PollParams{
			Choices:  body.Poll.Choices,
			Multiple: body.Poll.Multiple,
		}
		switch {
		case body.Poll.ExpiredAfter != nil:
			exp := time.Now().Add(time.Duration(*body.Poll.ExpiredAfter) * time.Millisecond)
			poll.ExpiresAt = &exp
		case body.Poll.ExpiresAt != nil:
			exp := time.UnixMilli(*body.Poll.ExpiresAt)
			poll.ExpiresAt = &exp
		}
		params.Poll = poll
	}

	n, err := s.notes.Create(r.Context(), params)
	if err != nil {
		s.writeNoteError(w, err)
		return
	}

	// Notify the users the note is addressed to.
	s.notifyMentioned(r, n, info.User.ID)
	// Fan the note out to the streaming timelines and main channels.
	s.publishNoteCreated(n, info.User.ID)

	pc := s.notePackContext(r.Context(), n, info.User.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"createdNote": packNoteDetail(r.Context(), s, n, pc),
	})
}

// notifyMentioned creates mention notifications.
func (s *Server) notifyMentioned(r *http.Request, n *note.Note, actorID string) {
	if len(n.Mentions) == 0 {
		return
	}
	for _, target := range n.Mentions {
		if target == actorID {
			continue
		}
		notificationID, err := s.notifications.CreateReturningID(r.Context(), target, "mention", &actorID, &n.ID, nil)
		if err != nil {
			continue
		}
		s.publishMentionEvent(n, target)
		s.publishNotificationCreated(notificationID, target)
	}
}

// writeNoteError maps a note service error onto Misskey's error shape.
func (s *Server) writeNoteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, note.ErrNoSuchReplyTarget):
		writeAPIError(w, &APIError{
			Message: "No such reply target.", Code: "NO_SUCH_REPLY_TARGET",
			ID: errIDNoSuchReplyTarget, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, note.ErrNoSuchRenoteTarget):
		writeAPIError(w, &APIError{
			Message: "No such renote target.", Code: "NO_SUCH_RENOTE_TARGET",
			ID: errIDNoSuchRenoteTarget, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, note.ErrCannotRenoteToPureRenote):
		writeAPIError(w, &APIError{
			Message: "You can not Renote a pure Renote.", Code: "CANNOT_RENOTE_TO_A_PURE_RENOTE",
			ID: errIDCannotReRenote, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, note.ErrCannotReplyToPureRenote):
		writeAPIError(w, &APIError{
			Message: "You can not reply to a pure Renote.", Code: "CANNOT_REPLY_TO_A_PURE_RENOTE",
			ID: errIDCannotReplyPureRenote, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, note.ErrTextTooLong):
		writeAPIError(w, &APIError{
			Message: "Note is too long.", Code: "NOTE_TOO_LONG",
			ID: errIDNoteTooLong, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, note.ErrEmptyNote):
		writeAPIError(w, &APIError{
			Message: "A note needs text, files or a poll.", Code: "NOTE_EMPTY",
			ID: errIDNoteEmpty, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, note.ErrInvalidVisibility):
		writeAPIError(w, &APIError{
			Message: "Invalid visibility.", Code: "INVALID_VISIBILITY",
			ID: errIDInvalidParam, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, note.ErrPollExpired):
		writeAPIError(w, &APIError{
			Message: "Poll is already expired.", Code: "CANNOT_CREATE_ALREADY_EXPIRED_POLL",
			ID: "04da457d-b083-4055-9082-955525eda5a5", Kind: "client", Status: http.StatusBadRequest,
		})
	default:
		writeAPIError(w, &APIError{
			Message: "Could not create the note.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		})
	}
}

// handleNotesShow implements POST /api/notes/show.
func (s *Server) handleNotesShow(w http.ResponseWriter, r *http.Request) {
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

	info, _ := s.authenticate(r.Context(), r, bodyAsMap(r))
	viewerID := ""
	if info != nil {
		viewerID = info.User.ID
	}

	n, err := s.notes.Get(r.Context(), body.NoteID)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such note.", Code: "NO_SUCH_NOTE",
			ID: errIDNoSuchNote, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}

	pc := s.notePackContext(r.Context(), n, viewerID)
	writeJSON(w, http.StatusOK, packNoteDetail(r.Context(), s, n, pc))
}

// handleNotesDelete implements POST /api/notes/delete.
func (s *Server) handleNotesDelete(w http.ResponseWriter, r *http.Request) {
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

	// Load the note first: the deletion event carries its visibility and
	// audience, which are gone once the row is soft deleted.
	n, err := s.notes.Get(r.Context(), body.NoteID)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such note.", Code: "NO_SUCH_NOTE",
			ID: errIDNoSuchNote, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}

	if err := s.notes.Delete(r.Context(), body.NoteID, info.User.ID); err != nil {
		writeAPIError(w, &APIError{
			Message: "No such note.", Code: "NO_SUCH_NOTE",
			ID: errIDNoSuchNote, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}

	s.publishNoteDeleted(n)
	writeJSON(w, http.StatusOK, map[string]any{})
}

// handleNotesReactionsCreate implements POST /api/notes/reactions/create.
func (s *Server) handleNotesReactionsCreate(w http.ResponseWriter, r *http.Request) {
	if s.noteReactions == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		NoteID string `json:"noteId"`
		Emoji  string `json:"reaction"`
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
	if body.Emoji == "" {
		writeAPIError(w, invalidParamError(map[string]any{"reason": "reaction is required"}))
		return
	}

	n, err := s.notes.Get(r.Context(), body.NoteID)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such note.", Code: "NO_SUCH_NOTE",
			ID: errIDNoSuchNote, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}

	if err := s.noteReactions.Add(r.Context(), body.NoteID, info.User.ID, body.Emoji); err != nil {
		writeAPIError(w, &APIError{
			Message: "Could not add the reaction.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		})
		return
	}

	// The reaction event goes out regardless of who reacted.
	s.publishNoteReaction(n, "reacted", body.Emoji, info.User.ID)

	// Notify the author, unless the reaction is on one's own note.
	if n.UserID != info.User.ID {
		if notificationID, err := s.notifications.CreateReturningID(
			r.Context(), n.UserID, "reaction", &info.User.ID, &n.ID, &body.Emoji); err == nil {
			s.publishNotificationCreated(notificationID, n.UserID)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{})
}

// handleNotesReactionsDelete implements POST /api/notes/reactions/delete.
func (s *Server) handleNotesReactionsDelete(w http.ResponseWriter, r *http.Request) {
	if s.noteReactions == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		NoteID string `json:"noteId"`
		Emoji  string `json:"reaction"`
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

	// When the client omits the emoji, Misskey removes whatever reaction
	// the user left, so resolve it before deleting.
	emoji := body.Emoji
	if emoji == "" {
		emoji, _ = s.noteReactions.MyReaction(r.Context(), body.NoteID, info.User.ID)
	}

	if err := s.noteReactions.Remove(r.Context(), body.NoteID, info.User.ID, emoji); err != nil {
		writeAPIError(w, &APIError{
			Message: "Could not remove the reaction.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		})
		return
	}

	// Publish only when a reaction was actually named: an empty emoji
	// means the user had none to withdraw, and the event would be a
	// phantom. Reacting is idempotent, so no signature change is needed
	// on Remove.
	if emoji != "" {
		if n, err := s.notes.Get(r.Context(), body.NoteID); err == nil {
			s.publishNoteReaction(n, "unreacted", emoji, info.User.ID)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}
