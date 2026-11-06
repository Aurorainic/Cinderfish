package httpapi_test

import (
	"bytes"
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
