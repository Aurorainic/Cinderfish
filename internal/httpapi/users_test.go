package httpapi_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUsersNotesAndCounts(t *testing.T) {
	ts := newFederationServer(t, "public").ts
	authorID, authorToken := createUser(t, ts, "alice")
	_, viewerToken := createUser(t, ts, "bob")

	publicID := createNote(t, ts, authorToken, map[string]any{"text": "public note"})
	createNote(t, ts, authorToken, map[string]any{"text": "followers only", "visibility": "followers"})

	resp, body := post(t, ts, "/api/users/show", map[string]any{"username": "alice"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show: status %d, body %v", resp.StatusCode, body)
	}
	if body["followersCount"].(float64) != 0 || body["notesCount"].(float64) != 1 {
		t.Errorf("counts before follow = followers %v notes %v, want 0/1", body["followersCount"], body["notesCount"])
	}

	if resp, body := post(t, ts, "/api/following/create", map[string]any{"userId": authorID, "i": viewerToken}); resp.StatusCode != http.StatusOK {
		t.Fatalf("follow: status %d, body %v", resp.StatusCode, body)
	}

	_, body = post(t, ts, "/api/users/show", map[string]any{"username": "alice"})
	if body["followersCount"].(float64) != 1 {
		t.Errorf("followersCount = %v, want 1", body["followersCount"])
	}

	resp, raw := postRaw(t, ts, "/api/users/notes", map[string]any{"username": "alice"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/notes: status %d, body %v", resp.StatusCode, raw)
	}
	notes, ok := raw.([]any)
	if !ok {
		t.Fatalf("users/notes returned %T, want array", raw)
	}
	if len(notes) != 1 {
		t.Fatalf("users/notes returned %d notes, want 1 (public only)", len(notes))
	}
	first, _ := notes[0].(map[string]any)
	if first["id"] != publicID {
		t.Errorf("first note = %v, want %s", first["id"], publicID)
	}
	if first["user"].(map[string]any)["username"] != "alice" {
		t.Errorf("note author = %v", first["user"])
	}

	// A viewer's own reaction is included in the detail packing.
	resp, body = post(t, ts, "/api/notes/reactions/create", map[string]any{
		"noteId": publicID, "reaction": "👍", "i": viewerToken,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("react: status %d, body %v", resp.StatusCode, body)
	}
	_, raw = postRaw(t, ts, "/api/users/notes", map[string]any{"username": "alice", "i": viewerToken})
	first = raw.([]any)[0].(map[string]any)
	if first["myReaction"] != "👍" {
		t.Errorf("myReaction = %v, want thumbs up", first["myReaction"])
	}
}

func TestUsersShowResolvesRemoteAcct(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	_, bobID := b.signup(t, "betabob")
	routeTo(t, b, b)

	resp, out := post(t, a.ts, "/api/users/show", map[string]any{"username": "betabob", "host": "beta.example"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("remote users/show = %d %v", resp.StatusCode, out)
	}
	if out["username"] != "betabob" {
		t.Errorf("resolved user = %v", out)
	}
	if out["host"] != "beta.example" || out["id"] == bobID || out["uri"] != "https://beta.example/users/"+bobID {
		t.Errorf("resolved remote identity = %v", out)
	}
}

func TestUsersNotesRequiresIdentity(t *testing.T) {
	ts := newFederationServer(t, "public").ts
	resp, body := post(t, ts, "/api/users/notes", map[string]any{})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body %v", resp.StatusCode, body)
	}
	resp, body = post(t, ts, "/api/users/notes", map[string]any{"username": "nobody"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body %v", resp.StatusCode, body)
	}
}

func TestUsersNotesFilters(t *testing.T) {
	ts := newTestServer(t, "open")
	_, authorToken := createUser(t, ts, "alice")
	_, bobToken := createUser(t, ts, "bob")

	resp, file := uploadFile(t, ts.URL, authorToken, "pic.png", bytes.NewBufferString("png bytes"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload: status %d, body %v", resp.StatusCode, file)
	}
	fileID, _ := file["id"].(string)

	photoID := createNote(t, ts, authorToken, map[string]any{"text": "with file", "fileIds": []string{fileID}})
	bobNote := createNote(t, ts, bobToken, map[string]any{"text": "bob's note"})
	replyID := createNote(t, ts, authorToken, map[string]any{"text": "reply", "replyId": bobNote})
	plainID := createNote(t, ts, authorToken, map[string]any{"text": "plain"})

	// Default: all three, newest first.
	raw := usersNotesRaw(t, ts, map[string]any{"username": "alice"})
	if got := rawIDs(raw); !equalIDs(got, []string{plainID, replyID, photoID}) {
		t.Errorf("default users/notes = %v, want %v", got, []string{plainID, replyID, photoID})
	}

	// withFiles keeps only the note carrying a file.
	raw = usersNotesRaw(t, ts, map[string]any{"username": "alice", "withFiles": true})
	if got := rawIDs(raw); !equalIDs(got, []string{photoID}) {
		t.Errorf("withFiles users/notes = %v, want [%s]", got, photoID)
	}

	// withReplies=false drops replies to other users.
	raw = usersNotesRaw(t, ts, map[string]any{"username": "alice", "withReplies": false})
	if got := rawIDs(raw); !equalIDs(got, []string{plainID, photoID}) {
		t.Errorf("withReplies=false users/notes = %v, want %v", got, []string{plainID, photoID})
	}
}

// usersNotesRaw posts /api/users/notes and returns the raw response.
func usersNotesRaw(t *testing.T, ts *httptest.Server, body map[string]any) any {
	t.Helper()
	resp, raw := postRaw(t, ts, "/api/users/notes", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/notes: status %d, body %v", resp.StatusCode, raw)
	}
	return raw
}

// rawIDs extracts the note ids from a users/notes response.
func rawIDs(raw any) []string {
	list, _ := raw.([]any)
	ids := make([]string, 0, len(list))
	for _, item := range list {
		m, _ := item.(map[string]any)
		if id, ok := m["id"].(string); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func equalIDs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
