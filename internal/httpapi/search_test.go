package httpapi_test

import (
	"bytes"
	"context"
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
	bobToken, _ := f.registerUser(t, "bob")
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

// TestNotesSearchFilters covers the parameters the timeline search accepts
// beyond the query itself: offset, userId, host (local only) and withFiles.
func TestNotesSearchFilters(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")

	first := createNote(t, f.ts, aliceToken, map[string]any{"text": "w63filter first"})
	second := createNote(t, f.ts, aliceToken, map[string]any{"text": "w63filter second"})
	createNote(t, f.ts, bobToken, map[string]any{"text": "w63filter third"})

	// Attach a drive file to the second note so withFiles has something to
	// match. The file row only needs to exist for the EXISTS subquery.
	if _, err := f.pool.Exec(context.Background(), `
INSERT INTO drive_files (id, user_id, name, content_type, size, md5, storage_key, storage_backend)
VALUES ('w63file', $1, 'w63.txt', 'text/plain', 3, 'w63md5', 'w63/key', 'local')
`, aliceID); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE notes SET file_ids = ARRAY['w63file']::text[] WHERE id = $1`, second); err != nil {
		t.Fatalf("attach file: %v", err)
	}

	search := func(params map[string]any) []any {
		params["query"] = "w63filter"
		return postListWithToken(t, f.ts, "/api/notes/search", "", params)
	}

	// Offset pages through the same ordered set.
	all := search(map[string]any{})
	if len(all) != 3 {
		t.Fatalf("search = %d notes, want 3", len(all))
	}
	page := search(map[string]any{"offset": 2})
	if len(page) != 1 {
		t.Fatalf("offset page = %d notes, want 1", len(page))
	}
	// Newest first: the third note is row 0, so offset 2 lands on the first.
	oldest, _ := page[0].(map[string]any)
	if oldest["id"] != first {
		t.Errorf("offset page note = %v, want %s", oldest["id"], first)
	}

	// userId narrows to one author.
	byAlice := search(map[string]any{"userId": aliceID})
	if len(byAlice) != 2 {
		t.Fatalf("userId filter = %d notes, want 2", len(byAlice))
	}
	for _, item := range byAlice {
		if user, _ := item.(map[string]any)["user"].(map[string]any); user["id"] != aliceID {
			t.Errorf("userId filter returned %v", user["id"])
		}
	}

	// An empty host keeps local notes only.
	local := search(map[string]any{"host": ""})
	if len(local) != 3 {
		t.Fatalf("host air filter = %d notes, want 3", len(local))
	}
	// A non-empty host matches nothing here.
	remote := search(map[string]any{"host": "remote.example"})
	if len(remote) != 0 {
		t.Errorf("host remote.example = %d notes, want 0", len(remote))
	}

	// withFiles keeps the note that carries an attachment.
	withFiles := search(map[string]any{"withFiles": true})
	if len(withFiles) != 1 {
		t.Fatalf("withFiles = %d notes, want 1", len(withFiles))
	}
	if got, _ := withFiles[0].(map[string]any); got["id"] != second {
		t.Errorf("withFiles note = %v, want %s", got["id"], second)
	}

	// search-by-tag has the same withFiles switch.
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE notes SET tags = '["w63filter"]'::jsonb WHERE id = $1`, second); err != nil {
		t.Fatalf("tag: %v", err)
	}
	tagged := postListWithToken(t, f.ts, "/api/notes/search-by-tag", "", map[string]any{"tag": "w63filter", "withFiles": true})
	if len(tagged) != 1 || tagged[0].(map[string]any)["id"] != second {
		t.Errorf("search-by-tag withFiles = %v, want the tagged note", tagged)
	}
}

// TestNotesSearchRemoteAuthor covers the host filter with a real remote row:
// an empty host keeps local notes only, a named host returns the remote note,
// and the tag search finds it too.
func TestNotesSearchRemoteAuthor(t *testing.T) {
	f := newFederationServer(t, "public")

	if _, err := f.pool.Exec(context.Background(), `
INSERT INTO users (id, username, username_lower, password_hash, host)
VALUES ('srchremote', 'ruser', 'ruser', 'x', 'remote.example')`); err != nil {
		t.Fatalf("insert remote user: %v", err)
	}
	if _, err := f.pool.Exec(context.Background(), `
INSERT INTO notes (id, user_id, user_host, visibility, text, tags)
VALUES ('srchnote', 'srchremote', 'remote.example', 'public', 'w63remote searchable', '["remotetag"]'::jsonb)`); err != nil {
		t.Fatalf("insert remote note: %v", err)
	}

	search := func(params map[string]any) []any {
		params["query"] = "w63remote"
		return postListWithToken(t, f.ts, "/api/notes/search", "", params)
	}
	if got := search(map[string]any{"host": "remote.example"}); len(got) != 1 {
		t.Fatalf("remote host search = %v, want the remote note", got)
	} else if n, _ := got[0].(map[string]any); n["id"] != "srchnote" {
		t.Errorf("remote search note = %v", n["id"])
	}
	if got := search(map[string]any{"host": ""}); len(got) != 0 {
		t.Errorf("local host search = %v, want none", got)
	}
	if got := search(map[string]any{}); len(got) != 1 {
		t.Errorf("unfiltered search = %v, want the remote note", got)
	}

	tagged := postListWithToken(t, f.ts, "/api/notes/search-by-tag", "", map[string]any{"tag": "remotetag"})
	if len(tagged) != 1 {
		t.Fatalf("tag search = %v, want the remote note", tagged)
	}
}
