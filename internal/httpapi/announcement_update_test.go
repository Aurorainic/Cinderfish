package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestAnnouncementUpdate covers admin/announcements/update: partial updates keep
// the other fields, explicit null clears imageUrl and expiresAt, isActive hides
// the notice from the public list, and the validation branches.
func TestAnnouncementUpdate(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	future := time.Now().Add(time.Hour).UnixMilli()
	resp, created := postRawWithToken(t, ts, "/api/admin/announcements/create", "admintoken000000", map[string]any{
		"title": "before", "text": "first body",
		"imageUrl": "https://cdn.example/a.png", "expiresAt": future,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create = %d %v", resp.StatusCode, created)
	}
	obj, _ := created.(map[string]any)
	id, _ := obj["id"].(string)

	// Update answers 204, so every assertion re-reads the admin list.
	update := func(body map[string]any) {
		t.Helper()
		if status := postNoContentWithToken(t, ts, "/api/admin/announcements/update", "admintoken000000", body); status != http.StatusNoContent {
			t.Fatalf("update %v = %d, want 204", body, status)
		}
	}
	adminRow := func() map[string]any {
		t.Helper()
		list := postListWithToken(t, ts, "/api/admin/announcements/list", "admintoken000000", nil)
		if len(list) != 1 {
			t.Fatalf("admin list = %v, want one row", list)
		}
		row, _ := list[0].(map[string]any)
		return row
	}

	// A partial update touches only the title.
	update(map[string]any{"id": id, "title": "after"})
	got := adminRow()
	if got["title"] != "after" || got["text"] != "first body" || got["imageUrl"] != "https://cdn.example/a.png" {
		t.Errorf("partial update = %v", got)
	}
	if got["expiresAt"] == nil {
		t.Errorf("expiresAt was dropped by a partial update: %v", got)
	}

	// Null clears the image and the expiry.
	update(map[string]any{"id": id, "imageUrl": nil, "expiresAt": nil})
	got = adminRow()
	if got["imageUrl"] != nil || got["expiresAt"] != nil {
		t.Errorf("cleared fields = %v/%v", got["imageUrl"], got["expiresAt"])
	}

	// Deactivating hides it from the public list but keeps it in the admin list.
	update(map[string]any{"id": id, "isActive": false})
	if got := adminRow(); got["isActive"] != false {
		t.Errorf("isActive = %v, want false", got["isActive"])
	}
	if public := postListWithToken(t, ts, "/api/announcements", "", nil); len(public) != 0 {
		t.Errorf("inactive announcement visible to the public: %v", public)
	}
	if admin := postListWithToken(t, ts, "/api/admin/announcements/list", "admintoken000000", nil); len(admin) != 1 {
		t.Errorf("admin list = %v, want the inactive row", admin)
	}

	// An expiry in the past hides it again once reactivated.
	update(map[string]any{"id": id, "isActive": true, "expiresAt": time.Now().Add(-time.Hour).UnixMilli()})
	if public := postListWithToken(t, ts, "/api/announcements", "", nil); len(public) != 0 {
		t.Errorf("expired announcement visible to the public: %v", public)
	}

	// Validation and permissions.
	for _, body := range []map[string]any{
		{"id": id, "title": ""},
		{"id": id, "text": ""},
		{"id": id, "text": strings.Repeat("x", 10001)},
		{"id": id, "imageUrl": "ftp://cdn.example/a.png"},
	} {
		if resp, _ := postRawWithToken(t, ts, "/api/admin/announcements/update", "admintoken000000", body); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("invalid update %v = %d, want 400", body, resp.StatusCode)
		}
	}
	if resp, _ := postRawWithToken(t, ts, "/api/admin/announcements/update", "user-token-not-admin", map[string]any{"id": id, "title": "x"}); resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("non-admin = %d, want 403/401", resp.StatusCode)
	}
}
