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

// applyUnreadNotifications fills the MeDetailed unread state from the
// notification table (core/entities/UserEntityService.ts L612 and L614):
// unreadNotificationsCount is the number of unread rows and
// hasUnreadNotification says whether there are any. Types the user muted are
// left out of the badge, matching the list endpoint.
func (s *Server) applyUnreadNotifications(ctx context.Context, userID string, packed map[string]any) {
	if s.notifications == nil {
		return
	}
	count, err := s.notifications.UnreadCount(ctx, userID, s.mutedNotificationTypes(ctx, userID))
	if err != nil {
		return
	}
	packed["unreadNotificationsCount"] = count
	packed["hasUnreadNotification"] = count > 0
}

// mutedNotificationTypes reads the account's muted kinds, tolerating a
// missing profile or a transient read failure by filtering nothing.
func (s *Server) mutedNotificationTypes(ctx context.Context, userID string) []string {
	if s.accounts == nil {
		return nil
	}
	prof, err := s.accounts.GetProfile(ctx, userID)
	if err != nil || prof == nil {
		return nil
	}
	return prof.MutedNotificationTypes
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
		MarkAsRead   *bool    `json:"markAsRead"`
		IncludeTypes []string `json:"includeTypes"`
		ExcludeTypes []string `json:"excludeTypes"`
		UnreadOnly   bool     `json:"unreadOnly"`
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

	// Misskey merges the account's muted types into excludeTypes; an
	// explicit includeTypes still wins, as in NotificationService.
	body.ExcludeTypes = append(body.ExcludeTypes, s.mutedNotificationTypes(r.Context(), info.User.ID)...)

	notifications, err := s.notifications.List(
		r.Context(), info.User.ID, body.Limit, body.SinceID, body.UntilID,
		body.IncludeTypes, body.ExcludeTypes, body.UnreadOnly)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "Could not load the notifications.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		})
		return
	}

	if body.MarkAsRead == nil || *body.MarkAsRead {
		ids := make([]string, 0, len(notifications))
		for _, n := range notifications {
			if !n.IsRead {
				ids = append(ids, n.ID)
			}
		}
		if markErr := s.notifications.MarkRead(r.Context(), info.User.ID, ids); markErr == nil {
			for _, n := range notifications {
				n.IsRead = true
			}
		}
	}

	out := make([]any, 0, len(notifications))
	for _, n := range notifications {
		// The same packer the streaming main channel uses, so the HTTP
		// list and the pushed event cannot drift apart.
		out = append(out, s.packNotification(n, info.User.ID))
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
