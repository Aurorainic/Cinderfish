package httpapi

import (
	"context"

	"github.com/Aurorainic/cinderfish/internal/note"
)

// dispatchNoteWebhook tells the author's webhooks about a new note. The event
// type follows the note's shape: reply and renote are separate events in
// Misskey's enum, everything else is "note".
//
// Delivery is fire-and-forget inside the webhook service; the note request does
// not wait for third-party servers.
func (s *Server) dispatchNoteWebhook(ctx context.Context, n *note.Note, userID string) {
	if s.webhooks == nil || n == nil {
		return
	}
	eventType := "note"
	switch {
	case n.ReplyID != nil:
		eventType = "reply"
	case n.RenoteID != nil:
		eventType = "renote"
	}
	pc := s.notePackContext(ctx, n, userID)
	body := packNoteDetail(ctx, s, n, pc)
	s.webhooks.Dispatch(userID, eventType, body)
}
