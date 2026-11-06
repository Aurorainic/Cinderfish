package httpapi_test

import (
	"net/http"
	"testing"
)

// TestNotificationsMarkAsRead marks one notification through the REST endpoint
// and leaves the other unread.
func TestNotificationsMarkAsRead(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, "alice")
	bobID, bobToken := createUser(t, ts, "bob")

	// Two mentions from bob give alice two notifications.
	for i := 0; i < 2; i++ {
		createNote(t, ts, bobToken, map[string]any{"text": "hello @alice " + string(rune('a'+i))})
	}
	list := postListWithToken(t, ts, "/api/notifications", aliceToken, map[string]any{"markAsRead": false})
	if len(list) != 2 {
		t.Fatalf("notifications = %d, want 2", len(list))
	}
	first, _ := list[0].(map[string]any)
	second, _ := list[1].(map[string]any)
	firstID, _ := first["id"].(string)
	if firstID == "" {
		t.Fatalf("first notification has no id: %v", first)
	}
	_ = bobID

	status := postNoContentWithToken(t, ts, "/api/notifications/mark-as-read", aliceToken, map[string]any{
		"notificationId": firstID,
	})
	if status != http.StatusNoContent {
		t.Fatalf("mark-as-read status = %d, want 204", status)
	}

	list = postListWithToken(t, ts, "/api/notifications", aliceToken, map[string]any{"markAsRead": false})
	read := map[string]bool{}
	for _, raw := range list {
		item, _ := raw.(map[string]any)
		id, _ := item["id"].(string)
		isRead, _ := item["isRead"].(bool)
		read[id] = isRead
	}
	if !read[firstID] {
		t.Errorf("notification %s is still unread", firstID)
	}
	secondID, _ := second["id"].(string)
	if secondID != "" && secondID != firstID && read[secondID] {
		t.Errorf("the second notification was marked read too")
	}

	// Missing id is a client error; an unknown id is a no-op 204.
	if status := postNoContentWithToken(t, ts, "/api/notifications/mark-as-read", aliceToken, map[string]any{}); status != http.StatusBadRequest {
		t.Errorf("missing id status = %d, want 400", status)
	}
	if status := postNoContentWithToken(t, ts, "/api/notifications/mark-as-read", aliceToken, map[string]any{
		"notificationId": "00000000000000000000",
	}); status != http.StatusNoContent {
		t.Errorf("unknown id status = %d, want 204", status)
	}
}
