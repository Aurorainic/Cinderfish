package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

const wsWait = 5 * time.Second

// TestStreamingHandshake covers the HTTP-level outcomes of the upgrade.
func TestStreamingHandshake(t *testing.T) {
	ts := newStreamingServer(t, "open")

	t.Run("invalid token is rejected with 401", func(t *testing.T) {
		_, resp, err := websocket.DefaultDialer.Dial(wsURL(ts, "/streaming?i=not-a-real-token"), nil)
		if err == nil {
			t.Fatal("expected the handshake to fail")
		}
		if resp == nil || resp.StatusCode != http.StatusUnauthorized {
			got := 0
			if resp != nil {
				got = resp.StatusCode
			}
			t.Fatalf("status = %d, want 401", got)
		}
		auth := resp.Header.Get("WWW-Authenticate")
		if !stringsContains(auth, "invalid_token") {
			t.Errorf("WWW-Authenticate = %q, want it to mention invalid_token", auth)
		}
	})

	t.Run("anonymous connection is accepted", func(t *testing.T) {
		conn := dial(t, ts, "")
		// The local timeline is open to anonymous connections.
		sendFrame(t, conn, map[string]any{
			"type": "connect",
			"body": map[string]any{"channel": "localTimeline", "id": "lt", "pong": true},
		})
		m, ok := readWithTimeout(t, conn, wsWait)
		if !ok {
			t.Fatal("no reply to connect")
		}
		if m.Type != "connected" {
			t.Fatalf("message = %q, want connected", m.Type)
		}
	})

	t.Run("main is refused anonymously", func(t *testing.T) {
		conn := dial(t, ts, "")
		sendFrame(t, conn, map[string]any{
			"type": "connect",
			"body": map[string]any{"channel": "main", "id": "m", "pong": true},
		})
		// No reply at all: upstream neither answers nor closes.
		if m, ok := readWithTimeout(t, conn, 300*time.Millisecond); ok {
			t.Fatalf("unexpected reply %q to a credential-required channel", m.Type)
		}
	})
}

