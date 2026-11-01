package httpapi

import (
	"context"
	"fmt"
	"time"

	"github.com/Aurorainic/cinderfish/internal/note"
	"github.com/Aurorainic/cinderfish/internal/streaming"
)

// To satisfy the packer interface, note entities are rendered with the
// same helpers the HTTP endpoints use.

// packNoteFor renders a single note for a viewer. The streaming packer
// calls this for every published note.
func (s *Server) packNoteFor(n *note.Note, viewerID string) map[string]any {
	pc := s.notePackContext(context.Background(), n, viewerID)
	return s.packNoteDetailFor(n, pc)
}

// packNoteDetailFor renders a note in the Note schema for the packer
// interface, using the same detail packer the HTTP endpoints use so the
// two paths cannot drift apart.
func (s *Server) packNoteDetailFor(n *note.Note, pc notePackContext) map[string]any {
	return packNoteDetail(context.Background(), s, n, pc)
}

// streamingPacker renders stored entities for streaming payloads.
type streamingPacker struct {
	s *Server
}

// PackNote renders a note for a streaming viewer.
func (p streamingPacker) PackNote(noteID, viewerID string, detail bool) (streaming.PackedNote, error) {
	n, err := p.s.notes.Get(context.Background(), noteID)
	if err != nil {
		return nil, err
	}
	return streaming.PackedNote(p.s.packNoteFor(n, viewerID)), nil
}

// PackNotification renders a stored notification for its recipient.
func (p streamingPacker) PackNotification(notificationID, recipientID string) (map[string]any, error) {
	n, err := p.s.notifications.Get(context.Background(), notificationID)
	if err != nil {
		return nil, err
	}
	if n.RecipientID != recipientID {
		return nil, fmt.Errorf("httpapi: notification %s does not belong to %s", notificationID, recipientID)
	}
	return p.s.packNotification(n, recipientID), nil
}

// packNotification renders a notification in Misskey's Notification
// schema. The HTTP endpoint and the streaming main channel share it so a
// client sees the same object from both paths.
func (s *Server) packNotification(n *note.Notification, viewerID string) map[string]any {
	ctx := context.Background()
	item := map[string]any{
		"id":        n.ID,
		"createdAt": n.CreatedAt.UTC().Format(time.RFC3339Nano),
		"type":      n.Type,
		"isRead":    n.IsRead,
		"userId":    n.NotifierID,
		"noteId":    n.NoteID,
		"reaction":  n.Reaction,
	}
	if n.NotifierID != nil {
		if author, err := s.accounts.GetByID(ctx, *n.NotifierID); err == nil {
			item["user"] = packUserLite(author, "", nil)
		}
	}
	if n.NoteID != nil {
		if stored, err := s.notes.Get(ctx, *n.NoteID); err == nil {
			item["note"] = s.packNoteFor(stored, viewerID)
		}
	}
	return item
}
