package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/note"
)

// Error ids for the note endpoints, copied from Misskey.
const (
	errIDNoSuchRenoteTarget    = "b5c90186-4ab0-49c8-9bba-a1f76c282ba4"
	errIDCannotReRenote        = "fd4cc33e-2a37-48dd-99cc-9b806eb2031a"
	errIDCannotRenoteVis       = "be9529e9-fe72-4de0-ae43-0b363c4938af"
	errIDNoSuchReplyTarget     = "749ee0f6-d3da-459a-bf02-282e2da4292c"
	errIDCannotReplyInvisible  = "b98980fa-3780-406c-a935-b6d0eeee10d1"
	errIDCannotReplyPureRenote = "3ac74a84-8fd5-4bb0-870f-01804f82ce15"
	// CANNOT_REPLY_TO_SPECIFIED_VISIBILITY_NOTE_WITH_EXTENDED_VISIBILITY
	errIDCannotReplySpecifiedExtended = "ed940410-535c-4d5e-bfa3-af798671e93c"
	errIDNoteTooLong                  = "6d51c8a2-1e1e-4f7f-b1c9-9f5a0e2d0f3b"
	errIDNoteEmpty                    = "8f7d1c2e-2b1c-4a3e-9c5a-1e2d3c4b5a6f"
	errIDNoSuchNote                   = "24fcbfc6-2e37-42b6-8388-c29b3861a08d"
	// notes/reactions/create has its own NO_SUCH_NOTE id.
	errIDNoSuchNoteReaction  = "033d0620-5bfe-4027-965d-980b0c85a3ea"
	errIDCannotReactToRenote = "eaccdc08-ddef-43fe-908f-d108faad57f5"
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

	// Notify the users the note reaches.
	s.notifyNoteCreated(r, n, info.User.ID)
	// Fan the note out to the streaming timelines and main channels.
	s.publishNoteCreated(n, info.User.ID)

	pc := s.notePackContext(r.Context(), n, info.User.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"createdNote": packNoteDetail(r.Context(), s, n, pc),
	})
}

// notifyNoteCreated queues the notifications a new note produces, in
// Misskey's order and with its precedence: mentions first, then the reply
// target, then the author of the renoted note, where a later non-mention
// reason replaces an earlier mention of the same user
// (core/NoteCreateService.ts NotificationManager, L64-162 and L856-895).
// The note's author never notifies themselves, a specified note notifies
// only its audience, and a remote recipient gets no row because
// ActivityPub carries that copy.
func (s *Server) notifyNoteCreated(r *http.Request, n *note.Note, actorID string) {
	if s.notifications == nil {
		return
	}

	reasons := make(map[string]string, len(n.Mentions)+2)
	targets := make([]string, 0, len(n.Mentions)+2)
	push := func(target, reason string) {
		if target == "" || target == actorID {
			return
		}
		if _, seen := reasons[target]; seen {
			if reason != "mention" {
				reasons[target] = reason
			}
			return
		}
		reasons[target] = reason
		targets = append(targets, target)
	}

	for _, mentioned := range n.Mentions {
		push(mentioned, "mention")
	}
	if n.ReplyID != nil && n.ReplyUserID != nil {
		push(*n.ReplyUserID, "reply")
	}
	if n.RenoteID != nil && n.RenoteUserID != nil {
		reason := "renote"
		if n.IsQuote() {
			reason = "quote"
		}
		push(*n.RenoteUserID, reason)
	}
	if len(targets) == 0 {
		return
	}

	addressed := map[string]bool{}
	if n.Visibility == note.VisibilitySpecified {
		for _, id := range s.notes.VisibleUserIDs(r.Context(), n.ID) {
			addressed[id] = true
		}
	}

	for _, target := range targets {
		if n.Visibility == note.VisibilitySpecified && !addressed[target] {
			continue
		}
		notificationID, created, err := s.notifications.CreateForLocal(
			r.Context(), target, reasons[target], &actorID, &n.ID, nil)
		if err != nil || !created {
			continue
		}
		if reasons[target] == "mention" {
			s.publishMentionEvent(n, target)
		}
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
	case errors.Is(err, note.ErrCannotReplyToInvisibleNote):
		writeAPIError(w, &APIError{
			Message: "You cannot reply to an invisible Note.", Code: "CANNOT_REPLY_TO_AN_INVISIBLE_NOTE",
			ID: errIDCannotReplyInvisible, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, note.ErrCannotWidenSpecifiedReply):
		writeAPIError(w, &APIError{
			Message: "You cannot reply to a specified visibility note with extended visibility.",
			Code:    "CANNOT_REPLY_TO_SPECIFIED_VISIBILITY_NOTE_WITH_EXTENDED_VISIBILITY",
			ID:      errIDCannotReplySpecifiedExtended, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, note.ErrCannotRenoteDueToVisibility):
		writeAPIError(w, &APIError{
			Message: "You can not Renote due to target visibility.", Code: "CANNOT_RENOTE_DUE_TO_VISIBILITY",
			ID: errIDCannotRenoteVis, Kind: "client", Status: http.StatusBadRequest,
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

	// A note the viewer may not read is answered like a missing one, so
	// the response does not confirm the note exists. Upstream notes/show
	// has no visibility check, but its ecosystem keeps restricted ids out
	// of an outsider's reach; our packers embed parents by id, so the
	// check has to live here.
	visible, err := s.notes.VisibleTo(r.Context(), n, viewerID)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "Could not load the note.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		})
		return
	}
	if !visible {
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
			ID: errIDNoSuchNoteReaction, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}

	// Misskey refuses reactions on a pure renote: the reaction belongs to
	// the renoted note (core/ReactionService.ts L121-123).
	if n.IsPureRenote() {
		writeAPIError(w, &APIError{
			Message: "You cannot react to Renote.", Code: "CANNOT_REACT_TO_RENOTE",
			ID: errIDCannotReactToRenote, Kind: "client", Status: http.StatusBadRequest,
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
