package httpapi

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/note"
	"github.com/Aurorainic/cinderfish/internal/streaming"
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
// published separately once text is parsed for @-references; CinderFish
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

// publishDriveFileCreated emits the main channel "driveFileCreated" event.
// Firefish's photos widget listens for it to insert a new image without a
// reload; Misskey publishes the same event from its drive service.
func (s *Server) publishDriveFileCreated(userID string, file map[string]any) {
	if s.streamBus == nil {
		return
	}
	s.streamBus.PublishTyped(streaming.MainSubject(userID), "driveFileCreated", file)
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

// notifyFollowersOfNote records a "note" notification for every local
// follower whose follow asks for one (notify = "normal"), which is the
// setting following/update writes.
//
// Direct notes are skipped: their audience is addressed explicitly, and a
// follower outside that audience must not learn the note exists.
func (s *Server) notifyFollowersOfNote(ctx context.Context, n *note.Note, authorID string) {
	if s.notifications == nil || s.pool == nil || n == nil {
		return
	}
	if n.Visibility == note.VisibilitySpecified {
		return
	}
	rows, err := s.pool.Query(ctx, `
SELECT f.follower_id FROM followings f JOIN users u ON u.id = f.follower_id
WHERE f.followee_id = $1 AND f.notify IS NOT NULL AND u.host IS NULL
`, authorID)
	if err != nil {
		log.Printf("httpapi: followers to notify of %s: %v", n.ID, err)
		return
	}
	var recipients []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			log.Printf("httpapi: scan notification recipient: %v", err)
			return
		}
		recipients = append(recipients, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		log.Printf("httpapi: followers to notify of %s: %v", n.ID, err)
		return
	}

	for _, recipientID := range recipients {
		notificationID, created, err := s.notifications.CreateForLocal(
			ctx, recipientID, "note", &authorID, &n.ID, nil)
		if err != nil || !created {
			continue
		}
		s.publishNotificationCreated(notificationID, recipientID)
	}
}
