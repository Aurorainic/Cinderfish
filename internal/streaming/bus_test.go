package streaming

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
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

// TestWireMessagesChunkLargeBody checks that a body too large for one
// NOTIFY payload is relayed whole: the publisher splits it, the listener
// reassembles it, and the content is byte-identical.
func TestWireMessagesChunkLargeBody(t *testing.T) {
	text := strings.Repeat("長い投稿", 4000) // ~48 KiB of UTF-8
	msg, err := NewMessage("notification", map[string]any{"text": text, "index": 7})
	if err != nil {
		t.Fatal(err)
	}

	payloads, err := wireMessages("mainStream:alice", msg)
	if err != nil {
		t.Fatalf("wireMessages: %v", err)
	}
	if len(payloads) < 2 {
		t.Fatalf("payloads = %d, want the body split across several", len(payloads))
	}
	for i, p := range payloads {
		encoded, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("payload %d: %v", i, err)
		}
		if len(encoded) > NotifyPayloadLimit {
			t.Fatalf("payload %d is %d bytes, over the %d byte limit", i, len(encoded), NotifyPayloadLimit)
		}
	}

	// Reassemble in reverse order: chunks must not depend on delivery
	// order.
	r := newReassembler(t.Logf)
	var (
		subject string
		got     Message
		ready   bool
	)
	for i := len(payloads) - 1; i >= 0; i-- {
		subject, got, ready = r.consume(payloads[i])
	}
	if !ready {
		t.Fatal("the chunked event never became whole")
	}
	if subject != "mainStream:alice" {
		t.Errorf("subject = %q", subject)
	}
	if got.Type == nil || *got.Type != "notification" {
		t.Errorf("type = %v, want notification", got.Type)
	}
	if string(got.Body) != string(msg.Body) {
		t.Errorf("body differs: got %d bytes, want %d", len(got.Body), len(msg.Body))
	}
	if len(r.pending) != 0 {
		t.Errorf("pending assemblies = %d, want 0 after completion", len(r.pending))
	}
}

// TestWireMessagesRejectsOversizedBody checks a body beyond the relay
// limit is reported instead of being sent without its content.
func TestWireMessagesRejectsOversizedBody(t *testing.T) {
	msg, err := NewMessage("notification", strings.Repeat("x", maxRelayChars))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wireMessages("mainStream:alice", msg); err == nil {
		t.Fatal("an unrelayable body must produce an error, not a body-less wakeup")
	}
}

// TestReassemblerExpiresStaleChunks checks an event whose remaining
// chunks never arrived is dropped rather than held forever.
func TestReassemblerExpiresStaleChunks(t *testing.T) {
	r := newReassembler(t.Logf)
	first := wireMessage{Subject: "s", RelayID: "r1", Index: 0, Total: 2, Body: "e30="}
	if _, _, ready := r.consume(first); ready {
		t.Fatal("a single chunk cannot complete an event")
	}
	r.pending["r1"].first = time.Now().Add(-2 * relayAssemblyTTL)

	// The next chunk expires the stale assembly; it then starts a new one
	// that is still incomplete.
	if _, _, ready := r.consume(wireMessage{Subject: "s", RelayID: "r1", Index: 1, Total: 2, Body: "e30="}); ready {
		t.Fatal("the stale assembly must not be revived")
	}
	if got := len(r.pending); got != 1 {
		t.Errorf("pending assemblies = %d, want 1", got)
	}
	if got := r.pending["r1"].got; got != 1 {
		t.Errorf("chunks in the fresh assembly = %d, want 1", got)
	}
}

// TestReassemblerBoundsPendingAssemblies checks a sender cannot make the
// listener buffer unbounded state.
func TestReassemblerBoundsPendingAssemblies(t *testing.T) {
	r := newReassembler(t.Logf)
	for i := 0; i < maxRelayAssemblies; i++ {
		r.consume(wireMessage{Subject: "s", RelayID: fmt.Sprintf("r%d", i), Index: 0, Total: 2})
	}
	if got := len(r.pending); got != maxRelayAssemblies {
		t.Fatalf("pending assemblies = %d, want %d", got, maxRelayAssemblies)
	}
	r.consume(wireMessage{Subject: "s", RelayID: "overflow", Index: 0, Total: 2})
	if got := len(r.pending); got != maxRelayAssemblies {
		t.Errorf("pending assemblies = %d, want the cap to hold", got)
	}
}

// TestReassemblerDropsMalformedChunks checks a chunk that does not belong
// to any valid event is refused.
func TestReassemblerDropsMalformedChunks(t *testing.T) {
	r := newReassembler(t.Logf)
	cases := []wireMessage{
		{Subject: "s", RelayID: "", Index: 0, Total: 2},
		{Subject: "s", RelayID: "r", Index: 2, Total: 2},
		{Subject: "s", RelayID: "r", Index: -1, Total: 2},
		{Subject: "s", RelayID: "r", Index: 0, Total: maxRelayChunks + 1},
	}
	for _, w := range cases {
		if _, _, ready := r.consume(w); ready {
			t.Errorf("malformed chunk %+v completed an event", w)
		}
	}
	if got := len(r.pending); got != 0 {
		t.Errorf("pending assemblies = %d, want 0", got)
	}
}
