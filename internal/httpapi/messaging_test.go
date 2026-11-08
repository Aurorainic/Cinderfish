package httpapi_test

import (
	"net/http"
	"testing"
)

// The messaging/* family (fork-preserved legacy contract; the group
// branch is a recorded deferral).

func sendAndExpect(t *testing.T, f *fedInstance, senderToken string, body map[string]any, wantStatus int) map[string]any {
	t.Helper()
	resp, out := postWithToken(t, f.ts, "/api/messaging/messages/create", senderToken, body)
	if resp.StatusCode != wantStatus {
		t.Fatalf("messages/create %v: %d %v", body, resp.StatusCode, out)
	}
	return out
}

func TestMessagingThreadAndReadState(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")
	_, carolID := f.registerUser(t, "carol")

	// Content is required.
	out := sendAndExpect(t, f, aliceToken, map[string]any{"userId": bobID}, http.StatusBadRequest)
	if errCode(out) != "CONTENT_REQUIRED" {
		t.Fatalf("no-content error = %v", out)
	}
	// Sending to yourself is refused.
	out = sendAndExpect(t, f, aliceToken, map[string]any{"userId": aliceID, "text": "me"}, http.StatusBadRequest)
	if errCode(out) != "RECIPIENT_IS_YOURSELF" {
		t.Fatalf("self error = %v", out)
	}

	// A conversation.
	first := sendAndExpect(t, f, aliceToken, map[string]any{"userId": bobID, "text": "hello bob"}, http.StatusOK)
	if first["text"] != "hello bob" || first["isRead"] != false {
		t.Fatalf("created message = %v", first)
	}
	if _, ok := first["recipient"].(map[string]any); !ok {
		t.Errorf("create response embeds no recipient: %v", first)
	}
	sendAndExpect(t, f, bobToken, map[string]any{"userId": aliceID, "text": "hi alice"}, http.StatusOK)
	sendAndExpect(t, f, aliceToken, map[string]any{"userId": carolID, "text": "wrong thread"}, http.StatusOK)

	// The thread is directional and ordered; messages listing defaults
	// markAsRead to true, which flips bob's unread copy.
	thread := postListWithToken(t, f.ts, "/api/messaging/messages", bobToken, map[string]any{"userId": aliceID})
	if len(thread) != 2 {
		t.Fatalf("thread = %d messages, want 2", len(thread))
	}
	last, _ := thread[0].(map[string]any)
	if last["text"] != "hi alice" {
		t.Errorf("thread[0] = %v, want the newest message", last)
	}

	// history: one entry per conversation partner, newest first.
	history := postListWithToken(t, f.ts, "/api/messaging/history", aliceToken, nil)
	if len(history) != 2 {
		t.Fatalf("history = %d entries, want 2 conversations", len(history))
	}
	top, _ := history[0].(map[string]any)
	if top["userId"] != bobID && top["recipientId"] != bobID {
		t.Errorf("history[0] peers = %v/%v, want bob's conversation first", top["userId"], top["recipientId"])
	}
	// The group branch is a deferral and lists nothing.
	if got := postListWithToken(t, f.ts, "/api/messaging/history", aliceToken, map[string]any{"group": true}); len(got) != 0 {
		t.Errorf("group history = %d, want 0", len(got))
	}

	// The unread flag follows the reads: alice has not opened bob's reply.
	_, me := postWithToken(t, f.ts, "/api/i", aliceToken, nil)
	if me["hasUnreadMessagingMessage"] != true {
		t.Errorf("hasUnreadMessagingMessage = %v, want true", me["hasUnreadMessagingMessage"])
	}
	// Opening the thread as alice reads bob's message.
	_ = postListWithToken(t, f.ts, "/api/messaging/messages", aliceToken, map[string]any{"userId": bobID})
	_, me = postWithToken(t, f.ts, "/api/i", aliceToken, nil)
	if me["hasUnreadMessagingMessage"] != false {
		t.Errorf("hasUnreadMessagingMessage after read = %v, want false", me["hasUnreadMessagingMessage"])
	}
	// The read message carries the flipped flag.
	readMsg := postListWithToken(t, f.ts, "/api/messaging/messages", aliceToken, map[string]any{"userId": bobID, "markAsRead": false})
	sentByBob := false
	for _, item := range readMsg {
		m, _ := item.(map[string]any)
		if m["userId"] == bobID && m["isRead"] == true {
			sentByBob = true
		}
	}
	if !sentByBob {
		t.Errorf("bob's message never flipped to read")
	}

	// read of a message the caller sent is NO_SUCH_MESSAGE.
	firstID, _ := first["id"].(string)
	resp, out := postWithToken(t, f.ts, "/api/messaging/messages/read", aliceToken, map[string]any{"messageId": firstID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "NO_SUCH_MESSAGE" {
		t.Fatalf("self read: %d %v", resp.StatusCode, out)
	}
	// Reading bob's message explicitly is fine.
	thread = postListWithToken(t, f.ts, "/api/messaging/messages", aliceToken, map[string]any{"userId": bobID, "markAsRead": false})
	var bobMessageID string
	for _, item := range thread {
		m, _ := item.(map[string]any)
		if m["userId"] == bobID {
			bobMessageID, _ = m["id"].(string)
		}
	}
	if code := postNoContentToken(t, f.ts, "/api/messaging/messages/read", aliceToken, map[string]any{"messageId": bobMessageID}); code != http.StatusNoContent {
		t.Fatalf("explicit read: %d", code)
	}

	// Deletion: only the author deletes; the row goes hard.
	resp, out = postWithToken(t, f.ts, "/api/messaging/messages/delete", bobToken, map[string]any{"messageId": firstID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "NO_SUCH_MESSAGE" {
		t.Fatalf("recipient delete: %d %v", resp.StatusCode, out)
	}
	if code := postNoContentToken(t, f.ts, "/api/messaging/messages/delete", aliceToken, map[string]any{"messageId": firstID}); code != http.StatusNoContent {
		t.Fatalf("author delete: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/messaging/messages/read", aliceToken, map[string]any{"messageId": firstID})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("deleted message still readable: %d", resp.StatusCode)
	}

	// Blocked senders are refused with the fork-preserved id.
	if resp, bout := postWithToken(t, f.ts, "/api/blocking/create", bobToken, map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("block: %d %v", resp.StatusCode, bout)
	}
	out = sendAndExpect(t, f, aliceToken, map[string]any{"userId": bobID, "text": "still here"}, http.StatusBadRequest)
	if e, ok := out["error"].(map[string]any); !ok || e["id"] != "c15a5199-7422-4968-941a-2a462c478f7d" {
		t.Errorf("blocked id = %v", out["error"])
	}

	// mark-all-read clears the badge.
	_ = postListWithToken(t, f.ts, "/api/messaging/messages", bobToken, map[string]any{"userId": aliceID, "markAsRead": false})
	if code := postNoContentToken(t, f.ts, "/api/i/read-all-messaging-messages", aliceToken, nil); code != http.StatusNoContent {
		t.Fatalf("read-all: %d", code)
	}
	_, me = postWithToken(t, f.ts, "/api/i", aliceToken, nil)
	if me["hasUnreadMessagingMessage"] != false {
		t.Errorf("read-all left unread = %v", me["hasUnreadMessagingMessage"])
	}
}
