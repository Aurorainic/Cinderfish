package httpapi_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/streaming"
	"github.com/redis/go-redis/v9"
)

// startRedisRelay wires two buses through one real Redis server: the sender
// publishes with NewRedisRelay and the receiver runs ListenRedis, which is
// exactly how main.go connects two processes of one instance.
func startRedisRelay(t *testing.T) (*streaming.Bus, <-chan relayed) {
	t.Helper()
	url := os.Getenv("CINDERFISH_TEST_REDIS_URL")
	if url == "" {
		t.Skip("CINDERFISH_TEST_REDIS_URL is not set")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse redis url: %v", err)
	}
	publisher := redis.NewClient(opts)
	listener := redis.NewClient(opts)
	t.Cleanup(func() {
		publisher.Close()
		listener.Close()
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := publisher.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping redis %s: %v", opts.Addr, err)
	}

	prefix := fmt.Sprintf("cinderfish-test-%d", time.Now().UnixNano())

	receiver := streaming.NewBus()
	subscribed := make(chan struct{})
	var once sync.Once
	go func() {
		_ = streaming.ListenRedis(ctx, listener, prefix, receiver, func(format string, args ...any) {
			t.Logf(format, args...)
			once.Do(func() { close(subscribed) })
		})
	}()
	select {
	case <-subscribed:
	case <-time.After(3 * time.Second):
		t.Fatal("redis listener did not subscribe in time")
	}

	got := make(chan relayed, 64)
	receiver.Subscribe(func(subject string, msg streaming.Message) {
		select {
		case got <- relayed{subject: subject, msg: msg}:
		default:
		}
	})

	sender := streaming.NewBus()
	sender.SetPublisher(streaming.NewRedisRelay(publisher, prefix))
	return sender, got
}

// TestStreamingCrossProcessRelayRedis proves the Redis path used when
// [redis] enabled = true: a note event published in one process reaches a
// bus listening in another, without PostgreSQL's NOTIFY payload cap.
func TestStreamingCrossProcessRelayRedis(t *testing.T) {
	sender, got := startRedisRelay(t)

	noteID := "note-cross-process-redis"
	sender.PublishNoteStream(noteID, "alice", "public", nil, "reacted", map[string]any{
		"reaction": "👍",
		"userId":   "bob",
	})

	msg := waitForSubject(t, got, streaming.NoteSubject(noteID))
	if msg.Type == nil || *msg.Type != "reacted" {
		t.Fatalf("type = %v, want reacted", msg.Type)
	}
}
