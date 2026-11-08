package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/streaming"
)

// Misskey error ids for the messaging endpoints (the fork-preserved
// legacy contract; see docs/HANDOVER.md).
const (
	errIDNoSuchMessageDelete = "54b5b326-7925-42cf-8019-130fda8b56af"
	errIDNoSuchMessageRead   = "86d56a2f-a9c3-4afb-b13c-3e9bfef9aa14"
	errIDNoSuchUserMessaging = "11795c64-40ea-4198-b06e-3c873ed9039d"
	errIDRecipientIsYourself = "17e2ba79-e22a-4cbc-bf91-d327643f4a7e"
	errIDNoSuchFileMessage   = "4372b8e2-185d-4146-8749-2f68864a3e5f"
	errIDMessageNoContent    = "25587321-b0e6-449c-9239-f8925092942c"
	errIDRecipientBlockedYou = "c15a5199-7422-4968-941a-2a462c478f7d"
)

// packMessagingMessage renders the 1:1 MessagingMessage schema. Group
// fields stay null; the embedded user is the sender, the recipient is
// embedded in the shapes upstream packs with populateRecipient enabled.
func (s *Server) packMessagingMessage(ctx context.Context, m *account.MessagingMessage, embedRecipient bool) map[string]any {
	packed := map[string]any{
		"id":          m.ID,
		"createdAt":   m.CreatedAt.UTC().Format(time.RFC3339Nano),
		"userId":      m.UserID,
		"text":        m.Text,
		"fileId":      m.FileID,
		"file":        nil,
		"recipientId": m.RecipientID,
		"groupId":     nil,
		"isRead":      m.IsRead,
		"reads":       []any{},
	}
	if m.FileID != nil && s.drive != nil {
		if file, err := s.drive.Get(ctx, *m.FileID); err == nil && file != nil {
			packed["file"] = packDriveFile(file)
		}
	}
	if u, err := s.accounts.GetByID(ctx, m.UserID); err == nil && u != nil {
		packed["user"] = packUserLite(u, "", loadProfile(ctx, s, m.UserID))
	}
	if embedRecipient {
		if u, err := s.accounts.GetByID(ctx, m.RecipientID); err == nil && u != nil {
			packed["recipient"] = packUserLite(u, "", loadProfile(ctx, s, m.RecipientID))
		}
	}
	return packed
}

