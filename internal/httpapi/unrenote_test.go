package httpapi_test

import (
	"net/http"
	"testing"
)

// TestNotesUnrenote covers notes/unrenote: it removes the caller's pure renote
// of the original note, is idempotent, and leaves a quoted renote alone.
func TestNotesUnrenote(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t)+"a")
	_, bobToken := createUser(t, ts, uniqueName(t)+"b")

	bobNote := createNote(t, ts, bobToken, map[string]any{"text": "renote target"})

	// A pure renote.
	resp, renoteBody := post(t, ts, "/api/notes/create", map[string]any{
		"i": aliceToken, "renoteId": bobNote, "visibility": "public",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("renote = %d %v", resp.StatusCode, renoteBody)
	}
	created, _ := renoteBody["createdNote"].(map[string]any)
	renoteID, _ := created["id"].(string)

	list := postListWithToken(t, ts, "/api/notes/renotes", aliceToken, map[string]any{"noteId": bobNote})
	if len(list) == 0 {
		t.Fatalf("renotes = %v, want the pure renote", list)
	}

	if status := postNoContentWithToken(t, ts, "/api/notes/unrenote", aliceToken, map[string]any{"noteId": bobNote}); status != http.StatusNoContent {
		t.Fatalf("unrenote = %d, want 204", status)
	}
	if list := postListWithToken(t, ts, "/api/notes/renotes", aliceToken, map[string]any{"noteId": bobNote}); len(list) != 0 {
		t.Errorf("renotes after unrenote = %v, want none", list)
	}
	// The renote row is gone from the author's view as well.
	if resp, _ := postWithToken(t, ts, "/api/notes/show", aliceToken, map[string]any{"noteId": renoteID}); resp.StatusCode == http.StatusOK {
		t.Error("the renote still exists after unrenote")
	}

	// A second call is a no-op rather than an error.
	if status := postNoContentWithToken(t, ts, "/api/notes/unrenote", aliceToken, map[string]any{"noteId": bobNote}); status != http.StatusNoContent {
		t.Errorf("second unrenote = %d, want 204", status)
	}

	// A quoted renote is not a "pure" renote and must survive.
	resp, quoteBody := post(t, ts, "/api/notes/create", map[string]any{
		"i": aliceToken, "renoteId": bobNote, "text": "quoting it", "visibility": "public",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("quote = %d %v", resp.StatusCode, quoteBody)
	}
	if status := postNoContentWithToken(t, ts, "/api/notes/unrenote", aliceToken, map[string]any{"noteId": bobNote}); status != http.StatusNoContent {
		t.Errorf("unrenote with a quote present = %d, want 204", status)
	}
	if list := postListWithToken(t, ts, "/api/notes/renotes", aliceToken, map[string]any{"noteId": bobNote}); len(list) != 1 {
		t.Errorf("quoted renote was removed: %v", list)
	}

	// Missing parameter and anonymous callers.
	if status := postNoContentWithToken(t, ts, "/api/notes/unrenote", aliceToken, map[string]any{}); status != http.StatusBadRequest {
		t.Errorf("missing noteId = %d, want 400", status)
	}
	if resp, _ := post(t, ts, "/api/notes/unrenote", map[string]any{"noteId": bobNote}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", resp.StatusCode)
	}
}
