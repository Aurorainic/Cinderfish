package httpapi

import (
	"context"
	"net/http"
	"time"

	"milnfish/internal/note"
	"milnfish/internal/streaming"
)

// Event publishing.
//
// Every site below mirrors a publish call in misskey-dev/misskey's
// GlobalEventService and the service that calls it; see
// docs/research/2026-10-28-m4-streaming-source-calibration.md §6. The
// bus is always non-nil and Publish is a no-op when nothing is
// subscribed, so handlers publish unconditionally.

// publishNoteCreated announces a new note on notesStream and emits the
// per-user main channel events for replies and renotes.
//
// Upstream publishes notesStream once, packed for an anonymous viewer,
// and each connection decides whether it may see the note. Mentions are
// published separately once text is parsed for @-references; MilnFish
// does not parse mentions yet, so n.Mentions is empty.
func (s *Server) publishNoteCreated(n *note.Note, authorID string) {
	if s.streamBus == nil {
		return
	}

	packed := s.packNoteFor(n, "")
	s.streamBus.PublishNote(packed)

	// Reply: notify the author of the replied-to note.
	if n.ReplyID != nil && n.ReplyUserID != nil && *n.ReplyUserID != authorID {
		s.streamBus.PublishTyped(streaming.MainSubject(*n.ReplyUserID), "reply", packed)
	}

	// Renote: notify the author of the renoted note, unless they renoted
	// their own note.
	if n.RenoteID != nil && n.RenoteUserID != nil && *n.RenoteUserID != authorID {
		s.streamBus.PublishTyped(streaming.MainSubject(*n.RenoteUserID), "renote", packed)
	}
}

// publishMentionEvent emits the main channel "mention" event, which
// carries the note in detail form packed for the mentioned user, as
// Misskey's createMentionedEvents does.
func (s *Server) publishMentionEvent(n *note.Note, targetUserID string) {
	if s.streamBus == nil {
		return
	}
	packed := s.packNoteFor(n, targetUserID)
	s.streamBus.PublishTyped(streaming.MainSubject(targetUserID), "mention", packed)
}

// publishNotificationCreated emits a notification event to its
// recipient's main channel.
func (s *Server) publishNotificationCreated(notificationID, recipientID string) {
	if s.streamBus == nil {
		return
	}
	packed, err := s.streamingPacker().PackNotification(notificationID, recipientID)
	if err != nil {
		return
	}
	s.streamBus.PublishTyped(streaming.MainSubject(recipientID), "notification", packed)
}

// publishNoteReaction emits a reaction change on a note's stream.
func (s *Server) publishNoteReaction(n *note.Note, typ, emoji, userID string) {
	if s.streamBus == nil {
		return
	}
	body := map[string]any{
		"reaction": emoji,
		"userId":   userID,
	}
	if typ == "reacted" {
		// Misskey sends emoji: null for a plain unicode reaction.
		body["emoji"] = nil
	}
	s.streamBus.PublishNoteStream(n.ID, n.UserID, n.Visibility,
		s.notes.VisibleUserIDs(context.Background(), n.ID), typ, body)
}

// publishNoteDeleted emits the deletion event on a note's stream.
func (s *Server) publishNoteDeleted(n *note.Note) {
	if s.streamBus == nil {
		return
	}
	s.streamBus.PublishNoteStream(n.ID, n.UserID, n.Visibility,
		s.notes.VisibleUserIDs(context.Background(), n.ID), "deleted",
		map[string]any{"deletedAt": time.Now().UTC().Format(time.RFC3339Nano)})
}

// streamingPacker returns the packer used for streaming payloads.
func (s *Server) streamingPacker() streamingPacker { return streamingPacker{s: s} }

// publishFollowEvent emits the streaming follow event.
//
// Misskey publishes "follow" to the followee's main channel and
// "followed" to the follower's. Both are emitted here so a client that
// has both channels open sees a consistent state.
func (s *Server) publishFollowEvent(followerID, followeeID string) {
	if s.streamBus == nil {
		return
	}
	s.streamBus.PublishTyped(streaming.MainSubject(followeeID), "follow", map[string]any{"id": followerID})
	s.streamBus.PublishTyped(streaming.MainSubject(followerID), "followed", map[string]any{"id": followeeID})
}

// publishUnfollowEvent emits the streaming unfollow event.
func (s *Server) publishUnfollowEvent(followerID, followeeID string) {
	if s.streamBus == nil {
		return
	}
	s.streamBus.PublishTyped(streaming.MainSubject(followeeID), "unfollow", map[string]any{"id": followerID})
}

// writeInternalServerError emits the generic server-side error envelope.
func writeInternalServerError(w http.ResponseWriter) {
	writeAPIError(w, &APIError{
		Message: "Internal error.", Code: "INTERNAL_ERROR",
		ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
	})
}
