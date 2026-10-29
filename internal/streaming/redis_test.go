package streaming

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func redisTestClient(t *testing.T, mr *miniredis.Miniredis) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	return client
}

// waitForListener blocks until ListenRedis has subscribed, using its log
// line as the signal instead of sleeping.
func waitForListener(t *testing.T, logf func(string, ...any), subscribed chan struct{}) func(string, ...any) {
	t.Helper()
	return func(format string, args ...any) {
		logf(format, args...)
		select {
		case <-subscribed:
		default:
			close(subscribed)
		}
	}
}

func TestRedisRelayDeliversAcrossBuses(t *testing.T) {
	mr := miniredis.RunT(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientA := redisTestClient(t, mr)
	clientB := redisTestClient(t, mr)
	busA := NewBus()
	busB := NewBus()
	busA.SetPublisher(NewRedisRelay(clientA, "test"))

	subscribed := make(chan struct{})
	go func() {
		_ = ListenRedis(ctx, clientB, "test", busB, waitForListener(t, t.Logf, subscribed))
	}()

	select {
	case <-subscribed:
	case <-time.After(2 * time.Second):
		t.Fatal("redis listener did not subscribe in time")
	}

	events := make(chan Message, 4)
	busB.Subscribe(func(_ string, msg Message) { events <- msg })

	busA.PublishTyped("mainStream:u1", "notification", map[string]any{"id": "n1"})

	select {
	case msg := <-events:
		if msg.Type == nil || *msg.Type != "notification" {
			t.Fatalf("type = %v, want notification", msg.Type)
		}
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			t.Fatalf("body: %v", err)
		}
		if body["id"] != "n1" {
			t.Errorf("body = %v", body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("event did not arrive through redis")
	}
}

func TestRedisListenerDropsItsOwnEcho(t *testing.T) {
	mr := miniredis.RunT(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := redisTestClient(t, mr)
	listener := redisTestClient(t, mr)

	bus := NewBus()
	bus.SetPublisher(NewRedisRelay(publisher, "echo"))

	subscribed := make(chan struct{})
	go func() {
		_ = ListenRedis(ctx, listener, "echo", bus, waitForListener(t, t.Logf, subscribed))
	}()
	select {
	case <-subscribed:
	case <-time.After(2 * time.Second):
		t.Fatal("redis listener did not subscribe in time")
	}

	var mu = make(chan Message, 8)
	bus.Subscribe(func(_ string, msg Message) { mu <- msg })

	bus.PublishTyped("mainStream:u1", "follow", map[string]any{"id": "u2"})

	select {
	case <-mu:
	case <-time.After(time.Second):
		t.Fatal("local delivery missing")
	}
	select {
	case <-mu:
		t.Fatal("the listener re-delivered the publisher's own event")
	case <-time.After(300 * time.Millisecond):
	}
}

func TestRedisRelayLargeBodyIsNotChunked(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redisTestClient(t, mr)
	relay := NewRedisRelay(client, "big")

	if relay.channel != "big:stream" {
		t.Fatalf("channel = %q", relay.channel)
	}

	// A body well past PostgreSQL's 8000 byte NOTIFY cap still travels as
	// one Redis message.
	msg, err := NewMessage("notification", map[string]any{"text": string(make([]byte, 20000))})
	if err != nil {
		t.Fatal(err)
	}
	sub := client.Subscribe(context.Background(), relay.channel)
	defer sub.Close()
	if _, err := sub.Receive(context.Background()); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if err := relay.Notify(context.Background(), "origin", "mainStream:u1", msg); err != nil {
		t.Fatalf("notify: %v", err)
	}
	received, err := sub.ReceiveMessage(context.Background())
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	var w wireMessage
	if err := json.Unmarshal([]byte(received.Payload), &w); err != nil {
		t.Fatalf("payload: %v", err)
	}
	if w.Total != 0 || w.RelayID != "" {
		t.Errorf("redis event was chunked: %+v", w)
	}
	if len(w.Body) < 20000 {
		t.Errorf("body length = %d, want >= 20000", len(w.Body))
	}
}
