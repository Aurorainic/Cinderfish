package streaming

import (
	"encoding/json"
)

// timelineChannel is the shared frame of the four channels that read
// notesStream.
//
// Go embedding does not dispatch to an outer method's override: a
// concrete channel that defines its own onNote would not be reached
// through the embedded timelineChannel.HandleNote. The filter is
// therefore stored as a field, set by each concrete channel's Init.
type timelineChannel struct {
	BaseChannel
	withRenotes bool
	withReplies bool
	withFiles   bool
	// filter is the concrete channel's own note filter.
	filter func(PackedNote)
}

// HandleNote applies the channel's filter.
func (t *timelineChannel) HandleNote(note PackedNote) {
	if t.filter != nil {
		t.filter(note)
	}
}

// HandleNoteUpdate implements Channel: timeline channels do not consume
// per-note update events.
func (t *timelineChannel) HandleNoteUpdate(string, json.RawMessage, NoteMeta) {}

// replyAllowed applies Misskey's reply filter.
//
// When replies are enabled (through the per-follow setting or the
// channel param) only a reply to a followers-only note by an author the
// viewer does not follow is dropped. Otherwise the note survives only
// when it replies to the viewer, was written by the viewer, or replies
// to its own author's note.
func (t *timelineChannel) replyAllowed(note PackedNote, followWithReplies bool) bool {
	reply, ok := replyOf(note)
	if !ok {
		return true
	}
	v := t.viewer()
	replyUserID, _ := reply["userId"].(string)
	authorID, _ := note["userId"].(string)
	isMe := v.UserID != "" && v.UserID == authorID

	if followWithReplies || t.withReplies {
		if visibilityOf(reply) == "followers" && !v.follows(replyUserID) && replyUserID != v.UserID {
			return false
		}
		return true
	}

	return !(replyUserID != v.UserID && !isMe && replyUserID != authorID)
}

// renoteAllowed applies the withRenotes filter together with the
// visibility check on the renoted note's reply.
func (t *timelineChannel) renoteAllowed(note PackedNote) bool {
	if !isPureRenote(note) {
		return true
	}
	if !t.withRenotes {
		return false
	}
	renoted, ok := note["renote"].(map[string]any)
	if !ok {
		return true
	}
	reply, ok := replyOf(renoted)
	if !ok {
		return true
	}
	v := t.viewer()
	replyUserID, _ := reply["userId"].(string)
	return !(visibilityOf(reply) == "followers" && !v.follows(replyUserID) && replyUserID != v.UserID)
}

// filesAllowed applies the withFiles filter.
func (t *timelineChannel) filesAllowed(note PackedNote) bool {
	return !t.withFiles || hasFiles(note)
}

// hasChannelID reports whether the note belongs to a Misskey "channel".
// Cinderfish does not model channels yet, so this is always false.
func hasChannelID(note PackedNote) bool {
	v, ok := note["channelId"]
	return ok && v != nil
}

// isNoteMutedOrBlocked mirrors Misskey's filter. Cinderfish has no mute or
// block data yet, so it always allows the note; keeping the call site
// identical to upstream means the filter can be filled in without
// touching channel logic.
func (b *BaseChannel) isNoteMutedOrBlocked(PackedNote) bool { return false }

// emitNote sends a note to the client.
func (b *BaseChannel) emitNote(note PackedNote) {
	b.send("note", note)
}

// visibilityOf returns a packed entity's visibility string.
func visibilityOf(m map[string]any) string {
	v, _ := m["visibility"].(string)
	return v
}
