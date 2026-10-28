package streaming

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Limits copied from Misskey's stream/Connection.ts.
const (
	// maxChannelsPerConnection caps how many channels one WebSocket may
	// hold open.
	maxChannelsPerConnection = 32
	// maxSubscribingNotesPerConnection caps subNote subscriptions; the
	// oldest subscription is evicted when the cap is reached.
	maxSubscribingNotesPerConnection = 1536
	// pingInterval is how often the server pings an idle connection.
	pingInterval = 60 * time.Second
	// pongTimeout is how long a connection may stay silent before it is
	// torn down. Only pong frames refresh liveness, matching upstream.
	pongTimeout = 120 * time.Second
	// writeWait bounds a single write.
	writeWait = 10 * time.Second
	// contextRefreshInterval is how often the follow graph used by
	// channel filters is reloaded.
	contextRefreshInterval = 10 * time.Second
	// outboundQueueSize bounds the per-connection outbound buffer. When
	// it fills -- a client that cannot keep up -- the connection is
	// terminated so the client reconnects with an empty backlog, which is
	// the same intent as Misskey's bufferedAmount guard.
	outboundQueueSize = 256
)

// outbound is one message queued for a client.
type outbound struct {
	Type string `json:"type"`
	Body any    `json:"body"`
}

// wsConn wraps the gorilla connection with a serialised writer.
//
// Gorilla permits one concurrent writer, so every write is funnelled
// through a single goroutine reading from queue.
type wsConn struct {
	ws    *websocket.Conn
	queue chan outbound
	done  chan struct{}
	once  sync.Once
}

func newWSConn(ws *websocket.Conn) *wsConn {
	return &wsConn{
		ws:    ws,
		queue: make(chan outbound, outboundQueueSize),
		done:  make(chan struct{}),
	}
}

// close shuts the writer down. It is safe to call repeatedly.
func (c *wsConn) close() {
	c.once.Do(func() { close(c.done) })
}

// send queues a message. It never blocks: when the queue is full the
// connection is closed instead, mirroring Misskey's bufferedAmount
// guard.
func (c *wsConn) send(typ string, body any) {
	select {
	case <-c.done:
		return
	case c.queue <- outbound{Type: typ, Body: body}:
	default:
		// The client cannot keep up; drop it so it reconnects with a
		// fresh, empty backlog.
		c.close()
	}
}

// writeLoop drains the queue onto the wire until the connection ends.
func (c *wsConn) writeLoop() {
	for {
		select {
		case <-c.done:
			return
		case m := <-c.queue:
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteJSON(m); err != nil {
				c.close()
				return
			}
		}
	}
}

// heartbeat pings the peer and terminates connections that stopped
// answering pongs.
func (c *wsConn) heartbeat(lastSeen *atomicTime) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			if time.Since(lastSeen.load()) > pongTimeout {
				_ = c.ws.Close()
				c.close()
				return
			}
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				c.close()
				return
			}
		}
	}
}

// atomicTime is a small race-free time value.
type atomicTime struct {
	mu sync.RWMutex
	t  time.Time
}

func (a *atomicTime) store(t time.Time) {
	a.mu.Lock()
	a.t = t
	a.mu.Unlock()
}

func (a *atomicTime) load() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.t
}
