package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAnnouncementsLifecycle(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	resp, raw := postRawWithToken(t, ts, "/api/admin/announcements/create", "admintoken000000", map[string]any{
		"title": "维护通知", "text": "今晚 23:00 重启实例。",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: status %d, body %v", resp.StatusCode, raw)
	}
	created, _ := raw.(map[string]any)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("create returned no id: %v", created)
	}
	if created["isRead"] != false {
		t.Errorf("isRead = %v, want false", created["isRead"])
	}

	// Anonymous visitors see active announcements.
	resp, raw = postRaw(t, ts, "/api/announcements", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public list: status %d, body %v", resp.StatusCode, raw)
	}
	arr, _ := raw.([]any)
	if len(arr) != 1 {
		t.Fatalf("public announcements = %d, want 1: %v", len(arr), arr)
	}

	// The admin list shows the same row.
	resp, raw = postRawWithToken(t, ts, "/api/admin/announcements/list", "admintoken000000", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin list: status %d, body %v", resp.StatusCode, raw)
	}
	if arr, _ = raw.([]any); len(arr) != 1 {
		t.Fatalf("admin announcements = %d, want 1", len(arr))
	}

	// Delete removes it from both lists.
	if status := postNoContentWithToken(t, ts, "/api/admin/announcements/delete", "admintoken000000", map[string]any{"id": id}); status != http.StatusNoContent {
		t.Fatalf("delete: status %d, want 204", status)
	}
	resp, raw = postRaw(t, ts, "/api/announcements", map[string]any{})
	if arr, _ = raw.([]any); len(arr) != 0 {
		t.Errorf("public announcements after delete = %d, want 0", len(arr))
	}
}

func TestAnnouncementsValidationAndPermissions(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	// Missing title / text are rejected.
	resp, _ := postRawWithToken(t, ts, "/api/admin/announcements/create", "admintoken000000", map[string]any{"title": "", "text": "x"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty title status = %d, want 400", resp.StatusCode)
	}
	// A past expiry is rejected.
	resp, _ = postRawWithToken(t, ts, "/api/admin/announcements/create", "admintoken000000", map[string]any{
		"title": "t", "text": "x", "expiresAt": time.Now().Add(-time.Minute).UnixMilli(),
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("past expiry status = %d, want 400", resp.StatusCode)
	}
	// A javascript: image URL is rejected.
	resp, _ = postRawWithToken(t, ts, "/api/admin/announcements/create", "admintoken000000", map[string]any{
		"title": "t", "text": "x", "imageUrl": "javascript:alert(1)",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("javascript image status = %d, want 400", resp.StatusCode)
	}

	// A normal account cannot create or list.
	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	aliceToken, _ := alice["token"].(string)
	resp, _ = postRawWithToken(t, ts, "/api/admin/announcements/create", aliceToken, map[string]any{"title": "t", "text": "x"})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin create status = %d, want 403", resp.StatusCode)
	}
	resp, _ = postRawWithToken(t, ts, "/api/admin/announcements/list", aliceToken, map[string]any{})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin list status = %d, want 403", resp.StatusCode)
	}
}

func TestAnnouncementsExpiryHidesFromPublicList(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	expires := time.Now().Add(400 * time.Millisecond).UnixMilli()
	resp, raw := postRawWithToken(t, ts, "/api/admin/announcements/create", "admintoken000000", map[string]any{
		"title": "短期公告", "text": "马上过期", "expiresAt": expires,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: status %d, body %v", resp.StatusCode, raw)
	}
	time.Sleep(700 * time.Millisecond)

	resp, raw = postRaw(t, ts, "/api/announcements", map[string]any{})
	arr, _ := raw.([]any)
	if len(arr) != 0 {
		t.Errorf("expired announcement still public: %v", arr)
	}
	// The admin list still shows it, expired or not.
	resp, raw = postRawWithToken(t, ts, "/api/admin/announcements/list", "admintoken000000", map[string]any{})
	if arr, _ = raw.([]any); len(arr) != 1 {
		t.Errorf("admin list after expiry = %d, want 1", len(arr))
	}
}

// postNoContentWithToken posts a JSON body with a bearer token and returns the
// status without decoding, for the 204 endpoints.
func postNoContentWithToken(t *testing.T, ts *httptest.Server, path, token string, body map[string]any) int {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("post %s: %v", path, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// TestAnnouncementsPagination covers the limit and untilId cursor: the list is
// newest first, a short limit pages through it, and the service caps an
// oversized limit at 100.
func TestAnnouncementsPagination(t *testing.T) {
	ts, srv := adminTestServer(t, "open")

	ids := make([]string, 0, 3)
	for _, title := range []string{"one", "two", "three"} {
		resp, raw := postRawWithToken(t, ts, "/api/admin/announcements/create", "admintoken000000", map[string]any{
			"title": title, "text": "body " + title,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("create %s: %d %v", title, resp.StatusCode, raw)
		}
		created, _ := raw.(map[string]any)
		ids = append(ids, created["id"].(string))
	}

	list := func(params map[string]any) []any {
		resp, raw := postRaw(t, ts, "/api/announcements", params)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("announcements %v: %d %v", params, resp.StatusCode, raw)
		}
		arr, _ := raw.([]any)
		return arr
	}

	// Newest first.
	all := list(map[string]any{})
	if len(all) != 3 {
		t.Fatalf("all = %d, want 3", len(all))
	}
	if got, _ := all[0].(map[string]any); got["id"] != ids[2] {
		t.Errorf("first = %v, want the newest %s", got["id"], ids[2])
	}

	// limit=2 keeps the two newest; the cursor continues from the second.
	page := list(map[string]any{"limit": 2})
	if len(page) != 2 {
		t.Fatalf("limit 2 = %d rows, want 2", len(page))
	}
	if got, _ := page[1].(map[string]any); got["id"] != ids[1] {
		t.Errorf("page[1] = %v, want %s", got["id"], ids[1])
	}
	rest := list(map[string]any{"untilId": ids[1]})
	if len(rest) != 1 {
		t.Fatalf("untilId page = %d rows, want 1", len(rest))
	}
	if got, _ := rest[0].(map[string]any); got["id"] != ids[0] {
		t.Errorf("untilId page = %v, want %s", got["id"], ids[0])
	}

	// An oversized limit is capped at 100 rather than failing: seed enough
	// rows to observe the cap without going through the API 100 times.
	pool := srv.TestPool()
	if _, err := pool.Exec(context.Background(), `
INSERT INTO announcements (id, title, text)
SELECT 'w63cap' || lpad(g::text, 4, '0'), 'cap ' || g, 'body'
FROM generate_series(1, 150) AS g`); err != nil {
		t.Fatalf("seed announcements: %v", err)
	}
	capped := list(map[string]any{"limit": 1000})
	if len(capped) != 100 {
		t.Errorf("limit 1000 = %d rows, want the 100 cap", len(capped))
	}
}
