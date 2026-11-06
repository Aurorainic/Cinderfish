package federation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestRedisWakeSurvivesRestart starts a throwaway redis-server, wakes a listener,
// restarts the server on the same port, and proves the listener subscribes again
// and receives a second wakeup. The retry loop in ListenDeliveries is what makes
// this work; the test is skipped when redis-server is not installed.
func TestRedisWakeSurvivesRestart(t *testing.T) {
	bin, err := exec.LookPath("redis-server")
	if err != nil {
		t.Skip("redis-server is not installed; skipping restart test")
	}
	port := freePort(t)
	dir := t.TempDir()
	start := func() *exec.Cmd {
		cmd := exec.Command(bin, "--port", fmt.Sprint(port), "--save", "", "--appendonly", "no", "--dir", dir)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Start(); err != nil {
			t.Fatalf("start redis-server: %v", err)
		}
		return cmd
	}
	stop := func(cmd *exec.Cmd) {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}
	waitReady := func(url string) {
		t.Helper()
		client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("127.0.0.1:%d", port)})
		defer client.Close()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			err := client.Ping(ctx).Err()
			cancel()
			if err == nil {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatalf("redis-server at %s did not become ready", url)
	}

	cmd := start()
	defer stop(cmd)
	url := fmt.Sprintf("redis://127.0.0.1:%d/0", port)
	waitReady(url)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		t.Fatalf("random: %v", err)
	}
	prefix := "cinderfish-restart-" + hex.EncodeToString(random)

	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	publisher := redis.NewClient(opts)
	defer publisher.Close()
	listener := redis.NewClient(opts)
	defer listener.Close()

	woken := make(chan struct{}, 4)
	go func() {
		_ = ListenDeliveries(ctx, listener, prefix, func() {
			select {
			case woken <- struct{}{}:
			default:
			}
		}, func(string, ...any) {})
	}()

	publishUntilWoken := func(t *testing.T, waker *RedisWake, label string) {
		t.Helper()
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			if err := waker.NotifyDelivery(ctx); err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			select {
			case <-woken:
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
		t.Fatalf("%s: listener never received a wakeup", label)
	}

	publishUntilWoken(t, NewRedisWake(publisher, prefix), "before restart")

	// Restart on the same port; the listener's retry loop must resubscribe.
	stop(cmd)
	time.Sleep(300 * time.Millisecond)
	cmd = start()
	defer stop(cmd)
	waitReady(url)
	publishUntilWoken(t, NewRedisWake(publisher, prefix), "after restart")
}
