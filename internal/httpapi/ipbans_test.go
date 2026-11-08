package httpapi_test

import (
	"net/http"
	"testing"
)

// TestIPBansAndSigninAudit covers the Cinderfish IP-ban extension and the
// cross-account signin audit: CRUD, enforcement on signin and signup, and
// the audit rows the admin view reads.
func TestIPBansAndSigninAudit(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	aliceToken, _ := alice["token"].(string)
	if aliceToken == "" {
		t.Fatalf("alice signup failed: %v", alice)
	}

	// A non-admin cannot manage bans.
	if resp, _ := postRawWithToken(t, ts, "/api/admin/ip-bans/list", aliceToken, nil); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-admin list = %d, want 403", resp.StatusCode)
	}

	// Create a ban for the test client's own address (httptest uses 127.0.0.1).
	resp, raw := postRawWithToken(t, ts, "/api/admin/ip-bans/create", "admintoken000000", map[string]any{
		"ip": "127.0.0.1", "reason": "test ban",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create ban = %d, body %v", resp.StatusCode, raw)
	}
	created, _ := raw.(map[string]any)
	banID, _ := created["id"].(string)
	if created["ip"] != "127.0.0.1" || created["reason"] != "test ban" {
		t.Errorf("created ban = %v", created)
	}
	if creator, _ := created["createdBy"].(map[string]any); creator == nil || creator["username"] != "root" {
		t.Errorf("createdBy = %v, want root", created["createdBy"])
	}

	// Signin and signup from the banned address are refused.
	resp, blocked := post(t, ts, "/api/signin", map[string]any{"username": "alice", "password": "password123"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("banned signin = %d, want 403", resp.StatusCode)
	}
	if errBody, _ := blocked["error"].(map[string]any); errBody["code"] != "IP_BANNED" {
		t.Errorf("banned signin error = %v, want IP_BANNED", blocked["error"])
	}
	if resp, _ := post(t, ts, "/api/signup", map[string]any{"username": "bob", "password": "password123"}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("banned signup = %d, want 403", resp.StatusCode)
	}

	// Invalid rules are refused.
	if resp, _ := postRawWithToken(t, ts, "/api/admin/ip-bans/create", "admintoken000000", map[string]any{
		"ip": "not-an-ip",
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid ip = %d, want 400", resp.StatusCode)
	}

	// The admin API itself stays reachable from the banned address: only
	// signin/signup enforce the rule.
	list := postListWithToken(t, ts, "/api/admin/ip-bans/list", "admintoken000000", nil)
	if len(list) != 1 {
		t.Fatalf("bans = %d, want 1", len(list))
	}

	// Deleting the ban restores signin; a wrong password then records a
	// failed attempt for the audit view.
	if resp, _ := postRawWithToken(t, ts, "/api/admin/ip-bans/delete", "admintoken000000", map[string]any{"id": banID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("delete ban = %d, want 200", resp.StatusCode)
	}
	if resp, body := post(t, ts, "/api/signin", map[string]any{"username": "alice", "password": "password123"}); resp.StatusCode != http.StatusOK {
		t.Fatalf("signin after unban = %d, want 200 (body %v)", resp.StatusCode, body)
	}
	if resp, _ := post(t, ts, "/api/signin", map[string]any{"username": "alice", "password": "wrong-password"}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("wrong password = %d, want 403", resp.StatusCode)
	}

	// The audit view lists both attempts with the account attached.
	rows := postListWithToken(t, ts, "/api/admin/signin-history", "admintoken000000", map[string]any{"limit": 50})
	if len(rows) < 2 {
		t.Fatalf("audit rows = %d, want at least 2", len(rows))
	}
	var sawSuccess, sawFailure bool
	for _, row := range rows {
		entry, _ := row.(map[string]any)
		user, _ := entry["user"].(map[string]any)
		if user == nil || user["username"] != "alice" {
			continue
		}
		if entry["success"] == true {
			sawSuccess = true
		} else {
			sawFailure = true
		}
		if entry["ip"] != "127.0.0.1" {
			t.Errorf("audit ip = %v, want 127.0.0.1", entry["ip"])
		}
	}
	if !sawSuccess || !sawFailure {
		t.Errorf("audit rows missing outcomes: success=%v failure=%v", sawSuccess, sawFailure)
	}

	// An unknown ban id answers 404.
	if resp, _ := postRawWithToken(t, ts, "/api/admin/ip-bans/delete", "admintoken000000", map[string]any{"id": "missing"}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("delete missing ban = %d, want 404", resp.StatusCode)
	}
}
