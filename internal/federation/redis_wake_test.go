package federation

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisWakeReachesListener(t *testing.T) {
	mr := miniredis.RunT(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	listener := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		publisher.Close()
		listener.Close()
	})

	woken := make(chan struct{}, 4)
	var once sync.Once
	ready := make(chan struct{})
	go func() {
		_ = ListenDeliveries(ctx, listener, "test", func() {
			once.Do(func() { close(ready) })
			woken <- struct{}{}
		}, func(string, ...any) {})
	}()

	// Publish repeatedly until the listener has subscribed, rather than
	// sleeping a fixed amount.
	deadline := time.Now().Add(3 * time.Second)
	w := NewRedisWake(publisher, "test")
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

func TestDelivererWakeNeverBlocks(t *testing.T) {
	d := &Deliverer{wake: make(chan struct{}, 1)}
	d.Wake()
	d.Wake()
	d.Wake()
	select {
	case <-d.wake:
	case <-time.After(time.Second):
		t.Fatal("Wake did not enqueue a wakeup")
	}
}
