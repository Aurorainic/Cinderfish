package httpapi_test

import (
	"net/http"
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
