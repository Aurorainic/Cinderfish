package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// searchList posts a search endpoint and returns the decoded array.
func searchList(t *testing.T, f *fedInstance, path string, body map[string]any) []any {
	t.Helper()
	resp, raw := postRaw(t, f.ts, path, body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: status %d, body %v", path, resp.StatusCode, raw)
	}
	arr, ok := raw.([]any)
	if !ok {
		t.Fatalf("%s returned %T, want an array: %v", path, raw, raw)
	}
	return arr
}

func createSearchNote(t *testing.T, f *fedInstance, token, text, visibility string) string {
	t.Helper()
	resp, out := postWithToken(t, f.ts, "/api/notes/create", token, map[string]any{
		"text": text, "visibility": visibility,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("notes/create: status %d, body %v", resp.StatusCode, out)
	}
	note, _ := out["createdNote"].(map[string]any)
	id, _ := note["id"].(string)
	if id == "" {
		t.Fatalf("notes/create returned no id: %v", out)
	}
	return id
}

func TestNotesSearchRespectsVisibility(t *testing.T) {
	f := newFederationServer(t, "public")
	token, _ := f.registerUser(t, "alice")

	createSearchNote(t, f, token, "alpha searchterm", "public")
	createSearchNote(t, f, token, "beta searchterm", "home")
	createSearchNote(t, f, token, "gamma searchterm", "followers")

	if got := len(searchList(t, f, "/api/notes/search", map[string]any{"query": "searchterm"})); got != 2 {
		t.Errorf("anonymous hits = %d, want 2 (public + home)", got)
	}
	if got := len(searchList(t, f, "/api/notes/search", map[string]any{"query": "searchterm", "i": token})); got != 3 {
		t.Errorf("author hits = %d, want 3", got)
	}
	if got := len(searchList(t, f, "/api/notes/search", map[string]any{"query": "SEARCHTERM"})); got != 2 {
		t.Errorf("case-insensitive hits = %d, want 2", got)
	}
	// The query is a substring match, and LIKE wildcards must not leak in.
	if got := len(searchList(t, f, "/api/notes/search", map[string]any{"query": "%"})); got != 0 {
		t.Errorf("wildcard query hits = %d, want 0", got)
	}
}

func TestNotesSearchHidesMutedAuthor(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")
	_ = bobID
	createSearchNote(t, f, aliceToken, "muted searchterm", "public")

	if got := len(searchList(t, f, "/api/notes/search", map[string]any{"query": "muted searchterm", "i": bobToken})); got != 1 {
		t.Fatalf("before mute hits = %d, want 1", got)
	}

	if status := postNoContent(t, f, "/api/mute/create", map[string]any{
		"i": bobToken, "userId": userID(t, f, "alice"),
	}); status != http.StatusNoContent {
		t.Fatalf("mute/create: status %d, want 204", status)
	}
	if got := len(searchList(t, f, "/api/notes/search", map[string]any{"query": "muted searchterm", "i": bobToken})); got != 0 {
		t.Errorf("after mute hits = %d, want 0", got)
	}
}

// postNoContent posts a body and returns the status without decoding, for the
// 204 endpoints that carry no JSON.
func postNoContent(t *testing.T, f *fedInstance, path string, body map[string]any) int {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
	resp, err := f.ts.Client().Post(f.ts.URL+path, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post %s: %v", path, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// userID resolves a username to its id through users/show.
func userID(t *testing.T, f *fedInstance, username string) string {
	t.Helper()
	resp, out := post(t, f.ts, "/api/users/show", map[string]any{"username": username})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show %s: status %d, body %v", username, resp.StatusCode, out)
	}
	id, _ := out["id"].(string)
	if id == "" {
		t.Fatalf("users/show %s returned no id: %v", username, out)
	}
	return id
}

func TestNotesSearchByTag(t *testing.T) {
	f := newFederationServer(t, "public")
	token, _ := f.registerUser(t, "alice")
	createSearchNote(t, f, token, "tagged body #FooBar", "public")
	createSearchNote(t, f, token, "untagged body", "public")

	if got := len(searchList(t, f, "/api/notes/search-by-tag", map[string]any{"tag": "foobar"})); got != 1 {
		t.Errorf("lowercase tag hits = %d, want 1", got)
	}
	if got := len(searchList(t, f, "/api/notes/search-by-tag", map[string]any{"tag": "FooBar"})); got != 1 {
		t.Errorf("mixed-case tag hits = %d, want 1", got)
	}
	if got := len(searchList(t, f, "/api/notes/search-by-tag", map[string]any{"tag": "nope"})); got != 0 {
		t.Errorf("unknown tag hits = %d, want 0", got)
	}
}

func TestUsersSearch(t *testing.T) {
	f := newFederationServer(t, "public")
	f.registerUser(t, "alice")
	f.registerUser(t, "bob")

	hits := searchList(t, f, "/api/users/search", map[string]any{"query": "ali"})
	if len(hits) != 1 {
		t.Fatalf("query ali hits = %d, want 1: %v", len(hits), hits)
	}
	user, _ := hits[0].(map[string]any)
	if user["username"] != "alice" {
		t.Errorf("username = %v, want alice", user["username"])
	}
	if _, leaked := user["token"]; leaked {
		t.Errorf("users/search leaked a token field: %v", user)
	}

	hits = searchList(t, f, "/api/users/search", map[string]any{"query": "bob", "origin": "local"})
	if len(hits) != 1 {
		t.Errorf("origin local hits = %d, want 1", len(hits))
	}
	hits = searchList(t, f, "/api/users/search", map[string]any{"query": "bob", "origin": "remote"})
	if len(hits) != 0 {
		t.Errorf("origin remote hits = %d, want 0", len(hits))
	}

	resp, raw := postRaw(t, f.ts, "/api/users/search", map[string]any{"query": "bob", "origin": "nonsense"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad origin status = %d, want 400 (body %v)", resp.StatusCode, raw)
	}
}

func TestUsersSearchHidesBlocked(t *testing.T) {
	f := newFederationServer(t, "public")
	f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")

	resp, out := postWithToken(t, f.ts, "/api/blocking/create", bobToken, map[string]any{
		"userId": userID(t, f, "alice"),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("blocking/create: status %d, body %v", resp.StatusCode, out)
	}

	if got := len(searchList(t, f, "/api/users/search", map[string]any{"query": "ali", "i": bobToken})); got != 0 {
		t.Errorf("blocked user hits = %d, want 0", got)
	}
	if got := len(searchList(t, f, "/api/users/search", map[string]any{"query": "ali"})); got != 1 {
		t.Errorf("anonymous hits = %d, want 1", got)
	}
}
