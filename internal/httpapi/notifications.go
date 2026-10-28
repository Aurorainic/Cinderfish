package httpapi

import (
	"context"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/streaming"
)

// markAllNotificationsRead marks every notification of a user read and
// announces it on the user's main channel, so every open client clears
// its unread badge without polling.
func (s *Server) markAllNotificationsRead(ctx context.Context, userID string) error {
	if err := s.notifications.MarkAllRead(ctx, userID); err != nil {
		return err
	}
	if s.streamBus != nil {
		s.streamBus.Publish(streaming.MainSubject(userID), streaming.NullMessage("readAllNotifications"))
	}
	return nil
}

// handleNotifications implements POST /api/notifications.
func (s *Server) handleNotifications(w http.ResponseWriter, r *http.Request) {
	if s.notifications == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		Limit        int      `json:"limit"`
		SinceID      string   `json:"sinceId"`
		UntilID      string   `json:"untilId"`
		MarkAsRead   bool     `json:"markAsRead"`
		IncludeTypes []string `json:"includeTypes"`
		ExcludeTypes []string `json:"excludeTypes"`
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

	exclude := body.ExcludeTypes
	if body.MarkAsRead {
		exclude = append(exclude, "read")
	}

	notifications, err := s.notifications.List(r.Context(), info.User.ID, body.Limit, body.SinceID, body.UntilID, exclude)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "Could not load the notifications.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		})
		return
	}

	out := make([]any, 0, len(notifications))
	for _, n := range notifications {
		item := map[string]any{
			"id":        n.ID,
			"createdAt": n.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
			"type":      n.Type,
			"isRead":    n.IsRead,
			"userId":    n.NotifierID,
			"noteId":    n.NoteID,
			"reaction":  n.Reaction,
			"body":      "",
		}
		if n.NotifierID != nil {
			if author, err := s.accounts.GetByID(r.Context(), *n.NotifierID); err == nil {
				item["user"] = packUserLite(author, "")
			}
		}
		if n.NoteID != nil {
			if n, err := s.notes.Get(r.Context(), *n.NoteID); err == nil {
				pc := s.notePackContext(r.Context(), n, info.User.ID)
				item["note"] = packNoteDetail(r.Context(), s, n, pc)
			}
		}
		out = append(out, item)
	}

	writeJSON(w, http.StatusOK, out)
}

// handleNotificationsMarkAllAsRead implements
// POST /api/notifications/mark-all-as-read.
func (s *Server) handleNotificationsMarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	if s.notifications == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}

	if err := s.markAllNotificationsRead(r.Context(), info.User.ID); err != nil {
		writeAPIError(w, &APIError{
			Message: "Could not mark the notifications as read.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}
