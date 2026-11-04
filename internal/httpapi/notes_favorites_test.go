package httpapi_test

import (
	"net/http"
	"testing"
)

// TestNoteFavoritesRoundTrip pins notes/favorites/create + i/favorites +
// notes/favorites/delete: the create response is a NoteFavorite carrying the
// packed note, a duplicate create returns the same row, and delete removes it.
func TestNoteFavoritesRoundTrip(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	first := createNote(t, ts, token, map[string]any{"text": "favorite me"})

	resp, fav := post(t, ts, "/api/notes/favorites/create", map[string]any{"i": token, "noteId": first})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("favorites/create: %d %v", resp.StatusCode, fav)
	}
	if fav["noteId"] != first {
		t.Errorf("favorite noteId = %v, want %s", fav["noteId"], first)
	}
	if note, _ := fav["note"].(map[string]any); note["id"] != first {
		t.Errorf("favorite note = %v, want %s", fav, first)
	}
	if id, _ := fav["id"].(string); id == "" {
		t.Errorf("favorite id empty: %v", fav)
	}

	_, again := post(t, ts, "/api/notes/favorites/create", map[string]any{"i": token, "noteId": first})
	if again["id"] != fav["id"] {
		t.Errorf("second create id = %v, want %v", again["id"], fav["id"])
	}

	list := postListWithToken(t, ts, "/api/i/favorites", token, nil)
	if len(list) != 1 {
		t.Fatalf("i/favorites = %v, want one favorite", list)
	}
	item, _ := list[0].(map[string]any)
	if item["noteId"] != first {
		t.Errorf("listed noteId = %v, want %s", item["noteId"], first)
	}
	if note, _ := item["note"].(map[string]any); note["id"] != first {
		t.Errorf("listed note = %v, want %s", item, first)
	}

	if resp, out := post(t, ts, "/api/notes/favorites/delete", map[string]any{"i": token, "noteId": first}); resp.StatusCode != http.StatusOK {
		t.Fatalf("favorites/delete: %d %v", resp.StatusCode, out)
	}
	if rest := postListWithToken(t, ts, "/api/i/favorites", token, nil); len(rest) != 0 {
		t.Errorf("favorites after delete = %v, want none", rest)
	}
	// Deleting twice is fine.
	if resp, out := post(t, ts, "/api/notes/favorites/delete", map[string]any{"i": token, "noteId": first}); resp.StatusCode != http.StatusOK {
		t.Errorf("second delete: %d %v", resp.StatusCode, out)
	}
}

// TestNoteFavoritesListPagination covers the newest-first order and the
// untilId cursor.
func TestNoteFavoritesListPagination(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	ids := make([]string, 0, 3)
	for _, text := range []string{"one", "two", "three"} {
		id := createNote(t, ts, token, map[string]any{"text": text})
		ids = append(ids, id)
		if resp, out := post(t, ts, "/api/notes/favorites/create", map[string]any{"i": token, "noteId": id}); resp.StatusCode != http.StatusOK {
			t.Fatalf("favorite %s: %d %v", id, resp.StatusCode, out)
		}
	}

	page := postListWithToken(t, ts, "/api/i/favorites", token, map[string]any{"limit": 2})
	if len(page) != 2 {
		t.Fatalf("first page = %v, want 2", page)
	}
	first, _ := page[0].(map[string]any)
	second, _ := page[1].(map[string]any)
	if first["noteId"] != ids[2] || second["noteId"] != ids[1] {
		t.Errorf("first page = %v, want newest first (%v, %v)", page, ids[2], ids[1])
	}

	next := postListWithToken(t, ts, "/api/i/favorites", token, map[string]any{
		"limit": 2, "untilId": second["id"],
	})
	if len(next) != 1 {
		t.Fatalf("second page = %v, want one", next)
	}
	oldest, _ := next[0].(map[string]any)
	if oldest["noteId"] != ids[0] {
		t.Errorf("second page = %v, want %s", oldest, ids[0])
	}
}

// TestNoteFavoritesErrors pins the failure modes.
func TestNoteFavoritesErrors(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	if resp, _ := post(t, ts, "/api/notes/favorites/create", map[string]any{"noteId": "x"}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous create = %d, want 401", resp.StatusCode)
	}
	if resp, _ := post(t, ts, "/api/i/favorites", map[string]any{}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous list = %d, want 401", resp.StatusCode)
	}
	if resp, _ := post(t, ts, "/api/notes/favorites/create", map[string]any{"i": token}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing noteId = %d, want 400", resp.StatusCode)
	}
	resp, out := post(t, ts, "/api/notes/favorites/create", map[string]any{"i": token, "noteId": "00000000000000000000"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown note = %d %v, want 404", resp.StatusCode, out)
	}
	if e, _ := out["error"].(map[string]any); e["code"] != "NO_SUCH_NOTE" {
		t.Errorf("unknown note code = %v", e["code"])
	}
}
