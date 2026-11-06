package httpapi_test

import (
	"net/http"
	"testing"
)

// TestNotesState covers notes/state: the favourite and pin flags follow the
// real rows, unfavouriting and unpinning clear them, and the error branches.
func TestNotesState(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t)+"a")
	noteID := createNote(t, ts, token, map[string]any{"text": "state target"})

	read := func() map[string]any {
		resp, body := postWithToken(t, ts, "/api/notes/state", token, map[string]any{"noteId": noteID})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("notes/state = %d %v", resp.StatusCode, body)
		}
		return body
	}

	if state := read(); state["isFavorited"] != false || state["isPinned"] != false {
		t.Fatalf("fresh state = %v", state)
	}

	if resp, body := postWithToken(t, ts, "/api/notes/favorites/create", token, map[string]any{"noteId": noteID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("favorite: %d %v", resp.StatusCode, body)
	}
	if resp, body := postWithToken(t, ts, "/api/i/pin", token, map[string]any{"noteId": noteID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("pin: %d %v", resp.StatusCode, body)
	}
	state := read()
	if state["isFavorited"] != true || state["isPinned"] != true {
		t.Errorf("state after favorite+pin = %v", state)
	}

	if resp, body := postWithToken(t, ts, "/api/notes/favorites/delete", token, map[string]any{"noteId": noteID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("unfavorite: %d %v", resp.StatusCode, body)
	}
	if resp, body := postWithToken(t, ts, "/api/i/unpin", token, map[string]any{"noteId": noteID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("unpin: %d %v", resp.StatusCode, body)
	}
	state = read()
	if state["isFavorited"] != false || state["isPinned"] != false {
		t.Errorf("state after clearing = %v", state)
	}
	// Thread muting is not implemented; the flag stays false rather than absent.
	if state["isMutedThread"] != false {
		t.Errorf("isMutedThread = %v, want false", state["isMutedThread"])
	}

	// Unknown note, missing parameter and anonymous callers.
	if resp, _ := postWithToken(t, ts, "/api/notes/state", token, map[string]any{"noteId": "00mw000000000000000"}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown note = %d, want 404", resp.StatusCode)
	}
	if resp, _ := postWithToken(t, ts, "/api/notes/state", token, map[string]any{}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing noteId = %d, want 400", resp.StatusCode)
	}
	if resp, _ := post(t, ts, "/api/notes/state", map[string]any{"noteId": noteID}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", resp.StatusCode)
	}
}
