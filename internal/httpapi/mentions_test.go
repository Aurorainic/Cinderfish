package httpapi_test

import (
	"net/http"
	"testing"
)

func TestNotesMentions(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")
	carolToken, _ := f.registerUser(t, "carol")

	createSearchNote(t, f, aliceToken, "hello @bob check this", "public")

	// The mentioned user sees it.
	hits := searchList(t, f, "/api/notes/mentions", map[string]any{"i": bobToken})
	if len(hits) != 1 {
		t.Fatalf("bob mentions = %d, want 1: %v", len(hits), hits)
	}
	// Someone else does not.
	if hits := searchList(t, f, "/api/notes/mentions", map[string]any{"i": carolToken}); len(hits) != 0 {
		t.Errorf("carol mentions = %d, want 0", len(hits))
	}
	// Anonymous is refused: mentions are personal.
	resp, _ := postRaw(t, f.ts, "/api/notes/mentions", map[string]any{})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous mentions status = %d, want 401", resp.StatusCode)
	}

	// following=true hides notes from accounts bob does not follow.
	hits = searchList(t, f, "/api/notes/mentions", map[string]any{"i": bobToken, "following": true})
	if len(hits) != 0 {
		t.Errorf("following mentions before follow = %d, want 0", len(hits))
	}
	if resp, out := postWithToken(t, f.ts, "/api/following/create", bobToken, map[string]any{
		"userId": userID(t, f, "alice"),
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("follow: status %d body %v", resp.StatusCode, out)
	}
	hits = searchList(t, f, "/api/notes/mentions", map[string]any{"i": bobToken, "following": true})
	if len(hits) != 1 {
		t.Errorf("following mentions after follow = %d, want 1", len(hits))
	}
	_ = bobID
}

func TestNotificationsUnreadOnly(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")

	createSearchNote(t, f, aliceToken, "mention for unread @bob", "public")

	if hits := searchList(t, f, "/api/i/notifications", map[string]any{"i": bobToken, "unreadOnly": true}); len(hits) != 1 {
		t.Fatalf("unread notifications = %d, want 1: %v", len(hits), hits)
	}

	// Mark everything read; the unread view empties, the full view does not.
	if status := postNoContentWithToken(t, f.ts, "/api/notifications/mark-all-as-read", bobToken, map[string]any{}); status != http.StatusOK && status != http.StatusNoContent {
		t.Fatalf("mark-all-as-read status = %d", status)
	}
	if hits := searchList(t, f, "/api/i/notifications", map[string]any{"i": bobToken, "unreadOnly": true}); len(hits) != 0 {
		t.Errorf("unread after mark = %d, want 0", len(hits))
	}
	if hits := searchList(t, f, "/api/i/notifications", map[string]any{"i": bobToken}); len(hits) != 1 {
		t.Errorf("all after mark = %d, want 1", len(hits))
	}
}
