package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestAdminDatabaseStats(t *testing.T) {
	ts, _ := adminTestServer(t, "open")
	post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})

	resp, raw := postRawWithToken(t, ts, "/api/admin/get-table-stats", "admintoken000000", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("table stats: status %d, body %v", resp.StatusCode, raw)
	}
	tables, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("table stats type = %T, want object", raw)
	}
	for _, name := range []string{"users", "notes", "webhooks"} {
		entry, _ := tables[name].(map[string]any)
		if entry == nil {
			t.Errorf("table %q missing from stats", name)
			continue
		}
		if _, ok := entry["size"].(float64); !ok {
			t.Errorf("table %q has no numeric size: %v", name, entry)
		}
		if _, ok := entry["count"].(float64); !ok {
			t.Errorf("table %q has no numeric count: %v", name, entry)
		}
	}

	resp, raw = postRawWithToken(t, ts, "/api/admin/get-index-stats", "admintoken000000", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("index stats: status %d, body %v", resp.StatusCode, raw)
	}
	indexes, _ := raw.([]any)
	if len(indexes) == 0 {
		t.Fatalf("index stats is empty")
	}
	found := false
	for _, item := range indexes {
		row, _ := item.(map[string]any)
		def, _ := row["indexdef"].(string)
		if strings.Contains(def, "users") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no index mentions users: %v", indexes[:min(3, len(indexes))])
	}

	// A normal account is refused.
	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "bob", "password": "password123"})
	bobToken, _ := alice["token"].(string)
	resp, _ = postRawWithToken(t, ts, "/api/admin/get-table-stats", bobToken, map[string]any{})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin status = %d, want 403", resp.StatusCode)
	}
	resp, _ = postRaw(t, ts, "/api/admin/get-index-stats", map[string]any{})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous status = %d, want 401", resp.StatusCode)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
