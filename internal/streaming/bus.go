// Package streaming implements MilnFish's Misskey-compatible Streaming
// API: the /streaming WebSocket endpoint, the per-connection channel
// registry and the event bus that carries note, reaction and
// notification events to subscribed clients.
//
// The protocol follows misskey-dev/misskey's server/api/stream package.
// See docs/research/2026-10-28-m4-streaming-source-calibration.md for
// the source-read record this package is calibrated against.
package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Subject names, matching the "channel" field Misskey publishes on its
// Redis bus. Keeping the same strings means a future Redis backend can
// be dropped in without touching channel code.
const (
	// SubjectNotes carries every packed note when it is created.
	SubjectNotes = "notesStream"
	// SubjectBroadcast carries instance-wide announcements.
	SubjectBroadcast = "broadcast"
)

// MainSubject returns the subject carrying a user's MainChannel events.
func MainSubject(userID string) string { return "mainStream:" + userID }

// NoteSubject returns the subject carrying updates to a single note.
func NoteSubject(noteID string) string { return "noteStream:" + noteID }

// Message is one event delivered on a subject.
//
// Type is the inner event type ("notification", "reacted", ...). A nil
// Type means the payload itself is the message, which is how Misskey
// publishes notesStream: Body then holds the packed note directly.
type Message struct {
	Type *string         `json:"type"`
	Body json.RawMessage `json:"body"`
}

// NewMessage builds a Message whose body is value encoded as JSON.
func NewMessage(typ string, value any) (Message, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return Message{}, err
	}
	t := typ
	return Message{Type: &t, Body: raw}, nil
}

// NullMessage builds a Message carrying an event with no body, such as
// "readAllNotifications".
func NullMessage(typ string) Message {
	t := typ
	return Message{Type: &t, Body: json.RawMessage("null")}
}

// Handler receives every published event.
type Handler func(subject string, msg Message)

// Bus fans events out to subscribers.
//
// Delivery is synchronous and must not block: subscribers hand events to
// a per-connection queue, so a slow client never stalls a publisher.
type Bus struct {
	mu        sync.RWMutex
	subs      map[int]Handler
	next      int
	publisher *Publisher
}

// NewBus returns an in-process event bus.
func NewBus() *Bus {
	return &Bus{subs: make(map[int]Handler)}
}

// Subscribe registers a handler and returns a function that removes it.
// The returned function is safe to call more than once.
func (b *Bus) Subscribe(h Handler) (cancel func()) {
	b.mu.Lock()
	id := b.next
	b.next++
	b.subs[id] = h
	b.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subs, id)
			b.mu.Unlock()
		})
	}
}

// SubscriberCount reports how many handlers are registered, for tests
// and diagnostics.
func (b *Bus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}

// SetPublisher installs the cross-process relay. Publish calls it after
// local delivery so other processes of the instance receive the event
// too. A nil publisher (the default) keeps the bus single-process.
func (b *Bus) SetPublisher(p *Publisher) {
	b.mu.Lock()
	b.publisher = p
	b.mu.Unlock()
}

// Publish delivers a message to every local subscriber and, when a
// publisher is installed, relays it to other processes.
func (b *Bus) Publish(subject string, msg Message) {
	b.mu.RLock()
	subs := make([]Handler, 0, len(b.subs))
	for _, h := range b.subs {
		subs = append(subs, h)
	}
	pub := b.publisher
	b.mu.RUnlock()

	for _, h := range subs {
		h(subject, msg)
	}

	if pub != nil {
		// Relay in the background: a slow database must never stall the
		// HTTP handler that produced the event.
		go func() {
			_ = pub.Notify(context.Background(), subject, msg)
		}()
	}
}

// PublishTyped publishes an event whose body is value encoded as JSON.
func (b *Bus) PublishTyped(subject, typ string, value any) {
	msg, err := NewMessage(typ, value)
	if err != nil {
		return
	}
	b.Publish(subject, msg)
}

// PublishNote announces a freshly created note on notesStream. The
// payload is the packed note itself, with no {type, body} envelope, as
// Misskey's publishNotesStream does.
func (b *Bus) PublishNote(value any) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	b.Publish(SubjectNotes, Message{Body: raw})
}

