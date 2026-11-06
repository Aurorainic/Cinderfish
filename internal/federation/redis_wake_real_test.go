package federation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestRedisWakeRealServer exercises the delivery wakeup against a real Redis,
// which miniredis cannot fully stand in for (TCP, subscription timing). Set
// CINDERFISH_TEST_REDIS_URL=redis://127.0.0.1:6379/0 to run it; the channel prefix is
// randomised so a shared server sees no cross-talk.
func TestRedisWakeRealServer(t *testing.T) {
	url := os.Getenv("CINDERFISH_TEST_REDIS_URL")
	if url == "" {
		t.Skip("CINDERFISH_TEST_REDIS_URL is not set; skipping real Redis test")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse redis url: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		t.Fatalf("random: %v", err)
	}
	prefix := "cinderfish-test-" + hex.EncodeToString(random)
	// Isolate this run in its own DB when the URL allows it.
	opts.DB = 15
	publisher := redis.NewClient(opts)
	listener := redis.NewClient(opts)
	defer publisher.Close()
	defer listener.Close()

	woken := make(chan struct{}, 1)
	go func() {
		_ = ListenDeliveries(ctx, listener, prefix, func() {
			select {
			case woken <- struct{}{}:
			default:
			}
		}, func(string, ...any) {})
	}()

	w := NewRedisWake(publisher, prefix)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if err := w.NotifyDelivery(ctx); err != nil {
			t.Fatalf("notify: %v", err)
		}
		select {
		case <-woken:
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
	t.Fatal("listener never received a wakeup from the real Redis")
}

// freePort reserves a loopback port for a test-managed process.
func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
