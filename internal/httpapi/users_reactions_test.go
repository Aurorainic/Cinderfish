package httpapi_test

import (
	"context"
	"net/http"
	"testing"
)

// TestUsersReactions covers the profile reaction tab: the reaction list, the
// publicReactions switch (default true, Misskey's default), self visibility,
// unknown and remote users.
func TestUsersReactions(t *testing.T) {
	ts, pool := newTestServerWithPool(t, "open")
	ownerID, ownerToken := createUser(t, ts, "owner")
	aliceID, aliceToken := createUser(t, ts, "alice")
	noteID := createNote(t, ts, ownerToken, map[string]any{"text": "react to me"})

	if resp, body := postWithToken(t, ts, "/api/notes/reactions/create", aliceToken, map[string]any{
		"noteId": noteID, "reaction": "👍",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("reaction: status %d body %v", resp.StatusCode, body)
	}

	// The switch defaults to public, matching Misskey.
	_, show := postWithToken(t, ts, "/api/users/show", aliceToken, map[string]any{"userId": aliceID})
	if show["publicReactions"] != true {
		t.Fatalf("publicReactions default = %v, want true", show["publicReactions"])
	}

	resp, raw := postRaw(t, ts, "/api/users/reactions", map[string]any{"userId": aliceID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/reactions: status %d body %v", resp.StatusCode, raw)
	}
	list, _ := raw.([]any)
	if len(list) != 1 {
		t.Fatalf("reactions = %v, want one", raw)
	}
	item, _ := list[0].(map[string]any)
	if item["type"] != "👍" {
		t.Errorf("type = %v, want 👍", item["type"])
	}
	if user, _ := item["user"].(map[string]any); user == nil || user["id"] != aliceID {
		t.Errorf("user = %v, want alice", item["user"])
	}
	if n, _ := item["note"].(map[string]any); n == nil || n["id"] != noteID {
		t.Errorf("note = %v, want the reacted note", item["note"])
	}
	if item["createdAt"] == "" {
		t.Error("createdAt is empty")
	}

	// Turning the switch off hides the list from others but not from alice.
	if resp, body := postWithToken(t, ts, "/api/i/update", aliceToken, map[string]any{
		"publicReactions": false,
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("i/update: status %d body %v", resp.StatusCode, body)
	}
	resp, body := post(t, ts, "/api/users/reactions", map[string]any{"userId": aliceID})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("anonymous status = %d, want 403 (%v)", resp.StatusCode, body)
	}
	if code, _ := body["error"].(map[string]any)["code"].(string); code != "REACTIONS_NOT_PUBLIC" {
		t.Errorf("code = %v, want REACTIONS_NOT_PUBLIC", code)
	}
	if resp, raw := postRawWithToken(t, ts, "/api/users/reactions", aliceToken, map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("self status = %d, want 200 (%v)", resp.StatusCode, raw)
	}

	// Unknown and remote accounts are refused with their own codes.
	if resp, _ := post(t, ts, "/api/users/reactions", map[string]any{"userId": "00000000000000000000"}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown user status = %d, want 404", resp.StatusCode)
	}
	if _, err := pool.Exec(context.Background(), `
INSERT INTO users (id, username, username_lower, host, uri)
VALUES ('w145r', 'w145r', 'w145r', 'remote.example', 'https://remote.example/users/r')
`); err != nil {
		t.Fatalf("insert remote: %v", err)
	}
	resp, body = post(t, ts, "/api/users/reactions", map[string]any{"userId": "w145r"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("remote status = %d, want 400 (%v)", resp.StatusCode, body)
	}
	if code, _ := body["error"].(map[string]any)["code"].(string); code != "IS_REMOTE_USER" {
		t.Errorf("remote code = %v, want IS_REMOTE_USER", code)
	}

	// The owner's own reaction list is independent of alice's.
	_, ownerList := postRaw(t, ts, "/api/users/reactions", map[string]any{"userId": ownerID})
	if items, _ := ownerList.([]any); len(items) != 0 {
		t.Errorf("owner reactions = %v, want none", ownerList)
	}
}
