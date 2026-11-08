package streaming

import (
	"context"
	"encoding/json"
)

// AntennaLoader checks antenna ownership for the channel's init check,
// the same userId-scoped lookup upstream's init does.
type AntennaLoader interface {
	// AntennaOwned reports whether antennaID exists and belongs to viewerID.
	AntennaOwned(ctx context.Context, antennaID, viewerID string) (bool, error)
}

// newAntennaChannel builds the antenna channel.
func newAntennaChannel(c *Connection, id string) Channel {
	return &antennaChannel{BaseChannel: BaseChannel{id: id, conn: c}}
}

// antennaChannel forwards one antenna's "note" events, the packed note
// the ingest path published on the antenna's subject. Only the owner can
// watch an antenna, matching upstream's init existence check.
type antennaChannel struct {
	BaseChannel
	antennaID string
}

func (a *antennaChannel) Name() string { return "antenna" }

func (a *antennaChannel) Init(params map[string]json.RawMessage) bool {
	raw, ok := params["antennaId"]
	if !ok || len(raw) == 0 {
		return false
	}
	var antennaID string
	if err := json.Unmarshal(raw, &antennaID); err != nil || antennaID == "" {
		return false
	}
	if a.conn.antennaLoader == nil {
		return false
	}
	owned, err := a.conn.antennaLoader.AntennaOwned(a.conn.ctx, antennaID, a.viewer().UserID)
	if err != nil || !owned {
		return false
	}
	a.antennaID = antennaID

	a.subscribeToBus(AntennaSubject(antennaID), a.onData)
	return true
}

func (a *antennaChannel) onData(_ string, msg Message) {
	if msg.Type == nil {
		return
	}
	// Upstream forwards "note" fully packed and passes every other type
	// through unchanged; this port only emits "note".
	switch *msg.Type {
	case "note":
		var note PackedNote
		if err := json.Unmarshal(msg.Body, &note); err != nil {
			return
		}
		v := a.viewer()
		if !isNoteVisibleForMe(note, v) {
			return
		}
		if a.isNoteMutedOrBlocked(note) {
			return
		}
		a.emitNote(note)
	}
}

// HandleNote implements Channel; antenna does not consume notesStream.
func (a *antennaChannel) HandleNote(PackedNote) {}

// HandleNoteUpdate implements Channel: antenna events ride the antenna
// subject, not per-note subscriptions.
func (a *antennaChannel) HandleNoteUpdate(string, json.RawMessage, NoteMeta) {}
