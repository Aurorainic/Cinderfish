package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRelay carries cross-process Streaming events over Redis pub/sub.
//
// It implements Relay, so the Bus is unaware of the difference between
// Redis and the PostgreSQL LISTEN/NOTIFY transport. Redis has no payload
// cap, so events are never chunked on this path.
type RedisRelay struct {
	client  *redis.Client
	channel string
}

// NewRedisRelay returns a relay publishing on <prefix>:stream.
func NewRedisRelay(client *redis.Client, prefix string) *RedisRelay {
	if prefix == "" {
		prefix = "cinderfish"
	}
	return &RedisRelay{client: client, channel: prefix + ":stream"}
}

// Notify publishes one event for the other processes of the instance.
func (r *RedisRelay) Notify(ctx context.Context, origin, subject string, msg Message) error {
	if r.client == nil {
		return errors.New("streaming: redis relay has no client")
	}
	w := wireMessage{Subject: subject, Type: msg.Type, Origin: origin}
	if json.Valid(msg.Body) {
		w.Body = string(msg.Body)
	}
	payload, err := json.Marshal(w)
	if err != nil {
		return err
	}
	return r.client.Publish(ctx, r.channel, payload).Err()
}

// ListenRedis subscribes to events published by other processes and
// feeds them into bus, blocking until ctx is cancelled.
//
// As with the PostgreSQL listener, a dropped connection only loses events
// produced by *other* processes; the local Publish path is unaffected. The
// loop therefore reconnects.
func ListenRedis(ctx context.Context, client *redis.Client, prefix string, bus *Bus, logf func(string, ...any)) error {
	if logf == nil {
		logf = log.Printf
	}
	if client == nil || bus == nil {
		return errors.New("streaming: redis listener needs a client and a bus")
	}
	relay := NewRedisRelay(client, prefix)
	for {
		err := listenRedisOnce(ctx, relay.channel, client, bus, logf)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			logf("streaming: redis listener stopped: %v", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
		}
	}
}

func listenRedisOnce(ctx context.Context, channel string, client *redis.Client, bus *Bus, logf func(string, ...any)) error {
	pubsub := client.Subscribe(ctx, channel)
	defer pubsub.Close()

	// Wait for the subscribe acknowledgement before serving traffic, so
	// events published right after startup are not dropped by a race.
	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}
	logf("streaming: listening for cross-process events on redis channel %q", channel)

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("streaming: redis receive: %w", err)
		}
		var w wireMessage
		if err := json.Unmarshal([]byte(msg.Payload), &w); err != nil {
			logf("streaming: malformed redis event: %v", err)
			continue
		}
		if w.Origin == bus.id {
			// This process published it and already fanned it out.
			continue
		}
		if w.Total != 0 || w.RelayID != "" {
			// Chunking belongs to the PostgreSQL transport; a chunked
			// event on Redis is a mixed-configuration mistake.
			logf("streaming: ignoring a chunked event on redis for %q", w.Subject)
			continue
		}
		bus.deliver(w.Subject, Message{Type: w.Type, Body: decodeBody(w.Body)})
	}
}
