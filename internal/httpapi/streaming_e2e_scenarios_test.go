package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/auth"
	"github.com/gorilla/websocket"
)

const wsWait = 5 * time.Second

// TestStreamingMeUpdated checks that a profile write reaches the account's own
// main channel: Firefish's init.ts replaces `$i` from this event, which is what
// keeps a second tab in step.
func TestStreamingMeUpdated(t *testing.T) {
	ts := newStreamingServer(t, "open")
	_, aliceToken := signup(t, ts, "alice")

	conn := dial(t, ts, aliceToken)
	sendFrame(t, conn, map[string]any{
		"type": "connect",
		"body": map[string]any{"channel": "main", "id": "main1", "pong": true},
	})
	connected := false
	for deadline := time.Now().Add(wsWait); time.Now().Before(deadline); {
		msg, ok := readWithTimeout(t, conn, time.Until(deadline))
		if !ok {
			break
		}
		if msg.Type == "connected" {
			connected = true
			break
		}
	}
	if !connected {
		t.Fatal("main channel did not connect")
	}

	resp, updated := post(t, ts, "/api/i/update", map[string]any{
		"i": aliceToken, "name": "w168 renamed",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i/update = %d %v", resp.StatusCode, updated)
	}

	deadline := time.Now().Add(wsWait)
	for {
		msg, ok := waitForChannelMessage(t, conn, "main1", time.Until(deadline))
		if !ok {
			t.Fatal("no meUpdated frame")
		}
		if msg["type"] != "meUpdated" {
			continue
		}
		body, _ := msg["body"].(map[string]any)
		if body["name"] != "w168 renamed" {
			t.Errorf("meUpdated name = %v, want w168 renamed", body["name"])
		}
		if _, ok := body["followingCount"]; !ok {
			t.Errorf("meUpdated is missing the derived counters: %v", body)
		}
		return
	}
}

// TestStreamingMeUpdatedOnTwoFactor checks that enabling TOTP reaches the
// account's main channel: the settings page in another tab reads
// `twoFactorEnabled` from `$i`, and Misskey publishes meUpdated here too.
func TestStreamingMeUpdatedOnTwoFactor(t *testing.T) {
	ts := newStreamingServer(t, "open")
	_, aliceToken := signup(t, ts, "alice")

	conn := dial(t, ts, aliceToken)
	sendFrame(t, conn, map[string]any{
		"type": "connect",
		"body": map[string]any{"channel": "main", "id": "main1", "pong": true},
	})
	connected := false
	for deadline := time.Now().Add(wsWait); time.Now().Before(deadline); {
		msg, ok := readWithTimeout(t, conn, time.Until(deadline))
		if !ok {
			break
		}
		if msg.Type == "connected" {
			connected = true
			break
		}
	}
	if !connected {
		t.Fatal("main channel did not connect")
	}

	resp, registered := post(t, ts, "/api/i/2fa/register", map[string]any{
		"i": aliceToken, "password": "correct horse battery staple",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("2fa/register = %d %v", resp.StatusCode, registered)
	}
	secret, _ := registered["secret"].(string)
	code, err := auth.TOTPCode(secret, time.Now())
	if err != nil {
		t.Fatalf("totp code: %v", err)
	}
	if status := postNoContentWithToken(t, ts, "/api/i/2fa/done", aliceToken, map[string]any{"token": code}); status != http.StatusNoContent {
		t.Fatalf("2fa/done = %d, want 204", status)
	}

	deadline := time.Now().Add(wsWait)
	for {
		msg, ok := waitForChannelMessage(t, conn, "main1", time.Until(deadline))
		if !ok {
			t.Fatal("no meUpdated frame after enabling 2FA")
		}
		if msg["type"] != "meUpdated" {
			continue
		}
		body, _ := msg["body"].(map[string]any)
		if body["twoFactorEnabled"] != true {
			t.Errorf("meUpdated twoFactorEnabled = %v, want true", body["twoFactorEnabled"])
		}
		return
	}
}

// TestStreamingDriveFileCreated checks that an upload reaches the uploader's
// main channel: Firefish's photos widget inserts the file from this event.
func TestStreamingDriveFileCreated(t *testing.T) {
	ts := newStreamingServer(t, "open")
	_, aliceToken := signup(t, ts, "alice")

	conn := dial(t, ts, aliceToken)
	sendFrame(t, conn, map[string]any{
		"type": "connect",
		"body": map[string]any{"channel": "main", "id": "main1", "pong": true},
	})
	connected := false
	for deadline := time.Now().Add(wsWait); time.Now().Before(deadline); {
		msg, ok := readWithTimeout(t, conn, time.Until(deadline))
		if !ok {
			break
		}
		if msg.Type == "connected" {
			connected = true
			break
		}
	}
	if !connected {
		t.Fatal("main channel did not connect")
	}

	resp, uploaded := uploadFile(t, ts.URL, aliceToken, "photo.png",
		bytes.NewBufferString("\x89PNG\r\n\x1a\nfake"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload: status %d (%v)", resp.StatusCode, uploaded)
	}
	fileID, _ := uploaded["id"].(string)
	if fileID == "" {
		t.Fatalf("upload response has no id: %v", uploaded)
	}

	deadline := time.Now().Add(wsWait)
	for {
		msg, ok := waitForChannelMessage(t, conn, "main1", time.Until(deadline))
		if !ok {
			t.Fatalf("no driveFileCreated frame for %s", fileID)
		}
		if msg["type"] != "driveFileCreated" {
			continue
		}
		body, _ := msg["body"].(map[string]any)
		if body["id"] != fileID {
			t.Fatalf("driveFileCreated id = %v, want %s", body["id"], fileID)
		}
		if name, _ := body["name"].(string); name != "photo.png" {
			t.Errorf("driveFileCreated name = %q, want photo.png", name)
		}
		if typ, _ := body["type"].(string); typ == "" {
			t.Errorf("driveFileCreated has no type: %v", body)
		}
		return
	}
}

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

	// The author reacts to her own note, which the new visibility check
	// allows even on a followers-only note; bob must still see nothing.
	resp, _ := post(t, ts, "/api/notes/reactions/create", map[string]any{
		"i": aliceToken, "noteId": noteID, "reaction": "👀",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reaction: status %d", resp.StatusCode)
	}

	if m, ok := readWithTimeout(t, conn, 400*time.Millisecond); ok {
		t.Fatalf("bob must not receive noteUpdated for a followers-only note; got %q", m.Type)
	}
}

// TestStreamingReadNotificationIDs checks the `notificationIds` body: a folded
// notification row marks its whole group in one frame, and ids that are not
// listed stay unread.
func TestStreamingReadNotificationIDs(t *testing.T) {
	ts := newStreamingServer(t, "open")
	_, aliceToken := signup(t, ts, "alice")
	_, bobToken := signup(t, ts, "bob")
	_, carolToken := signup(t, ts, "carol")

	_, created := post(t, ts, "/api/notes/create", map[string]any{
		"i": aliceToken, "text": "fold me",
	})
	noteID := created["createdNote"].(map[string]any)["id"].(string)
	for _, token := range []string{bobToken, carolToken} {
		if resp, body := post(t, ts, "/api/notes/reactions/create", map[string]any{
			"i": token, "noteId": noteID, "reaction": "⭐",
		}); resp.StatusCode != http.StatusOK {
			t.Fatalf("reaction failed: %v", body)
		}
	}

	// Wait for both notifications, then pick the older one.
	var ids []string
	deadline := time.Now().Add(wsWait)
	for {
		_, raw := postRaw(t, ts, "/api/notifications", map[string]any{"i": aliceToken, "markAsRead": false})
		list, _ := raw.([]any)
		ids = ids[:0]
		for _, item := range list {
			m, _ := item.(map[string]any)
			id, _ := m["id"].(string)
			if id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("notifications = %v, want 2", list)
		}
		time.Sleep(50 * time.Millisecond)
	}
	// The list is newest first, so ids[1] is the older notification.
	target := ids[1]

	conn := dial(t, ts, aliceToken)
	sendFrame(t, conn, map[string]any{
		"type": "readNotification",
		"body": map[string]any{"notificationIds": []string{target}},
	})

	deadline = time.Now().Add(wsWait)
	for {
		_, raw := postRaw(t, ts, "/api/notifications", map[string]any{"i": aliceToken, "markAsRead": false})
		list, _ := raw.([]any)
		read := map[string]bool{}
		for _, item := range list {
			m, _ := item.(map[string]any)
			id, _ := m["id"].(string)
			read[id] = m["isRead"] == true
		}
		if read[target] {
			if read[ids[0]] {
				t.Fatalf("notification %s was not listed but is read: %v", ids[0], read)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("notification %s was not marked read: %v", target, read)
		}
		time.Sleep(50 * time.Millisecond)
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
		_, raw := postRaw(t, ts, "/api/notifications", map[string]any{"i": aliceToken, "markAsRead": false})
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
