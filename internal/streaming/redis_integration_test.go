package streaming

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// realRedis opens a client for integration tests. Like the database tests,
// these skip when CINDERFISH_TEST_REDIS_URL is unset; once it is set, a
// connection failure is a real failure rather than a skip.
func realRedis(t *testing.T) *redis.Client {
	t.Helper()
	url := os.Getenv("CINDERFISH_TEST_REDIS_URL")
	if url == "" {
		t.Skip("CINDERFISH_TEST_REDIS_URL is not set")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse redis url: %v", err)
	}
	client := redis.NewClient(opts)
	t.Cleanup(func() { client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping redis %s: %v", opts.Addr, err)
	}
	return client
}

// testPrefix isolates concurrent runs sharing one Redis database.
func testPrefix(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("cinderfish-test-%d", time.Now().UnixNano())
}

func TestRedisRelayAgainstServer(t *testing.T) {
	url := os.Getenv("CINDERFISH_TEST_REDIS_URL")
	if url == "" {
		t.Skip("CINDERFISH_TEST_REDIS_URL is not set")
	}
	client := realRedis(t)
	prefix := testPrefix(t)

	// A second client stands in for the other process of the instance.
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse redis url: %v", err)
	}
	listener := redis.NewClient(opts)
	t.Cleanup(func() { listener.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := NewBus()
	bus.SetPublisher(NewRedisRelay(client, prefix))

	subscribed := make(chan struct{})
	go func() {
		_ = ListenRedis(ctx, listener, prefix, bus, waitForListener(t, t.Logf, subscribed))
	}()
	select {
	case <-subscribed:
	case <-time.After(3 * time.Second):
		t.Fatal("redis listener did not subscribe in time")
	}

	events := make(chan Message, 4)
	bus.Subscribe(func(_ string, msg Message) { events <- msg })

	// No PostgreSQL 8000 byte NOTIFY cap on this path.
	body := map[string]any{"text": string(make([]byte, 20000))}
	bus.PublishTyped("mainStream:u1", "note", body)

	select {
	case msg := <-events:
		if msg.Type == nil || *msg.Type != "note" {
			t.Fatalf("type = %v, want note", msg.Type)
		}
		if len(msg.Body) < 20000 {
			t.Errorf("body length = %d, want >= 20000 (chunked on redis?)", len(msg.Body))
		}
	case <-time.After(3 * time.Second):
		t.Fatal("event did not arrive through redis")
	}
}
