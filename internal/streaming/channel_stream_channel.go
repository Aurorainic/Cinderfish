package streaming

import (
	"encoding/json"
)

// newChannelStreamChannel builds the streaming "channel" channel (the
// per-board subscription; the spec registry entry is also named "channel",
// so the constructor name avoids the clash).
func newChannelStreamChannel(c *Connection, id string) Channel {
	return &channelStreamChannel{timelineChannel: timelineChannel{BaseChannel: BaseChannel{id: id, conn: c}}}
}

// channelStreamChannel carries the notes of one channel board. Upstream
// subscribes the shared notesStream and filters on channels/channel.ts,
// performing no existence check: subscribing to an unknown id just never
// delivers, and that behaviour is reproduced here.
type channelStreamChannel struct {
	timelineChannel
	channelID string
}

func (s *channelStreamChannel) Name() string { return "channel" }

func (s *channelStreamChannel) Init(params map[string]json.RawMessage) bool {
	raw, ok := params["channelId"]
	if !ok || len(raw) == 0 {
		return false
	}
	var channelID string
	if err := json.Unmarshal(raw, &channelID); err != nil || channelID == "" {
		return false
	}
	s.channelID = channelID

	// channels/timeline always includes pure renotes, and the endpoint has
	// no withFiles/withReplies params.
	s.withRenotes = true
	s.withReplies = true
	s.withFiles = false
	s.filter = s.onNote
	return true
}

func (s *channelStreamChannel) onNote(note PackedNote) {
	if !s.filesAllowed(note) {
		return
	}
	if noteChannelID(note) != s.channelID {
		return
	}
	// A channel note is always public and local, so the visibility check
	// reduces to the signed-viewer requirement upstream models through
	// requireSigninToViewContents, not modelled here.
	if !isNoteVisibleForMe(note, s.viewer()) {
		return
	}
	if s.isNoteMutedOrBlocked(note) {
		return
	}
	s.emitNote(note)
}

// noteChannelID reads a packed note's channelId.
func noteChannelID(note PackedNote) string {
	id, _ := note["channelId"].(string)
	return id
}
