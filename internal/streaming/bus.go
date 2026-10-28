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
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
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

// Relay forwards one event to the other processes of the instance.
type Relay interface {
	// Notify publishes the event, tagging it with the origin process so
	// the sender does not receive its own event back.
	Notify(ctx context.Context, origin, subject string, msg Message) error
}

// Bus fans events out to subscribers.
//
// Delivery is synchronous and must not block: subscribers hand events to
// a per-connection queue, so a slow client never stalls a publisher.
type Bus struct {
	// id identifies this bus among the processes sharing the relay. It
	// lets the listener recognise -- and drop -- the echo of an event
	// this process published itself.
	id string

	mu    sync.RWMutex
	subs  map[int]Handler
	next  int
	relay Relay
}

// NewBus returns an in-process event bus.
func NewBus() *Bus {
	return &Bus{id: newID(), subs: make(map[int]Handler)}
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
func (b *Bus) SetPublisher(r Relay) {
	b.mu.Lock()
	b.relay = r
	b.mu.Unlock()
}

// Publish delivers a message to every local subscriber and, when a relay
// is installed, forwards it to the other processes.
func (b *Bus) Publish(subject string, msg Message) {
	subs, relay := b.snapshot()
	for _, h := range subs {
		h(subject, msg)
	}

	if relay != nil {
		// Relay in the background: a slow database must never stall the
		// HTTP handler that produced the event. A failure is logged: the
		// event then stays visible only on this process, which must not
		// happen quietly.
		origin := b.id
		go func() {
			if err := relay.Notify(context.Background(), origin, subject, msg); err != nil {
				log.Printf("streaming: relaying %s to other processes failed: %v", subject, err)
			}
		}()
	}
}

// deliver publishes a message to local subscribers only.
//
// Listen uses it for events that arrived from another process: that
// process already relayed the event, and relaying it again here would feed
// the listener with its own echo, forever.
func (b *Bus) deliver(subject string, msg Message) {
	subs, _ := b.snapshot()
	for _, h := range subs {
		h(subject, msg)
	}
}

// snapshot copies the subscriber list and the relay so the fanout runs
// without holding the lock.
func (b *Bus) snapshot() ([]Handler, Relay) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	subs := make([]Handler, 0, len(b.subs))
	for _, h := range b.subs {
		subs = append(subs, h)
	}
	return subs, b.relay
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

// NotifyPayloadLimit is PostgreSQL's NOTIFY payload cap in bytes.
// pg_notify rejects anything larger outright.
const NotifyPayloadLimit = 8000

// relayChunkChars is how many base64 characters one chunk carries.
// base64 never needs JSON escaping, so a chunked notification stays well
// under NotifyPayloadLimit.
const relayChunkChars = 4000

// maxRelayChunks bounds one relayed body. A larger body is reported as an
// error instead of being relayed without its content.
const maxRelayChunks = 128

// maxRelayChars is the base64 length one event may reach.
const maxRelayChars = maxRelayChunks * relayChunkChars

// relayAssemblyTTL is how long a partially received event is kept before
// the listener gives up on the chunks that never arrived.
const relayAssemblyTTL = 30 * time.Second

// maxRelayAssemblies bounds the partially received events one listener
// holds at a time.
const maxRelayAssemblies = 64

// Publisher writes events to PostgreSQL so that every other process of
// the instance receives them through Listen.
//
// An event that does not fit in one NOTIFY payload is base64 encoded and
// split across several notifications, so a large packed note or
// notification is relayed whole. Misskey has no such limit because Redis
// pub/sub accepts arbitrary payloads; PostgreSQL's 8000 byte cap is ours
// to work around.
type Publisher struct {
	pool *pgxpool.Pool
}

// NewPublisher returns a Publisher writing on the given pool.
func NewPublisher(pool *pgxpool.Pool) *Publisher { return &Publisher{pool: pool} }

// wireMessage is one NOTIFY payload. A chunked event repeats subject and
// type in every chunk and numbers them with index and total.
type wireMessage struct {
	Subject string  `json:"s"`
	Type    *string `json:"t,omitempty"`
	Body    string  `json:"b,omitempty"`
	// Origin identifies the bus that published the event, so that bus can
	// drop the echo of its own notification.
	Origin string `json:"o,omitempty"`
	// RelayID identifies the event a chunk belongs to.
	RelayID string `json:"i,omitempty"`
	// Index is the 0-based position of this chunk.
	Index int `json:"n,omitempty"`
	// Total is the number of chunks the body was split into. Zero means
	// the body was not split.
	Total int `json:"k,omitempty"`
}

// Notify publishes one event for other processes.
func (p *Publisher) Notify(ctx context.Context, origin, subject string, msg Message) error {
	payloads, err := wireMessages(origin, subject, msg)
	if err != nil {
		return err
	}
	for _, w := range payloads {
		encoded, err := json.Marshal(w)
		if err != nil {
			return err
		}
		if len(encoded) > NotifyPayloadLimit {
			// wireMessages sizes the payloads to fit, so this is a bug in
			// this package rather than a client problem. Fail loudly
			// instead of letting pg_notify drop the event.
			return fmt.Errorf("streaming: notification of %d bytes exceeds the %d byte limit", len(encoded), NotifyPayloadLimit)
		}
		if _, err := p.pool.Exec(ctx, "SELECT pg_notify($1, $2)", pgNotifyChannel, string(encoded)); err != nil {
			return err
		}
	}
	return nil
}

// wireMessages encodes one event as the sequence of NOTIFY payloads that
// carries it intact: one payload when the body fits, several when it does
// not. It returns an error rather than a body-less placeholder when the
// body is too large to relay at all.
func wireMessages(origin, subject string, msg Message) ([]wireMessage, error) {
	w := wireMessage{Subject: subject, Type: msg.Type, Origin: origin}
	if json.Valid(msg.Body) {
		w.Body = string(msg.Body)
	}
	encoded, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}
	if len(encoded) <= NotifyPayloadLimit {
		return []wireMessage{w}, nil
	}

	chunks, err := splitBody(w.Body)
	if err != nil {
		return nil, err
	}
	id := newID()
	out := make([]wireMessage, len(chunks))
	for i, chunk := range chunks {
		out[i] = wireMessage{
			Subject: subject,
			Type:    msg.Type,
			Origin:  origin,
			RelayID: id,
			Index:   i,
			Total:   len(chunks),
			Body:    chunk,
		}
	}
	return out, nil
}

