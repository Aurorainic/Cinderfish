package httpapi_test

import (
	"context"
	"testing"
	"time"

	"milnfish/internal/streaming"
)

// TestStreamingCrossProcessRelay proves the LISTEN/NOTIFY path: a bus
// with a PostgreSQL publisher relays to a second, independent bus whose
// listener is running, which is how two processes of one instance see
// each other's events.
func TestStreamingCrossProcessRelay(t *testing.T) {
	pool := testPool(t)

	// The receiving "process".
	receiver := streaming.NewBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- streaming.Listen(ctx, pool, receiver, t.Logf)
	}()

	// The sending "process".
	sender := streaming.NewBus()
	sender.SetPublisher(streaming.NewPublisher(pool))

	type received struct {
		subject string
		msg     streaming.Message
	}
	got := make(chan received, 4)
	receiver.Subscribe(func(subject string, msg streaming.Message) {
		select {
		case got <- received{subject: subject, msg: msg}:
		default:
		}
	})

	// Give LISTEN a moment to attach before publishing.
	time.Sleep(300 * time.Millisecond)

	noteID := "note-cross-process"
	sender.PublishNoteStream(noteID, "alice", "public", nil, "reacted", map[string]any{
		"reaction": "👍",
		"userId":   "bob",
	})

	deadline := time.After(5 * time.Second)
	for {
		select {
		case r := <-got:
			if r.subject != streaming.NoteSubject(noteID) {
				continue
			}
			if r.msg.Type == nil || *r.msg.Type != "reacted" {
				t.Fatalf("type = %v, want reacted", r.msg.Type)
			}
			return
		case <-deadline:
			t.Fatal("the relayed event never arrived")
		}
	}
}
