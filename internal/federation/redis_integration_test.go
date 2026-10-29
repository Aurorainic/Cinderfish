package federation

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// realRedisPair opens two clients against CINDERFISH_TEST_REDIS_URL, standing
// in for two processes sharing one Redis. It skips when the variable is unset.
func realRedisPair(t *testing.T) (*redis.Client, *redis.Client) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := publisher.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping redis %s: %v", opts.Addr, err)
	}
	return publisher, listener
}

func TestRedisWakeAgainstServer(t *testing.T) {
	publisher, listener := realRedisPair(t)
	prefix := fmt.Sprintf("cinderfish-test-%d", time.Now().UnixNano())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	woken := make(chan struct{}, 16)
	ready := make(chan struct{})
	var once sync.Once
	go func() {
		_ = ListenDeliveries(ctx, listener, prefix, func() {
			once.Do(func() { close(ready) })
			woken <- struct{}{}
		}, t.Logf)
	}()

	// Publish repeatedly until the listener is subscribed instead of
	// sleeping a fixed amount.
	w := NewRedisWake(publisher, prefix)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := w.NotifyDelivery(ctx); err != nil {
			t.Fatalf("notify: %v", err)
		}
		select {
		case <-woken:
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
	t.Fatal("listener never received a wakeup")
}