// splitBody cuts a JSON body into base64 chunks that each fit in one
// NOTIFY payload. base64 avoids splitting a multi-byte character in half
// and needs no JSON escaping, so a chunk's size is predictable.
func splitBody(body string) ([]string, error) {
	if !json.Valid([]byte(body)) {
		return nil, errors.New("streaming: refusing to relay a body that is not valid JSON")
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	if len(encoded) > maxRelayChars {
		return nil, fmt.Errorf("streaming: event body of %d bytes exceeds the %d byte relay limit",
			len(body), maxRelayChars/4*3)
	}
	chunks := make([]string, 0, (len(encoded)+relayChunkChars-1)/relayChunkChars)
	for len(encoded) > 0 {
		n := min(relayChunkChars, len(encoded))
		chunks = append(chunks, encoded[:n])
		encoded = encoded[n:]
	}
	return chunks, nil
}

// newID returns a random 8 byte identifier: the origin of a relayed event
// or the grouping key of its chunks.
func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Only uniqueness matters here, and a timestamp still
		// distinguishes two processes well enough.
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b[:])
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

	pending := newReassembler(logf)
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
		if w.Origin == bus.id {
			// This process published the event and has already delivered
			// it locally; the notification is its own echo.
			continue
		}
		subject, msg, ready := pending.consume(w)
		if !ready {
			continue
		}
		// Local fanout only: the producing process is responsible for
		// relaying, so relaying again here would loop without end.
		bus.deliver(subject, msg)
	}
}

// reassembler rebuilds events that arrive as several NOTIFY payloads.
//
// Only the listener goroutine feeds it, so it needs no locking.
type reassembler struct {
	logf    func(string, ...any)
	pending map[string]*assembly
}

// assembly is a partially received chunked event.
type assembly struct {
	subject string
	typ     *string
	chunks  []string
	got     int
	first   time.Time
}

func newReassembler(logf func(string, ...any)) *reassembler {
	return &reassembler{logf: logf, pending: make(map[string]*assembly)}
}

// consume takes one notification. It reports ready=true only when a whole
// event is available, which for a chunked event means the last chunk.
func (r *reassembler) consume(w wireMessage) (subject string, msg Message, ready bool) {
	if w.Total == 0 {
		return w.Subject, Message{Type: w.Type, Body: decodeBody(w.Body)}, true
	}

	if w.RelayID == "" || w.Total < 0 || w.Total > maxRelayChunks || w.Index < 0 || w.Index >= w.Total {
		r.logf("streaming: discarding malformed chunked event on %q", w.Subject)
		return "", Message{}, false
	}

	r.expire()

	a, ok := r.pending[w.RelayID]
	if !ok {
		if len(r.pending) >= maxRelayAssemblies {
			r.logf("streaming: discarding a chunked event on %q: %d events are already pending",
				w.Subject, len(r.pending))
			return "", Message{}, false
		}
		a = &assembly{subject: w.Subject, typ: w.Type, chunks: make([]string, w.Total), first: time.Now()}
		r.pending[w.RelayID] = a
	}
	if len(a.chunks) != w.Total {
		r.logf("streaming: discarding a chunked event on %q: chunk count changed", w.Subject)
		delete(r.pending, w.RelayID)
		return "", Message{}, false
	}
	if a.chunks[w.Index] != "" {
		// A repeated chunk of an event still being assembled. NOTIFY does
		// not retry, so this is noise rather than an error.
		return "", Message{}, false
	}
	a.chunks[w.Index] = w.Body
	a.got++
	if a.got < len(a.chunks) {
		return "", Message{}, false
	}

	delete(r.pending, w.RelayID)
	raw, err := base64.StdEncoding.DecodeString(strings.Join(a.chunks, ""))
	if err != nil || !json.Valid(raw) {
		r.logf("streaming: discarding a chunked event on %q: undecodable body", a.subject)
		return "", Message{}, false
	}
	return a.subject, Message{Type: a.typ, Body: raw}, true
}

// expire drops events whose missing chunks never arrived, and says so: a
// lost chunk loses one event, which must not be silent.
func (r *reassembler) expire() {
	if len(r.pending) == 0 {
		return
	}
	cutoff := time.Now().Add(-relayAssemblyTTL)
	for id, a := range r.pending {
		if a.first.Before(cutoff) {
			r.logf("streaming: dropping a chunked event on %q: %d of %d chunks arrived",
				a.subject, a.got, len(a.chunks))
			delete(r.pending, id)
		}
	}
}

// decodeBody keeps a notification's body, defaulting to null when the
// sender carried none.
func decodeBody(body string) json.RawMessage {
	if body != "" && json.Valid([]byte(body)) {
		return json.RawMessage(body)
	}
	return json.RawMessage("null")
}
