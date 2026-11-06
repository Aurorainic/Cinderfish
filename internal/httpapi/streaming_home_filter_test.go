package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestStreamingHomeTimelineRespectsWithReplies proves the follow switch is
// applied when the event is published, not only in the channel unit test: a
// reply does not reach a follower who turned replies off, while a normal note
// still does.
func TestStreamingHomeTimelineRespectsWithReplies(t *testing.T) {
	ts := newStreamingServer(t, "open")
	_, aliceToken := signup(t, ts, "alice")
	_, bobToken := signup(t, ts, "bob")
	_, carolToken := signup(t, ts, "carol")

	// bob follows alice and opts out of her replies.
	resp, out := postWithToken(t, ts, "/api/following/create", bobToken, map[string]any{"userId": userIDOf(t, ts, "alice")})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("follow: %d %v", resp.StatusCode, out)
	}
	resp, out = postWithToken(t, ts, "/api/following/update", bobToken, map[string]any{
		"userId": userIDOf(t, ts, "alice"), "withReplies": false,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("following/update: %d %v", resp.StatusCode, out)
	}

	conn := dial(t, ts, bobToken)
	sendFrame(t, conn, map[string]any{
		"type": "connect",
		"body": map[string]any{"channel": "homeTimeline", "id": "home", "pong": true},
	})
	if m, ok := readWithTimeout(t, conn, 5*time.Second); !ok || m.Type != "connected" {
		t.Fatalf("homeTimeline connect: %+v ok=%v", m, ok)
	}

	// alice replies to carol, then writes a normal note. Events keep their
	// order on the bus, so the first note the follower sees must be the
	// normal one; a leaked reply would arrive first.
	carolNote := createNote(t, ts, carolToken, map[string]any{"text": "carol's note"})
	createNote(t, ts, aliceToken, map[string]any{"text": "a reply to carol", "replyId": carolNote})
	want := createNote(t, ts, aliceToken, map[string]any{"text": "a normal note"})

	got, ok := noteEventFor(conn, 8*time.Second)
	if !ok {
		t.Fatal("the normal note never reached the home timeline")
	}
	if got["id"] != want {
		t.Errorf("first streamed note = %v (%v), want the normal note %s", got["id"], got["text"], want)
	}
}

// noteEventFor reads channel frames until a home-timeline note event arrives.
func noteEventFor(conn *websocket.Conn, d time.Duration) (map[string]any, bool) {
	_ = conn.SetReadDeadline(time.Now().Add(d))
	for {
		var m message
		if err := conn.ReadJSON(&m); err != nil {
			return nil, false
		}
		if m.Type != "channel" {
			continue
		}
		var body map[string]any
		if err := json.Unmarshal(m.Body, &body); err != nil {
			continue
		}
		if body["type"] != "note" {
			continue
		}
		note, _ := body["body"].(map[string]any)
		return note, true
	}
}

// userIDOf resolves a username to its id through users/show.
func userIDOf(t *testing.T, ts *httptest.Server, username string) string {
	t.Helper()
	_, show := post(t, ts, "/api/users/show", map[string]any{"username": username})
	id, _ := show["id"].(string)
	if id == "" {
		t.Fatalf("no id for %s: %v", username, show)
	}
	return id
}