// PublishNoteStream announces an update to one note, wrapped in
// Misskey's {id,userId,visibility,visibleUserIds,body} envelope.
func (b *Bus) PublishNoteStream(noteID, authorID, visibility string, visibleUserIDs []string, typ string, value any) {
	if visibleUserIDs == nil {
		visibleUserIDs = []string{}
	}
	inner, err := json.Marshal(value)
	if err != nil {
		return
	}
	env, err := json.Marshal(map[string]any{
		"id":             noteID,
		"userId":         authorID,
		"visibility":     visibility,
		"visibleUserIds": visibleUserIDs,
		"body":           json.RawMessage(inner),
	})
	if err != nil {
		return
	}
	t := typ
	b.Publish(NoteSubject(noteID), Message{Type: &t, Body: env})
}

// pgNotifyChannel is the single PostgreSQL channel used to wake other
// processes of the same instance.
const pgNotifyChannel = "milnfish_stream"

// notifyPayloadLimit is PostgreSQL's NOTIFY payload cap.
const notifyPayloadLimit = 8000

// Publisher writes events to PostgreSQL so that every other process of
// the instance receives them through Listen.
//
// The body is sent inline when it fits in PostgreSQL's payload limit;
// otherwise only the subject is sent and remote processes treat the
// wakeup as a body-less notification. Bodies above the limit are rare
// (a packed note is normally a few hundred bytes) and only affect
// multi-process deployments.
type Publisher struct {
	pool *pgxpool.Pool
}

// NewPublisher returns a Publisher writing on the given pool.
func NewPublisher(pool *pgxpool.Pool) *Publisher { return &Publisher{pool: pool} }

type wireMessage struct {
	Subject string  `json:"s"`
	Type    *string `json:"t,omitempty"`
	Body    string  `json:"b,omitempty"`
}

// Notify publishes one event for other processes.
func (p *Publisher) Notify(ctx context.Context, subject string, msg Message) error {
	w := wireMessage{Subject: subject, Type: msg.Type}
	if json.Valid(msg.Body) {
		w.Body = string(msg.Body)
	}
	encoded, err := json.Marshal(w)
	if err != nil {
		return err
	}
	if len(encoded) > notifyPayloadLimit {
		encoded, err = json.Marshal(wireMessage{Subject: subject, Type: msg.Type})
		if err != nil {
			return err
		}
	}
	_, err = p.pool.Exec(ctx, "SELECT pg_notify($1, $2)", pgNotifyChannel, string(encoded))
	return err
}

// Listen subscribes to events published by other processes and feeds
// them into bus, blocking until ctx is cancelled.
//
// A dropped connection silently loses wakeups; Publish on this process
// is the source of truth for its own events, so a gap can only affect
// events produced by other processes. Run() therefore retries.
func Listen(ctx context.Context, pool *pgxpool.Pool, bus *Bus, logf func(string, ...any)) error {
	if logf == nil {
		logf = log.Printf
	}
	for {
		err := listenOnce(ctx, pool, bus, logf)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			logf("streaming: notification listener stopped: %v", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
		}
	}
}

func listenOnce(ctx context.Context, pool *pgxpool.Pool, bus *Bus, logf func(string, ...any)) error {
	// Acquire a single connection and keep it pinned: LISTEN state is
	// per session and a pooled connection must not be shared.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "LISTEN "+pgNotifyChannel); err != nil {
		return err
	}
	logf("streaming: listening for cross-process events on %q", pgNotifyChannel)

	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if errors.Is(err, context.Canceled) {
				return nil
			}
			logf("streaming: notification wait failed: %v", err)
			return err
		}
		var w wireMessage
		if err := json.Unmarshal([]byte(n.Payload), &w); err != nil {
			logf("streaming: malformed notification: %v", err)
			continue
		}
		// body is already known to be valid JSON: the sender only sets it
		// when json.Valid accepted it. An oversized payload arrives without
		// a body and is delivered as a null-body wakeup.
		body := json.RawMessage("null")
		if w.Body != "" && json.Valid([]byte(w.Body)) {
			body = json.RawMessage(w.Body)
		}
		bus.Publish(w.Subject, Message{Type: w.Type, Body: body})
	}
}