// writeMessagingError maps the message service sentinels onto Misskey's
// envelope with the fork-preserved ids.
func writeMessagingError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, account.ErrNoSuchMessage):
		// The caller picked the per-endpoint id before mapping.
		writeAPIError(w, &APIError{
			Message: "No such message.", Code: "NO_SUCH_MESSAGE",
			ID: errIDNoSuchMessageRead, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrRecipientIsYourself):
		writeAPIError(w, &APIError{
			Message: "You cannot send a message to yourself.", Code: "RECIPIENT_IS_YOURSELF",
			ID: errIDRecipientIsYourself, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrMessageNoContent):
		writeAPIError(w, &APIError{
			Message: "Content required. You need to enter some text or a file.",
			Code:    "CONTENT_REQUIRED",
			ID:      errIDMessageNoContent, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrMessageFileNotFound):
		writeAPIError(w, &APIError{
			Message: "No such file.", Code: "NO_SUCH_FILE",
			ID: errIDNoSuchFileMessage, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrRecipientBlockedYou):
		writeAPIError(w, &APIError{
			Message: "You have been blocked by that user.", Code: "YOU_HAVE_BEEN_BLOCKED",
			ID: errIDRecipientBlockedYou, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrMessageTextTooLong):
		writeAPIError(w, invalidParamError("text is too long"))
	default:
		writeInternalServerError(w)
	}
}

// handleMessagingHistory implements POST /api/messaging/history. The
// group branch is a recorded deferral and answers an empty list rather
// than an error, matching the empty-history shape the index page renders.
func (s *Server) handleMessagingHistory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Limit int  `json:"limit"`
		Group bool `json:"group"`
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
	if body.Group {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	messages, err := s.accounts.HistoryMessages(r.Context(), info.User.ID, body.Limit)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]any, 0, len(messages))
	for _, m := range messages {
		out = append(out, s.packMessagingMessage(r.Context(), m, true))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleMessagingMessages implements POST /api/messaging/messages.
func (s *Server) handleMessagingMessages(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID     string `json:"userId"`
		Limit      int    `json:"limit"`
		SinceID    string `json:"sinceId"`
		UntilID    string `json:"untilId"`
		MarkAsRead *bool  `json:"markAsRead"`
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
	if body.UserID == "" {
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDNoSuchUserMessaging, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	target, err := s.accounts.GetByID(r.Context(), body.UserID)
	if err != nil || target == nil {
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDNoSuchUserMessaging, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	markAsRead := true
	if body.MarkAsRead != nil {
		markAsRead = *body.MarkAsRead
	}
	messages, err := s.accounts.ThreadMessages(r.Context(), info.User.ID, body.UserID, account.MessagingThreadQuery{
		Limit:      body.Limit,
		SinceID:    body.SinceID,
		UntilID:    body.UntilID,
		MarkAsRead: markAsRead,
		ReaderID:   info.User.ID,
	})
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if markAsRead {
		if readErr := s.publishMessagesRead(r.Context(), info.User.ID, body.UserID); readErr != nil {
			log.Printf("httpapi: messaging read event: %v", readErr)
		}
	}
	out := make([]any, 0, len(messages))
	for _, m := range messages {
		out = append(out, s.packMessagingMessage(r.Context(), m, false))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleMessagingMessagesCreate implements POST
// /api/messaging/messages/create.
func (s *Server) handleMessagingMessagesCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string  `json:"userId"`
		Text   *string `json:"text"`
		FileID *string `json:"fileId"`
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
	if body.UserID == "" {
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDNoSuchUserMessaging, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	m, err := s.accounts.CreateMessage(r.Context(), info.User.ID, body.UserID, body.Text, body.FileID)
	if err != nil {
		writeMessagingError(w, err)
		return
	}
	s.publishMessageCreated(m)
	packed := s.packMessagingMessage(r.Context(), m, true)
	writeJSON(w, http.StatusOK, packed)
}

// handleMessagingMessagesDelete implements POST
// /api/messaging/messages/delete.
func (s *Server) handleMessagingMessagesDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		MessageID string `json:"messageId"`
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
	m, err := s.accounts.GetMessage(r.Context(), body.MessageID, info.User.ID)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such message.", Code: "NO_SUCH_MESSAGE",
			ID: errIDNoSuchMessageDelete, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	// Only the author may delete, as upstream's delete checks the sender.
	if m.UserID != info.User.ID {
		writeAPIError(w, &APIError{
			Message: "No such message.", Code: "NO_SUCH_MESSAGE",
			ID: errIDNoSuchMessageDelete, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if err := s.accounts.DeleteMessage(r.Context(), m.ID); err != nil {
		writeInternalServerError(w)
		return
	}
	s.publishMessageDeleted(m)
	w.WriteHeader(http.StatusNoContent)
}

// handleMessagingMessagesRead implements POST /api/messaging/messages/read.
func (s *Server) handleMessagingMessagesRead(w http.ResponseWriter, r *http.Request) {
	var body struct {
		MessageID string `json:"messageId"`
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
	m, err := s.accounts.GetMessage(r.Context(), body.MessageID, info.User.ID)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such message.", Code: "NO_SUCH_MESSAGE",
			ID: errIDNoSuchMessageRead, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if m.RecipientID != info.User.ID {
		// Only the recipient reads; the caller's own message is already
		// read from their perspective.
		writeAPIError(w, &APIError{
			Message: "No such message.", Code: "NO_SUCH_MESSAGE",
			ID: errIDNoSuchMessageRead, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if err := s.accounts.MarkMessageRead(r.Context(), info.User.ID, m.ID); err != nil {
		writeMessagingError(w, err)
		return
	}
	s.publishMessagesRead(r.Context(), info.User.ID, m.UserID)
	w.WriteHeader(http.StatusNoContent)
}

// handleIReadAllMessagingMessages implements POST
// /api/i/read-all-messaging-messages.
func (s *Server) handleIReadAllMessagingMessages(w http.ResponseWriter, r *http.Request) {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	if _, err := s.accounts.MarkAllMessagesRead(r.Context(), info.User.ID); err != nil {
		writeInternalServerError(w)
		return
	}
	if s.streamBus != nil {
		s.streamBus.PublishTyped(streaming.MainSubject(info.User.ID), "readAllMessagingMessages", nil)
	}
	w.WriteHeader(http.StatusNoContent)
}

// publishMessageCreated sends the packed message to both sides' streams
// and main channels, then schedules the 2-second unread event, matching
// the legacy delivery order.
func (s *Server) publishMessageCreated(m *account.MessagingMessage) {
	if s.streamBus == nil {
		return
	}
	packed := s.packMessagingMessage(context.Background(), m, false)
	subject := streaming.MessagingStreamSubject(m.UserID, m.RecipientID)
	s.streamBus.PublishTyped(subject, "message", packed)
	s.streamBus.PublishTyped(streaming.MessagingIndexSubject(m.UserID), "message", packed)
	s.streamBus.PublishTyped(streaming.MessagingIndexSubject(m.RecipientID), "message", packed)
	s.streamBus.PublishTyped(streaming.MainSubject(m.UserID), "messagingMessage", packed)
	s.streamBus.PublishTyped(streaming.MainSubject(m.RecipientID), "messagingMessage", packed)

	// Upstream delays the unread event by two seconds and only fires it
	// if the recipient has not read the message by then.
	time.AfterFunc(2*time.Second, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		still, err := s.accounts.GetMessage(ctx, m.RecipientID, m.RecipientID)
		if err != nil || still == nil || still.IsRead {
			return
		}
		s.streamBus.PublishTyped(streaming.MainSubject(m.RecipientID), "unreadMessagingMessage",
			s.packMessagingMessage(ctx, m, false))
	})
}

// publishMessagesRead emits the read receipts on both sides of the
// conversation and clears the unread marker when nothing remains.
func (s *Server) publishMessagesRead(ctx context.Context, readerID, otherID string) error {
	if s.streamBus == nil {
		return nil
	}
	remaining, err := s.accounts.HasUnreadMessages(ctx, readerID)
	if err != nil {
		return err
	}
	body := map[string]any{"readerId": readerID}
	subject := streaming.MessagingStreamSubject(readerID, otherID)
	s.streamBus.PublishTyped(subject, "read", body)
	s.streamBus.PublishTyped(streaming.MessagingIndexSubject(readerID), "read", body)
	if !remaining {
		s.streamBus.PublishTyped(streaming.MainSubject(readerID), "readAllMessagingMessages", nil)
	}
	return nil
}

// publishMessageDeleted emits the deletion on both sides' feeds.
func (s *Server) publishMessageDeleted(m *account.MessagingMessage) {
	if s.streamBus == nil {
		return
	}
	subject := streaming.MessagingStreamSubject(m.UserID, m.RecipientID)
	s.streamBus.PublishTyped(subject, "deleted", map[string]any{"id": m.ID})
	s.streamBus.PublishTyped(streaming.MessagingIndexSubject(m.UserID), "deleted", map[string]any{"id": m.ID})
	s.streamBus.PublishTyped(streaming.MessagingIndexSubject(m.RecipientID), "deleted", map[string]any{"id": m.ID})
}

// publishMessagingTyping answers the typingOnMessaging frame: the typer's
// packed user rides the conversation subject.
func (s *Server) publishMessagingTyping(userID, partner string) {
	if s.streamBus == nil {
		return
	}
	u, err := s.accounts.GetByID(context.Background(), userID)
	if err != nil || u == nil {
		return
	}
	s.streamBus.PublishTyped(streaming.MessagingStreamSubject(userID, partner), "typing",
		packUserLite(u, "", loadProfile(context.Background(), s, userID)))
}
