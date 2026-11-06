package httpapi_test

import (
	"net/http"
	"testing"
)

func TestUsersDirectoryExcludesSuspended(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	_, bob := post(t, ts, "/api/signup", map[string]any{"username": "bob", "password": "password123"})
	bobID, _ := bob["id"].(string)

	resp, raw := postRaw(t, ts, "/api/users", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users: status %d, body %v", resp.StatusCode, raw)
	}
	// root (the test administrator) + alice + bob.
	if arr, _ := raw.([]any); len(arr) != 3 {
		t.Fatalf("users before suspend = %d, want 3: %v", len(arr), arr)
	}

	if status := postNoContentWithToken(t, ts, "/api/admin/suspend-user", "admintoken000000", map[string]any{"userId": bobID}); status != http.StatusOK && status != http.StatusNoContent {
		t.Fatalf("suspend status = %d", status)
	}
	resp, raw = postRaw(t, ts, "/api/users", map[string]any{})
	arr, _ := raw.([]any)
	if len(arr) != 2 {
		t.Fatalf("users after suspend = %d, want 2: %v", len(arr), arr)
	}
	for _, item := range arr {
		row, _ := item.(map[string]any)
		if row["username"] == "bob" {
			t.Errorf("suspended bob is still listed: %v", row["username"])
		}
	}
}

func TestUsersDirectorySortAndFlags(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")
	f.registerUser(t, "carol")

	// Two followers for alice, none for the others.
	for _, token := range []string{bobToken} {
		if resp, out := postWithToken(t, f.ts, "/api/following/create", token, map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusOK {
			t.Fatalf("follow: status %d body %v", resp.StatusCode, out)
		}
	}

	// Sort by followers puts alice first; anonymous reads work.
	rows := searchList(t, f, "/api/users", map[string]any{"sort": "+follower", "origin": "local"})
	if len(rows) != 3 {
		t.Fatalf("local users = %d, want 3: %v", len(rows), rows)
	}
	first, _ := rows[0].(map[string]any)
	if first["username"] != "alice" {
		t.Errorf("first by follower = %v, want alice", first["username"])
	}

	// Remote origin has nothing on a single instance.
	if rows := searchList(t, f, "/api/users", map[string]any{"origin": "remote"}); len(rows) != 0 {
		t.Errorf("remote users = %d, want 0", len(rows))
	}

	// A signed-in viewer gets follow flags.
	rows = searchList(t, f, "/api/users", map[string]any{"i": bobToken, "origin": "local"})
	for _, item := range rows {
		row, _ := item.(map[string]any)
		if row["username"] == "alice" && row["isFollowing"] != true {
			t.Errorf("bob -> alice isFollowing = %v, want true", row["isFollowing"])
		}
	}

	// Unknown state is rejected instead of ignored.
	resp, _ := postRaw(t, f.ts, "/api/users", map[string]any{"state": "bogus"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad state status = %d, want 400", resp.StatusCode)
	}
	_ = aliceToken
}
