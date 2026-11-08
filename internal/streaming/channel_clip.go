package streaming

import (
	"context"
	"encoding/json"
)

// ClipLoader reports a clip's visibility for the channel's init check.
type ClipLoader interface {
	// ClipVisible reports whether the clip exists and is readable by the
	// viewer — its owner, or anyone when the clip is public.
	ClipVisible(ctx context.Context, clipID, viewerID string) (bool, error)
}

// Connection field: the clip loader installed by the Server.
// (See connection.go.)

// newClipChannel builds the clip channel.
func newClipChannel(c *Connection, id string) Channel {
	return &clipChannel{BaseChannel: BaseChannel{id: id, conn: c}}
}

// clipChannel forwards one clip's update events: "updated" when its note
// membership or metadata changes, "deleted" when it is removed. The body
// is null for both, matching publishClipStream's emit shape.
type clipChannel struct {
	BaseChannel
	clipID string
}

func (c *clipChannel) Name() string { return "clip" }

func (c *clipChannel) Init(params map[string]json.RawMessage) bool {
	raw, ok := params["clipId"]
	if !ok || len(raw) == 0 {
		return false
	}
	var clipID string
	if err := json.Unmarshal(raw, &clipID); err != nil || clipID == "" {
		return false
	}
	if c.conn.clipLoader == nil {
		return false
	}
	visible, err := c.conn.clipLoader.ClipVisible(c.conn.ctx, clipID, c.viewer().UserID)
	if err != nil || !visible {
		return false
	}
	c.clipID = clipID

	c.subscribeToBus(ClipSubject(clipID), c.onData)
	return true
}

func (c *clipChannel) onData(_ string, msg Message) {
	if msg.Type == nil {
		return
	}
	switch *msg.Type {
	case "updated", "deleted":
		c.send(*msg.Type, nil)
	}
}

// HandleNote implements Channel; clip does not consume notesStream.
func (c *clipChannel) HandleNote(PackedNote) {}

// HandleNoteUpdate implements Channel: clip events ride the clip subject,
// not per-note subscriptions.
func (c *clipChannel) HandleNoteUpdate(string, json.RawMessage, NoteMeta) {}
