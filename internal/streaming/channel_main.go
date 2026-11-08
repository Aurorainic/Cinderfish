package streaming

import (
	"encoding/json"
)

// channelSpec describes a channel the server can instantiate.
type channelSpec struct {
	Name              string
	RequireCredential bool
	Shareable         bool
	New               func(c *Connection, id string) Channel
}

// channelSpecs is the registry of supported channels. Channels Misskey
// also defines but Cinderfish does not implement yet (clip, hashtag,
// antenna, drive, serverStats, admin, chat*, reversi*, ...) are
// deliberately absent: connecting to them is silently ignored, which
// is what upstream clients observe when the constructor throws.
var channelSpecs = map[string]channelSpec{
	"main":           {Name: "main", RequireCredential: true, Shareable: true, New: newMainChannel},
	"homeTimeline":   {Name: "homeTimeline", RequireCredential: true, New: newHomeTimelineChannel},
	"localTimeline":  {Name: "localTimeline", New: newLocalTimelineChannel},
	"hybridTimeline": {Name: "hybridTimeline", RequireCredential: true, New: newHybridTimelineChannel},
	"globalTimeline": {Name: "globalTimeline", New: newGlobalTimelineChannel},
	// Upstream declares requireCredential = false on userList but its
	// init then dereferences the viewer unconditionally, so an anonymous
	// subscriber crashes; requiring the credential here is the intent.
	"userList": {Name: "userList", RequireCredential: true, New: newUserListChannel},
}

// newMainChannel builds the main channel.
func newMainChannel(c *Connection, id string) Channel {
	return &mainChannel{BaseChannel: BaseChannel{id: id, conn: c}}
}

// mainChannel delivers a user's personal events: notifications,
// mentions, replies, renotes and account updates.
type mainChannel struct {
	BaseChannel
}

func (m *mainChannel) Name() string { return "main" }

func (m *mainChannel) Init(params map[string]json.RawMessage) bool {
	v := m.conn.viewerSnapshot()
	if v.UserID == "" {
		return false
	}
	m.subscribeToBus(MainSubject(v.UserID), m.onData)
	return true
}

func (m *mainChannel) onData(_ string, msg Message) {
	typ := ""
	if msg.Type != nil {
		typ = *msg.Type
	}

	switch typ {
	case "mention":
		var note PackedNote
		if err := json.Unmarshal(msg.Body, &note); err != nil {
			return
		}
		if !isNoteVisibleForMe(note, m.viewer()) {
			return
		}
		m.send(typ, note)
		return
	}

	// Every other event passes through unchanged.
	m.sendRaw(typ, msg.Body)
}

// sendRaw emits an event whose body is already encoded JSON.
func (m *mainChannel) sendRaw(typ string, body json.RawMessage) {
	m.conn.ws.send("channel", map[string]any{"id": m.id, "type": typ, "body": body})
}

// HandleNote implements Channel; main does not consume notesStream.
func (m *mainChannel) HandleNote(PackedNote) {}

// HandleNoteUpdate implements Channel; main does not consume noteStream.
func (m *mainChannel) HandleNoteUpdate(string, json.RawMessage, NoteMeta) {}
