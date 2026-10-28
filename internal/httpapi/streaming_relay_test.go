package httpapi_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"milnfish/internal/streaming"
)

// relayed is one event as a second process would see it.
type relayed struct {
	subject string
	msg     streaming.Message
}

// startRelay wires a sending bus to a second, independent bus whose
// PostgreSQL listener is running, which is how two processes of one
// instance see each other's events.
func startRelay(t *testing.T) (*streaming.Bus, <-chan relayed) {
	t.Helper()
	pool := testPool(t)

	receiver := streaming.NewBus()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = streaming.Listen(ctx, pool, receiver, t.Logf) }()

	got := make(chan relayed, 64)
	receiver.Subscribe(func(subject string, msg streaming.Message) {
		select {
		case got <- relayed{subject: subject, msg: msg}:
		default:
		}
	})

	sender := streaming.NewBus()
	sender.SetPublisher(streaming.NewPublisher(pool))

	// Give LISTEN a moment to attach before publishing.
	time.Sleep(300 * time.Millisecond)
	return sender, got
}

// waitForSubject returns the first relayed event for subject.
func waitForSubject(t *testing.T, got <-chan relayed, subject string) streaming.Message {
	t.Helper()
	deadline := time.After(15 * time.Second)
	for {
		select {
		case r := <-got:
			if r.subject == subject {
				return r.msg
			}
		case <-deadline:
			t.Fatalf("no relayed event for %q", subject)
		}
	}
}

// TestStreamingCrossProcessRelay proves the LISTEN/NOTIFY path: a bus
// with a PostgreSQL publisher relays to a second, independent bus.
func TestStreamingCrossProcessRelay(t *testing.T) {
	sender, got := startRelay(t)

	noteID := "note-cross-process"
	sender.PublishNoteStream(noteID, "alice", "public", nil, "reacted", map[string]any{
		"reaction": "👍",
		"userId":   "bob",
	})

	msg := waitForSubject(t, got, streaming.NoteSubject(noteID))
	if msg.Type == nil || *msg.Type != "reacted" {
		t.Fatalf("type = %v, want reacted", msg.Type)
	}
}

// TestStreamingCrossProcessRelayLargePayload proves an event larger than
// one NOTIFY payload is relayed with its content intact, instead of
// arriving as a body-less wakeup.
func TestStreamingCrossProcessRelayLargePayload(t *testing.T) {
	sender, got := startRelay(t)

	// Big enough that the notification has to be split: a packed note
	// with a long text plus its author is like this in practice.
	text := strings.Repeat("長い投稿", 4000)
	noteID := "note-cross-process-large"
	sender.PublishNoteStream(noteID, "alice", "public", nil, "reacted", map[string]any{
		"reaction": "👍",
		"text":     text,
	})

	msg := waitForSubject(t, got, streaming.NoteSubject(noteID))
	if len(msg.Body) <= streaming.NotifyPayloadLimit {
		t.Fatalf("relayed body is %d bytes, too small to have needed chunking", len(msg.Body))
	}

	var env struct {
		Body struct {
			Text string `json:"text"`
		} `json:"body"`
	}
	if err := json.Unmarshal(msg.Body, &env); err != nil {
		t.Fatalf("relayed body is not JSON: %v", err)
	}
	if env.Body.Text != text {
		t.Errorf("relayed text is %d bytes, want %d", len(env.Body.Text), len(text))
	}
}
