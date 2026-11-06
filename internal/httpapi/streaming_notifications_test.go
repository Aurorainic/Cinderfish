package httpapi_test

import (
	"net/http"
	"testing"
	"time"
)

// TestStreamingNotificationsPushTypes proves the main channel pushes each new
// notification with its type, and the unread counter follows.
func TestStreamingNotificationsPushTypes(t *testing.T) {
	ts := newStreamingServer(t, "open")
	_, aliceToken := signup(t, ts, "alice")
	_, bobToken := signup(t, ts, "bob")
	_, carolToken := signup(t, ts, "carol")
	_, daveToken := signup(t, ts, "dave")
	aliceID := userIDOf(t, ts, "alice")

	conn := dial(t, ts, aliceToken)
	sendFrame(t, conn, map[string]any{
		"type": "connect",
		"body": map[string]any{"channel": "main", "id": "main", "pong": true},
	})
	if m, ok := readWithTimeout(t, conn, 5*time.Second); !ok || m.Type != "connected" {
		t.Fatalf("main connect: %+v ok=%v", m, ok)
	}

	// bob mentions alice.
	createNote(t, ts, bobToken, map[string]any{"text": "@alice hello there", "visibility": "public"})
	// carol replies to a note of alice's.
	aliceNote := createNote(t, ts, aliceToken, map[string]any{"text": "reply target", "visibility": "public"})
	createNote(t, ts, carolToken, map[string]any{"text": "an answer", "replyId": aliceNote})
	// dave follows alice.
	if resp, out := postWithToken(t, ts, "/api/following/create", daveToken, map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("follow = %d %v", resp.StatusCode, out)
	}

	want := map[string]bool{"mention": false, "reply": false, "follow": false}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		body, ok := waitForChannelMessage(t, conn, "main", time.Until(deadline))
		if !ok {
			break
		}
		if body["type"] != "notification" {
			continue
		}
		packed, _ := body["body"].(map[string]any)
		kind, _ := packed["type"].(string)
		if _, known := want[kind]; known {
			want[kind] = true
		}
		if want["mention"] && want["reply"] && want["follow"] {
			break
		}
	}
	for kind, seen := range want {
		if !seen {
			t.Errorf("no %s notification event arrived", kind)
		}
	}

	// The unread counter is the observable effect of those rows.
	_, me := postWithToken(t, ts, "/api/i", aliceToken, map[string]any{})
	unread, _ := me["unreadNotificationsCount"].(float64)
	if unread < 3 {
		t.Errorf("unreadNotificationsCount = %v, want at least 3", me["unreadNotificationsCount"])
	}

	// The pushed body keeps the same shape the HTTP list uses.
	list := postListWithToken(t, ts, "/api/i/notifications", aliceToken, nil)
	if len(list) < 3 {
		t.Errorf("notification list = %d, want at least 3", len(list))
	}
	for _, item := range list {
		n, _ := item.(map[string]any)
		if _, ok := n["body"]; ok {
			t.Errorf("notification still carries a body field: %v", n)
		}
	}
}
