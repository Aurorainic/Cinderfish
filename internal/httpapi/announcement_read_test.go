package httpapi_test

import (
	"net/http"
	"testing"
)

// TestAnnouncementReadState covers i/read-announcement: the public list flips
// isRead for the reader only, the unread badge follows, a repeat call is
// harmless, and the parameter is required.
func TestAnnouncementReadState(t *testing.T) {
	ts, _ := adminTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t)+"a")
	_, bobToken := createUser(t, ts, uniqueName(t)+"b")

	resp, created := postRawWithToken(t, ts, "/api/admin/announcements/create", "admintoken000000", map[string]any{
		"title": "read state", "text": "body",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create = %d %v", resp.StatusCode, created)
	}
	createdObj, _ := created.(map[string]any)
	announcementID, _ := createdObj["id"].(string)

	readFor := func(token string) (isRead bool, unread bool) {
		list := postListWithToken(t, ts, "/api/announcements", token, nil)
		if len(list) != 1 {
			t.Fatalf("announcements = %v, want one", list)
		}
		item, _ := list[0].(map[string]any)
		isRead, _ = item["isRead"].(bool)
		// The badge is a boolean in the MeDetailed schema; Misskey has no
		// announcement count field.
		_, me := post(t, ts, "/api/i", map[string]any{"i": token})
		unread, _ = me["hasUnreadAnnouncement"].(bool)
		return isRead, unread
	}

	if isRead, unread := readFor(aliceToken); isRead || !unread {
		t.Fatalf("before reading = isRead %v, unread %v; want false/true", isRead, unread)
	}

	// Marking it read answers 204 and flips both the flag and the badge.
	if status := postNoContentWithToken(t, ts, "/api/i/read-announcement", aliceToken, map[string]any{"announcementId": announcementID}); status != http.StatusNoContent {
		t.Fatalf("read-announcement = %d, want 204", status)
	}
	if isRead, unread := readFor(aliceToken); !isRead || unread {
		t.Errorf("after reading = isRead %v, unread %v; want true/false", isRead, unread)
	}

	// Reading it again is idempotent.
	if status := postNoContentWithToken(t, ts, "/api/i/read-announcement", aliceToken, map[string]any{"announcementId": announcementID}); status != http.StatusNoContent {
		t.Errorf("second read = %d, want 204", status)
	}

	// Another account still sees it unread.
	if isRead, unread := readFor(bobToken); isRead || !unread {
		t.Errorf("bob = isRead %v, unread %v; want false/true", isRead, unread)
	}

	// The parameter is required, and anonymous callers are refused.
	if status := postNoContentWithToken(t, ts, "/api/i/read-announcement", aliceToken, map[string]any{}); status != http.StatusBadRequest {
		t.Errorf("missing announcementId = %d, want 400", status)
	}
	if resp, _ := post(t, ts, "/api/i/read-announcement", map[string]any{"announcementId": announcementID}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", resp.StatusCode)
	}
}
