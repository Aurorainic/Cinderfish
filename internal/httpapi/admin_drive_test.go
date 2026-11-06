package httpapi_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"
)

func TestAdminShowUsersFilters(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	_, bob := post(t, ts, "/api/signup", map[string]any{"username": "bob", "password": "password123"})
	bobID, _ := bob["id"].(string)

	resp, out := postWithToken(t, ts, "/api/admin/suspend-user", "admintoken000000", map[string]any{"userId": bobID})
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Fatalf("suspend bob: status %d, body %v", resp.StatusCode, out)
	}

	// state=suspended returns only bob.
	resp, raw := postRawWithToken(t, ts, "/api/admin/show-users", "admintoken000000", map[string]any{"state": "suspended"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("show-users suspended: status %d, body %v", resp.StatusCode, raw)
	}
	arr, _ := raw.([]any)
	if len(arr) != 1 {
		t.Fatalf("suspended users = %d, want 1: %v", len(arr), arr)
	}
	if user, _ := arr[0].(map[string]any); user["username"] != "bob" {
		t.Errorf("suspended username = %v, want bob", user["username"])
	}

	// username is a prefix filter.
	resp, raw = postRawWithToken(t, ts, "/api/admin/show-users", "admintoken000000", map[string]any{"username": "al"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("show-users username: status %d, body %v", resp.StatusCode, raw)
	}
	arr, _ = raw.([]any)
	if len(arr) != 1 {
		t.Fatalf("username prefix hits = %d, want 1: %v", len(arr), arr)
	}

	// origin=remote has nothing on a single instance.
	resp, raw = postRawWithToken(t, ts, "/api/admin/show-users", "admintoken000000", map[string]any{"origin": "remote"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("show-users remote: status %d, body %v", resp.StatusCode, raw)
	}
	if arr, _ = raw.([]any); len(arr) != 0 {
		t.Errorf("remote users = %d, want 0", len(arr))
	}

}

func TestAdminDriveFiles(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	aliceToken, _ := alice["token"].(string)
	if aliceToken == "" {
		t.Fatalf("signup returned no token: %v", alice)
	}

	resp, file := uploadFile(t, ts.URL, aliceToken, "note.bin", bytes.NewBufferString("hello"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload: status %d, body %v", resp.StatusCode, file)
	}

	resp, raw := postRawWithToken(t, ts, "/api/admin/drive/files", "admintoken000000", map[string]any{"origin": "local"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin/drive/files: status %d, body %v", resp.StatusCode, raw)
	}
	arr, _ := raw.([]any)
	if len(arr) != 1 {
		t.Fatalf("local files = %d, want 1: %v", len(arr), arr)
	}
	row, _ := arr[0].(map[string]any)
	user, _ := row["user"].(map[string]any)
	if user["username"] != "alice" {
		t.Errorf("uploader = %v, want alice", user)
	}

	// Type filter: the upload is application/octet-stream.
	resp, raw = postRawWithToken(t, ts, "/api/admin/drive/files", "admintoken000000", map[string]any{"type": "application"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("type filter: status %d, body %v", resp.StatusCode, raw)
	}
	if arr, _ = raw.([]any); len(arr) != 1 {
		t.Errorf("application files = %d, want 1", len(arr))
	}
	resp, raw = postRawWithToken(t, ts, "/api/admin/drive/files", "admintoken000000", map[string]any{"type": "image"})
	if arr, _ = raw.([]any); len(arr) != 0 {
		t.Errorf("image files = %d, want 0", len(arr))
	}

	// Invalid type is rejected instead of reaching LIKE.
	resp, _ = postRawWithToken(t, ts, "/api/admin/drive/files", "admintoken000000", map[string]any{"type": "image' OR 1=1"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid type status = %d, want 400", resp.StatusCode)
	}

	// Non-admin gets 403.
	resp, _ = postRawWithToken(t, ts, "/api/admin/drive/files", aliceToken, map[string]any{})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin status = %d, want 403", resp.StatusCode)
	}
}

// TestAdminDriveFilters covers the filters the admin file list applies on top
// of origin/type: userId, hostname, and the id cursor.
func TestAdminDriveFilters(t *testing.T) {
	ts, srv := adminTestServer(t, "open")

	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	aliceToken, _ := alice["token"].(string)
	aliceID, _ := alice["id"].(string)
	_, bob := post(t, ts, "/api/signup", map[string]any{"username": "bob", "password": "password123"})
	bobToken, _ := bob["token"].(string)

	if resp, body := uploadFile(t, ts.URL, aliceToken, "a.txt", bytes.NewBufferString("aaa")); resp.StatusCode != http.StatusOK {
		t.Fatalf("alice upload: %d %v", resp.StatusCode, body)
	}
	if resp, body := uploadFile(t, ts.URL, bobToken, "b.txt", bytes.NewBufferString("bbb")); resp.StatusCode != http.StatusOK {
		t.Fatalf("bob upload: %d %v", resp.StatusCode, body)
	}

	// A remote account with a file, inserted directly: the drive API only
	// creates local uploads.
	pool := srv.TestPool()
	if _, err := pool.Exec(context.Background(), `
INSERT INTO users (id, username, username_lower, host, uri)
VALUES ('w63remote', 'remotee', 'remotee', 'remote.example', 'https://remote.example/users/remotee')`); err != nil {
		t.Fatalf("insert remote user: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
INSERT INTO drive_files (id, user_id, name, content_type, size, md5, storage_key, storage_backend, created_at)
VALUES ('w63remotefile', 'w63remote', 'r.txt', 'text/plain', 1, 'r', 'r/key', 'local', now() + interval '1 minute')`); err != nil {
		t.Fatalf("insert remote file: %v", err)
	}

	list := func(params map[string]any) []any {
		resp, raw := postRawWithToken(t, ts, "/api/admin/drive/files", "admintoken000000", params)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("admin/drive/files %v: %d %v", params, resp.StatusCode, raw)
		}
		arr, _ := raw.([]any)
		return arr
	}

	if got := list(map[string]any{"origin": "combined"}); len(got) != 3 {
		t.Fatalf("combined files = %d, want 3", len(got))
	}
	if got := list(map[string]any{"userId": aliceID}); len(got) != 1 {
		t.Errorf("userId files = %d, want 1", len(got))
	}
	if got := list(map[string]any{"origin": "remote"}); len(got) != 1 {
		t.Errorf("remote files = %d, want 1", len(got))
	}
	if got := list(map[string]any{"hostname": "REMOTE.EXAMPLE"}); len(got) != 1 {
		t.Errorf("hostname files = %d, want 1 (the filter folds case)", len(got))
	}
	if got := list(map[string]any{"hostname": "other.example"}); len(got) != 0 {
		t.Errorf("hostname other = %d, want 0", len(got))
	}

	// The newest file is the remote one (created a minute ahead), so an untilId
	// cursor at its id keeps the two local uploads.
	if got := list(map[string]any{"untilId": "w63remotefile"}); len(got) != 2 {
		t.Errorf("untilId files = %d, want 2", len(got))
	}
	// sinceId walks the other way and finds only newer rows.
	if got := list(map[string]any{"sinceId": "w63remotefile"}); len(got) != 0 {
		t.Errorf("sinceId files = %d, want 0", len(got))
	}
	// Both cursors together bound the page.
	if got := list(map[string]any{"sinceId": "w63remotefile", "untilId": "aaaaaaaaaaaa"}); len(got) != 0 {
		t.Errorf("bounded cursor files = %d, want 0", len(got))
	}
}
