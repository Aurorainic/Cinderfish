package streaming

import (
	"encoding/json"
	"time"
)

// MessagingStreamSubject returns the subject carrying one 1:1
// conversation's events, in upstream's "<me>-<other>" pair shape with the
// smaller id first so both directions share one subject.
func MessagingStreamSubject(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return "messagingStream:" + a + "-" + b
}

// MessagingIndexSubject returns the subject carrying one user's
// conversation-list events.
func MessagingIndexSubject(userID string) string { return "messagingIndexStream:" + userID }

// newMessagingChannel builds the messaging channel.
func newMessagingChannel(c *Connection, id string) Channel {
	return &messagingChannel{timelineChannel: timelineChannel{BaseChannel: BaseChannel{id: id, conn: c}}}
}

// messagingChannel carries one conversation: message, read, deleted, and
// the five-second typers aggregate built from the typing events the
// httpapi frame hook publishes (their bodies are packed UserLite, which
// this channel only stores and replays). Group params are a recorded
// deferral and reject the silent way upstream rejects a bad param.
type messagingChannel struct {
	timelineChannel
	partner string

	typers map[string]typerEntry
}

// typerEntry keeps the packed user and when typing was last seen.
type typerEntry struct {
	body json.RawMessage
	at   time.Time
}

func (m *messagingChannel) Name() string { return "messaging" }

func (m *messagingChannel) Init(params map[string]json.RawMessage) bool {
	v := m.conn.viewerSnapshot()
	if v.UserID == "" {
		return false
	}
	raw, ok := params["otherparty"]
	if !ok || len(raw) == 0 {
		return false
	}
	var partner string
	if err := json.Unmarshal(raw, &partner); err != nil || partner == "" || partner == v.UserID {
		return false
	}
	m.partner = partner
	m.typers = map[string]typerEntry{}

	// Upstream emits typers on a five-second tick; the timer follows the
	// channel's lifecycle through the cleanup list.
	stop := make(chan struct{})
	m.cleanup = append(m.cleanup, func() { close(stop) })
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				m.deliverTypers()
			}
		}
	}()

	m.subscribeToBus(MessagingStreamSubject(v.UserID, m.partner), m.onData)
	return true
}

// deliverTypers emits the users that typed within the last five seconds
// and forgets stale entries.
func (m *messagingChannel) deliverTypers() {
	current := time.Now().Add(-5 * time.Second)
	users := make([]json.RawMessage, 0, len(m.typers))
	for id, entry := range m.typers {
		if entry.at.After(current) {
			users = append(users, entry.body)
		} else {
			delete(m.typers, id)
		}
	}
	if len(users) == 0 {
		return
	}
	m.conn.ws.send("channel", map[string]any{"id": m.id, "type": "typers", "body": users})
}

func (m *messagingChannel) onData(_ string, msg Message) {
	if msg.Type == nil {
		return
	}
	switch *msg.Type {
	case "message", "read", "deleted":
		m.conn.ws.send("channel", map[string]any{"id": m.id, "type": *msg.Type, "body": msg.Body})
	case "typing":
		// The body is the packed typer; the id keys the freshness map.
		var user struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(msg.Body, &user); err != nil || user.ID == "" {
			return
		}
		m.typers[user.ID] = typerEntry{
			body: append(json.RawMessage(nil), msg.Body...),
			at:   time.Now(),
		}
	}
}

// HandleNote implements Channel; messaging does not consume notesStream.
func (m *messagingChannel) HandleNote(PackedNote) {}

// HandleNoteUpdate implements Channel: conversation events ride the
// messaging subject, not per-note subscriptions.
func (m *messagingChannel) HandleNoteUpdate(string, json.RawMessage, NoteMeta) {}

// newMessagingIndexChannel builds the messagingIndex channel: the
// conversation-list feed, shareable like upstream's flag says.
func newMessagingIndexChannel(c *Connection, id string) Channel {
	return &messagingIndexChannel{BaseChannel: BaseChannel{id: id, conn: c}}
}

// messagingIndexChannel forwards message/read events for the viewer's
// conversation list.
type messagingIndexChannel struct {
	BaseChannel
}

func (m *messagingIndexChannel) Name() string { return "messagingIndex" }

func (m *messagingIndexChannel) Init(params map[string]json.RawMessage) bool {
	v := m.conn.viewerSnapshot()
	if v.UserID == "" {
		return false
	}
	m.subscribeToBus(MessagingIndexSubject(v.UserID), m.onData)
	return true
}

func (m *messagingIndexChannel) onData(_ string, msg Message) {
	if msg.Type == nil {
		return
	}
	switch *msg.Type {
	case "message", "read":
		m.conn.ws.send("channel", map[string]any{"id": m.id, "type": *msg.Type, "body": msg.Body})
	}
}

// HandleNote implements Channel; messagingIndex does not consume notesStream.
func (m *messagingIndexChannel) HandleNote(PackedNote) {}

// HandleNoteUpdate implements Channel: message events ride the
// messagingIndex subject, not per-note subscriptions.
func (m *messagingIndexChannel) HandleNoteUpdate(string, json.RawMessage, NoteMeta) {}
