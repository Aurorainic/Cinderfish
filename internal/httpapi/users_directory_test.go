package httpapi_test

import (
	"context"
	"net/http"
	"testing"
)

// TestUsersDirectoryFilters covers the public users directory: guests can read
// it, the local origin is the default, suspended accounts disappear, and the
// state/origin/hostname/sort parameters narrow the list.
func TestUsersDirectoryFilters(t *testing.T) {
	ts, srv := adminTestServer(t, "open")
	pool := srv.TestPool()

	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	aliceID, _ := alice["id"].(string)
	post(t, ts, "/api/signup", map[string]any{"username": "bob", "password": "password123"})
	if _, err := pool.Exec(context.Background(), `
INSERT INTO users (id, username, username_lower, host, uri)
VALUES ('w63remoteu', 'remotee', 'remotee', 'remote.example', 'https://remote.example/users/remotee')`); err != nil {
		t.Fatalf("insert remote user: %v", err)
	}

	names := func(params map[string]any) []string {
		resp, raw := postRaw(t, ts, "/api/users", params)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("users %v: %d %v", params, resp.StatusCode, raw)
		}
		arr, _ := raw.([]any)
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			u, _ := item.(map[string]any)
			name, _ := u["username"].(string)
			out = append(out, name)
		}
		return out
	}

	// Guests read the directory; the default origin is local.
	local := names(map[string]any{})
	if len(local) != 3 {
		t.Fatalf("local users = %v, want root/alice/bob", local)
	}
	for _, name := range local {
		if name == "remotee" {
			t.Error("the default origin must stay local")
		}
	}

	// state=admin maps onto administrators only.
	admins := names(map[string]any{"state": "admin"})
	if len(admins) != 1 || admins[0] != "root" {
		t.Errorf("state=admin = %v, want [root]", admins)
	}

	// origin=remote and hostname select the remote account.
	if got := names(map[string]any{"origin": "remote"}); len(got) != 1 || got[0] != "remotee" {
		t.Errorf("origin=remote = %v", got)
	}
	// hostname combines with origin, as in Misskey: the default local origin
	// would contradict a remote host, so the client asks for combined.
	if got := names(map[string]any{"origin": "combined", "hostname": "REMOTE.EXAMPLE"}); len(got) != 1 {
		t.Errorf("hostname match = %v, want the remote user", got)
	}
	if got := names(map[string]any{"origin": "combined", "hostname": "other.example"}); len(got) != 0 {
		t.Errorf("hostname other = %v, want none", got)
	}
	// state=alive has no data to filter on here (users carry no updatedAt), so
	// it behaves as all.
	if got := names(map[string]any{"state": "alive"}); len(got) != len(local) {
		t.Errorf("state=alive = %v, want the same list as all", got)
	}

	// Newest first by default; the oldest first with -createdAt.
	if got := names(map[string]any{"sort": "+createdAt"}); len(got) < 3 || got[0] != "bob" {
		t.Errorf("+createdAt = %v, want bob first", got)
	}
	if got := names(map[string]any{"sort": "-createdAt"}); len(got) < 3 || got[0] != "root" {
		t.Errorf("-createdAt = %v, want root first", got)
	}

	// Suspending the account removes it from the public directory.
	if resp, body := postWithToken(t, ts, "/api/admin/suspend-user", "admintoken000000", map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Fatalf("suspend: %d %v", resp.StatusCode, body)
	}
	for _, name := range names(map[string]any{}) {
		if name == "alice" {
			t.Error("a suspended account is still listed")
		}
	}

	// Unknown values are parameter errors.
	for _, params := range []map[string]any{{"state": "ghost"}, {"origin": "galaxy"}} {
		if resp, _ := postRaw(t, ts, "/api/users", params); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("users %v status = %d, want 400", params, resp.StatusCode)
		}
	}
}
