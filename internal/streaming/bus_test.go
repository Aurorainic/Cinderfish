package streaming

import (
	"encoding/json"
	"testing"
)

// TestNotesStreamReachesTimeline exercises the full bus path: publish a
// note on the bus and confirm a connected timeline channel receives it.
func TestNotesStreamReachesTimeline(t *testing.T) {
	h := newTestConnection(t, Viewer{})
	ch := newChannel(h, "localTimeline", "lt", nil)
	h.conn.channels["lt"] = ch

	// Mirror Connection.serve's notesStream subscription.
	unsub := h.conn.bus.Subscribe(func(subject string, msg Message) {
		if subject == SubjectNotes {
			h.conn.dispatchNote(msg)
		}
	})
	defer unsub()

	h.conn.bus.PublishNote(noteFor("n1", "alice", "public"))

	if got, want := h.ids(), []string{"n1"}; !equalStrings(got, want) {
		t.Errorf("notes = %v, want %v", got, want)
	}
}

// TestPublishNoteShape checks notesStream carries the bare note, with no
// {type, body} envelope.
func TestPublishNoteShape(t *testing.T) {
	bus := NewBus()
	var got Message
	bus.Subscribe(func(subject string, msg Message) {
		if subject == SubjectNotes {
			got = msg
		}
	})

	note := noteFor("n1", "alice", "public")
	bus.PublishNote(note)

	if got.Type != nil {
		t.Errorf("notesStream message must not carry a type, got %v", *got.Type)
	}
	var decoded map[string]any
	if err := json.Unmarshal(got.Body, &decoded); err != nil {
		t.Fatalf("body is not a JSON object: %v", err)
	}
	if decoded["id"] != "n1" {
		t.Errorf("body id = %v, want n1", decoded["id"])
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
