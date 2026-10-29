package federation

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisWake publishes delivery wakeups so workers in other processes poll
// the durable deliver_queue immediately. It carries no work itself: the
// queue table remains the source of truth.
type RedisWake struct {
	client  *redis.Client
	channel string
}

// NewRedisWake returns a waker publishing on <prefix>:deliver.
func NewRedisWake(client *redis.Client, prefix string) *RedisWake {
	if prefix == "" {
		prefix = "cinderfish"
	}
	return &RedisWake{client: client, channel: prefix + ":deliver"}
}

// NotifyDelivery publishes one wakeup.
func (w *RedisWake) NotifyDelivery(ctx context.Context) error {
	if w.client == nil {
		return errors.New("federation: redis waker has no client")
	}
	return w.client.Publish(ctx, w.channel, "1").Err()
}

// ListenDeliveries subscribes to wakeups and calls wake for each one,
// reconnecting until ctx is cancelled.
func ListenDeliveries(ctx context.Context, client *redis.Client, prefix string, wake func(), logf func(string, ...any)) error {
	if logf == nil {
		logf = log.Printf
	}
	if client == nil || wake == nil {
		return errors.New("federation: delivery listener needs a client and a wake function")
	}
	w := NewRedisWake(client, prefix)
	for {
		err := listenDeliveriesOnce(ctx, w.channel, client, wake, logf)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			logf("federation: delivery wakeup listener stopped: %v", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
		}
	}
}

func listenDeliveriesOnce(ctx context.Context, channel string, client *redis.Client, wake func(), logf func(string, ...any)) error {
	pubsub := client.Subscribe(ctx, channel)
	defer pubsub.Close()
	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}
	logf("federation: listening for delivery wakeups on redis channel %q", channel)

	for {
		_, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("federation: redis receive: %w", err)
		}
		wake()
	}
}
