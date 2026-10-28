package streaming

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// dialTestWS returns a server-side wsConn and its client peer.
func dialTestWS(t *testing.T) (*wsConn, *websocket.Conn) {
	t.Helper()

	upgraded := make(chan *websocket.Conn, 1)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		upgraded <- ws
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	select {
	case ws := <-upgraded:
		return newWSConn(ws), client
	case <-time.After(5 * time.Second):
		t.Fatal("the server never upgraded the connection")
		return nil, nil
	}
}

// TestOverloadedConnectionIsTerminated covers the load-shedding path: a
// client that cannot keep up must lose its socket, not just its writer.
func TestOverloadedConnectionIsTerminated(t *testing.T) {
	c, client := dialTestWS(t)

	// No writeLoop runs, so the queue stays full and the next send is the
	// overflow.
	for i := 0; i < outboundQueueSize; i++ {
		c.send("channel", map[string]any{"n": i})
	}
	c.send("channel", map[string]any{"overflow": true})

	select {
	case <-c.done:
	default:
		t.Fatal("an overflowing queue must close the connection")
	}

	// The peer must observe the close. Before the fix only the writer was
	// stopped, leaving a half-open socket that keeps its subscriptions.
	_ = client.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, _, err := client.ReadMessage(); err == nil {
		t.Fatal("the peer kept reading from an overloaded connection")
	}
}

// TestTerminatedConnectionReleasesSubscriptions is the regression test for
// the leak an overloaded connection used to cause: the read loop is the
// only thing that runs dispose, so it must unblock.
func TestTerminatedConnectionReleasesSubscriptions(t *testing.T) {
	ws, _ := dialTestWS(t)
	bus := NewBus()
	conn := newConnection(context.Background(), ws, bus, Viewer{})

	served := make(chan struct{})
	go func() {
		conn.serve()
		close(served)
	}()

	// serve registers the notesStream listener before it parks in
	// ReadMessage, so a subscriber proves the loop is running.
	waitFor(t, "the connection's bus subscription", func() bool { return bus.SubscriberCount() > 0 })

	ws.terminate()

	select {
	case <-served:
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after the socket was closed")
	}
	if got := bus.SubscriberCount(); got != 0 {
		t.Errorf("bus subscriptions = %d, want 0 after the connection ended", got)
	}
}

// TestSendAfterCloseDoesNotQueue checks that a closed connection neither
// queues nor panics; the test harness has no socket at all, which is also
// the nil guard on terminate.
func TestSendAfterCloseDoesNotQueue(t *testing.T) {
	h := newTestConnection(t, Viewer{})
	h.ws.close()
	h.ws.send("channel", map[string]any{"id": "a"})
	if got := len(h.ws.queue); got != 0 {
		t.Errorf("queued %d messages on a closed connection, want 0", got)
	}
	h.ws.terminate()
}

// TestWriteLoopStopsAfterClose checks a closed connection stops writing
// even with a backlog queued. It builds the backlog before the writer
// exists, so the outcome does not depend on scheduling.
func TestWriteLoopStopsAfterClose(t *testing.T) {
	c, client := dialTestWS(t)

	c.send("channel", map[string]any{"id": "a"})
	c.send("channel", map[string]any{"id": "b"})
	c.close()

	stop := make(chan struct{})
	go func() {
		c.writeLoop()
		close(stop)
	}()

	select {
	case <-stop:
	case <-time.After(5 * time.Second):
		t.Fatal("writeLoop did not stop on a closed connection")
	}

	// Nothing may reach the wire once the connection is closed.
	_ = client.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, _, err := client.ReadMessage(); err == nil {
		t.Fatal("a closed connection wrote a queued message")
	}
}

// waitFor blocks until cond holds or the deadline passes.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}
