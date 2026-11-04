package httpapi_test

import (
	"net/http"
	"testing"
)

// TestNotesReactionsList pins notes/reactions: reactions come back newest
// first with the NoteReaction shape, the type filter keeps one reaction, and
// the cursor pages.
func TestNotesReactionsList(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t))
	_, bobToken := createUser(t, ts, uniqueName(t)+"b")
	_, carolToken := createUser(t, ts, uniqueName(t)+"c")

	noteID := createNote(t, ts, aliceToken, map[string]any{"text": "react to me"})
	if resp, out := post(t, ts, "/api/notes/reactions/create", map[string]any{"i": bobToken, "noteId": noteID, "reaction": "👍"}); resp.StatusCode != http.StatusOK {
		t.Fatalf("bob reacts: %d %v", resp.StatusCode, out)
	}
	if resp, out := post(t, ts, "/api/notes/reactions/create", map[string]any{"i": carolToken, "noteId": noteID, "reaction": "⭐"}); resp.StatusCode != http.StatusOK {
		t.Fatalf("carol reacts: %d %v", resp.StatusCode, out)
	}

	all := postListWithToken(t, ts, "/api/notes/reactions", "", map[string]any{"noteId": noteID})
	if len(all) != 2 {
		t.Fatalf("reactions = %v, want 2", all)
	}
	newest, _ := all[0].(map[string]any)
	oldest, _ := all[1].(map[string]any)
	if newest["type"] != "⭐" || oldest["type"] != "👍" {
		t.Errorf("reaction order = %v, want newest first", all)
	}
	if user, _ := newest["user"].(map[string]any); user["id"] == nil || user["username"] == nil {
		t.Errorf("reaction user is not a UserLite: %v", newest)
	}
	if id, _ := newest["id"].(string); id == "" {
		t.Errorf("reaction id empty: %v", newest)
	}

	filtered := postListWithToken(t, ts, "/api/notes/reactions", "", map[string]any{"noteId": noteID, "type": "👍"})
	if len(filtered) != 1 {
		t.Fatalf("filtered reactions = %v, want one", filtered)
	}
	only, _ := filtered[0].(map[string]any)
	if only["type"] != "👍" {
		t.Errorf("filtered type = %v", only["type"])
	}

	// The untilId cursor pages back from the newest reaction.
	page := postListWithToken(t, ts, "/api/notes/reactions", "", map[string]any{
		"noteId": noteID, "limit": 1, "untilId": newest["id"],
	})
	if len(page) != 1 {
		t.Fatalf("cursor page = %v, want one", page)
	}
	next, _ := page[0].(map[string]any)
	if next["type"] != "👍" {
		t.Errorf("cursor page = %v, want the older reaction", next)
	}
}

// TestNotesReactionsListVisibility proves a note the viewer may not read does
// not leak its reaction list, and an unknown note answers NO_SUCH_NOTE with
// the endpoint's own error id.
func TestNotesReactionsListVisibility(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t))
	_, bobToken := createUser(t, ts, uniqueName(t)+"b")

	private := createNote(t, ts, aliceToken, map[string]any{
		"text": "followers only", "visibility": "followers",
	})
	if resp, out := post(t, ts, "/api/notes/reactions/create", map[string]any{"i": aliceToken, "noteId": private, "reaction": "⭐"}); resp.StatusCode != http.StatusOK {
		t.Fatalf("react: %d %v", resp.StatusCode, out)
	}

	// An outsider gets the same 404 as a missing note, so the reaction list
	// never confirms the note exists.
	if resp, out := post(t, ts, "/api/notes/reactions", map[string]any{"i": bobToken, "noteId": private}); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("outsider reactions = %d %v, want 404", resp.StatusCode, out)
	}
	// The author still sees their own list.
	if own := postListWithToken(t, ts, "/api/notes/reactions", aliceToken, map[string]any{"noteId": private}); len(own) != 1 {
		t.Errorf("author reactions = %v, want one", own)
	}
	resp, out := post(t, ts, "/api/notes/reactions", map[string]any{"noteId": private})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("anonymous private note = %d %v, want 404", resp.StatusCode, out)
	}
	e, _ := out["error"].(map[string]any)
	if e["code"] != "NO_SUCH_NOTE" || e["id"] != "263fff3d-d0e1-4af4-bea7-8408059b451a" {
		t.Errorf("error = %v, want NO_SUCH_NOTE with the reactions id", e)
	}

	if resp, out := post(t, ts, "/api/notes/reactions", map[string]any{"noteId": "00000000000000000000"}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown note = %d %v, want 404", resp.StatusCode, out)
	}
	if resp, _ := post(t, ts, "/api/notes/reactions", map[string]any{}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing noteId = %d, want 400", resp.StatusCode)
	}
}