// TestStreamingNoteDelivery is the end-to-end path: create a note over
// HTTP and observe it on a subscribed WebSocket.
func TestStreamingNoteDelivery(t *testing.T) {
	ts := newStreamingServer(t, "open")
	_, token := signup(t, ts, "alice")

	conn := dial(t, ts, token)
	sendFrame(t, conn, map[string]any{
		"type": "connect",
		"body": map[string]any{"channel": "localTimeline", "id": "lt", "pong": true},
	})
	if m, ok := readWithTimeout(t, conn, wsWait); !ok || m.Type != "connected" {
		t.Fatalf("connect failed: %+v ok=%v", m, ok)
	}

	resp, out := post(t, ts, "/api/notes/create", map[string]any{
		"i":    token,
		"text": "hello streaming",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: status %d body %v", resp.StatusCode, out)
	}
	wantID := out["createdNote"].(map[string]any)["id"].(string)

	body, ok := waitForChannelMessage(t, conn, "lt", wsWait)
	if !ok {
		t.Fatal("no note event arrived")
	}
	if body["type"] != "note" {
		t.Errorf("event type = %v, want note", body["type"])
	}
	note, _ := body["body"].(map[string]any)
	if note["id"] != wantID {
		t.Errorf("note id = %v, want %s", note["id"], wantID)
	}
	if note["text"] != "hello streaming" {
		t.Errorf("note text = %v", note["text"])
	}
}

// TestStreamingMainChannelNotification checks that a reaction publishes
// both the note-stream event and the author's notification.
func TestStreamingMainChannelNotification(t *testing.T) {
	ts := newStreamingServer(t, "open")
	_, aliceToken := signup(t, ts, "alice")
	_, bobToken := signup(t, ts, "bob")

	_, created := post(t, ts, "/api/notes/create", map[string]any{
		"i": aliceToken, "text": "react to me",
	})
	noteID := created["createdNote"].(map[string]any)["id"].(string)

	// Alice listens on her main channel and on the note itself.
	conn := dial(t, ts, aliceToken)
	sendFrame(t, conn, map[string]any{
		"type": "connect",
		"body": map[string]any{"channel": "main", "id": "main", "pong": true},
	})
	if m, ok := readWithTimeout(t, conn, wsWait); !ok || m.Type != "connected" {
		t.Fatalf("main connect failed: %+v", m)
	}
	sendFrame(t, conn, map[string]any{"type": "subNote", "body": map[string]any{"id": noteID}})

	// Bob reacts.
	resp, out := post(t, ts, "/api/notes/reactions/create", map[string]any{
		"i": bobToken, "noteId": noteID, "reaction": "👍",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reaction: status %d body %v", resp.StatusCode, out)
	}

	var sawNotification, sawNoteUpdated bool
	deadline := time.Now().Add(wsWait)
	for time.Now().Before(deadline) && !(sawNotification && sawNoteUpdated) {
		m, ok := readWithTimeout(t, conn, time.Until(deadline))
		if !ok {
			break
		}
		switch m.Type {
		case "channel":
			var body map[string]any
			_ = json.Unmarshal(m.Body, &body)
			if body["id"] == "main" && body["type"] == "notification" {
				n, _ := body["body"].(map[string]any)
				if n["type"] == "reaction" && n["reaction"] == "👍" {
					sawNotification = true
				}
			}
		case "noteUpdated":
			var body map[string]any
			_ = json.Unmarshal(m.Body, &body)
			if body["id"] == noteID && body["type"] == "reacted" {
				sawNoteUpdated = true
			}
		}
	}

	if !sawNotification {
		t.Error("no reaction notification on the main channel")
	}
	if !sawNoteUpdated {
		t.Error("no noteUpdated event for the subNote subscription")
	}
}

// TestStreamingSubNoteVisibility checks that a private note's updates do
// not reach an unentitled subscriber.
func TestStreamingSubNoteVisibility(t *testing.T) {
	ts := newStreamingServer(t, "open")
	_, aliceToken := signup(t, ts, "alice")
	_, bobToken := signup(t, ts, "bob")

	_, created := post(t, ts, "/api/notes/create", map[string]any{
		"i": aliceToken, "text": "secret", "visibility": "followers",
	})
	noteID := created["createdNote"].(map[string]any)["id"].(string)

	conn := dial(t, ts, bobToken)
	sendFrame(t, conn, map[string]any{"type": "subNote", "body": map[string]any{"id": noteID}})
	time.Sleep(100 * time.Millisecond)

	resp, _ := post(t, ts, "/api/notes/reactions/create", map[string]any{
		"i": bobToken, "noteId": noteID, "reaction": "👀",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reaction: status %d", resp.StatusCode)
	}

	if m, ok := readWithTimeout(t, conn, 400*time.Millisecond); ok {
		t.Fatalf("bob must not receive noteUpdated for a followers-only note; got %q", m.Type)
	}
}

// TestStreamingReadNotification checks the readNotification frame.
func TestStreamingReadNotification(t *testing.T) {
	ts := newStreamingServer(t, "open")
	_, aliceToken := signup(t, ts, "alice")
	_, bobToken := signup(t, ts, "bob")

	_, created := post(t, ts, "/api/notes/create", map[string]any{
		"i": aliceToken, "text": "notify me",
	})
	noteID := created["createdNote"].(map[string]any)["id"].(string)
	if resp, _ := post(t, ts, "/api/notes/reactions/create", map[string]any{
		"i": bobToken, "noteId": noteID, "reaction": "⭐",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("reaction failed")
	}

	conn := dial(t, ts, aliceToken)
	sendFrame(t, conn, map[string]any{"type": "readNotification"})

	// Give the server a moment, then read the unread count through the
	// API, which is the observable effect of the frame.
	deadline := time.Now().Add(wsWait)
	for {
		_, raw := postRaw(t, ts, "/api/notifications", map[string]any{"i": aliceToken})
		list, _ := raw.([]any)
		allRead := len(list) > 0
		for _, item := range list {
			m, _ := item.(map[string]any)
			if m["isRead"] != true {
				allRead = false
			}
		}
		if allRead {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("notifications were not marked read: %v", list)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// stringsContains is a tiny helper to avoid importing strings twice.
func stringsContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
