package httpapi_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const emojiAdminToken = "admintokenemojix"

// newEmojiAdminServer returns a server with a seeded administrator, so the
// emoji write endpoints have a caller.
func newEmojiAdminServer(t *testing.T) *httptest.Server {
	t.Helper()
	ts := newTestServer(t, "open")
	// testPool() truncates on every call, so the admin row goes in through a
	// plain pool over the same database instead of testPool(t) again.
	url := os.Getenv("CINDERFISH_TEST_DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(context.Background(), `
INSERT INTO users (id, username, username_lower, password_hash, token, is_admin)
VALUES ('rootemoji', 'rootemoji', 'rootemoji', 'x', $1, TRUE)`, emojiAdminToken); err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	return ts
}

// uploadEmojiFile stores a small PNG-ish blob through the drive endpoint and
// returns its id.
func uploadEmojiFile(t *testing.T, ts *httptest.Server, token, name string) string {
	t.Helper()
	resp, file := uploadFile(t, ts.URL, token, name, bytes.NewBufferString("png-bytes"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload: %d %v", resp.StatusCode, file)
	}
	id, _ := file["id"].(string)
	if id == "" {
		t.Fatalf("upload produced no id: %v", file)
	}
	return id
}

// TestEmojisRegistryLifecycle pins admin/emoji/add → emojis → admin/emoji/list
// → admin/emoji/delete: the public endpoint lists local emoji in the
// EmojiSimple shape, the admin list carries ids, and a duplicate name is
// refused.
func TestEmojisRegistryLifecycle(t *testing.T) {
	ts := newEmojiAdminServer(t)
	adminToken := emojiAdminToken

	// The admin uploads a file to point the emoji at.
	fileID := uploadEmojiFile(t, ts, adminToken, "blobcat.png")

	resp, emoji := post(t, ts, "/api/admin/emoji/add", map[string]any{
		"i": adminToken, "name": "blobcat", "fileId": fileID,
		"category": "cats", "aliases": []string{"blob_cat"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin/emoji/add: %d %v", resp.StatusCode, emoji)
	}
	emojiID, _ := emoji["id"].(string)
	if emojiID == "" || emoji["name"] != "blobcat" || emoji["host"] != nil {
		t.Fatalf("added emoji = %v", emoji)
	}

	_, public := post(t, ts, "/api/emojis", map[string]any{})
	wrapper, _ := public["emojis"].([]any)
	if len(wrapper) != 1 {
		t.Fatalf("public emojis = %v, want one", public)
	}
	item, _ := wrapper[0].(map[string]any)
	if item["name"] != "blobcat" || item["category"] != "cats" {
		t.Errorf("public emoji = %v", item)
	}
	if aliases, _ := item["aliases"].([]any); len(aliases) != 1 || aliases[0] != "blob_cat" {
		t.Errorf("aliases = %v, want blob_cat", item["aliases"])
	}
	if item["url"] == nil || item["url"] == "" {
		t.Errorf("emoji url empty: %v", item)
	}

	if list := postListWithToken(t, ts, "/api/admin/emoji/list", adminToken, nil); len(list) != 1 {
		t.Fatalf("admin list = %v, want one", list)
	}

	// A duplicate name is a client error, and an invalid name never lands.
	resp, dup := post(t, ts, "/api/admin/emoji/add", map[string]any{"i": adminToken, "name": "blobcat", "fileId": fileID})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("duplicate add = %d %v, want 400", resp.StatusCode, dup)
	}
	if e, _ := dup["error"].(map[string]any); e["code"] != "EMOJI_ALREADY_EXISTS" {
		t.Errorf("duplicate code = %v", e["code"])
	}
	if resp, _ := post(t, ts, "/api/admin/emoji/add", map[string]any{"i": adminToken, "name": "bad name!", "fileId": fileID}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid name = %d, want 400", resp.StatusCode)
	}

	if resp, out := post(t, ts, "/api/admin/emoji/delete", map[string]any{"i": adminToken, "id": emojiID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("delete: %d %v", resp.StatusCode, out)
	}
	if _, public := post(t, ts, "/api/emojis", map[string]any{}); len(public["emojis"].([]any)) != 0 {
		t.Errorf("emojis after delete = %v", public)
	}
}

// TestEmojisAdminOnly proves the write endpoints are administrator-only and
// the public read needs no credential.
func TestEmojisAdminOnly(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))
	fileID := uploadEmojiFile(t, ts, token, "blob.png")

	if resp, _ := post(t, ts, "/api/admin/emoji/add", map[string]any{"i": token, "name": "x", "fileId": fileID}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin add = %d, want 403", resp.StatusCode)
	}
	if resp, _ := post(t, ts, "/api/admin/emoji/list", map[string]any{"i": token}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin list = %d, want 403", resp.StatusCode)
	}
	if resp, _ := post(t, ts, "/api/emojis", map[string]any{}); resp.StatusCode != http.StatusOK {
		t.Errorf("anonymous emojis = %d, want 200", resp.StatusCode)
	}
}
