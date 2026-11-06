package httpapi_test

import (
	"net/http"
	"testing"
)

// TestNotesEditAndHistory walks notes/edit and notes/history: the author can
// change text/CW/visibility, users/show reflects it, each edit is recorded, and
// everyone else is refused.
func TestNotesEditAndHistory(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t)+"a")
	_, bobToken := createUser(t, ts, uniqueName(t)+"b")

	noteID := createNote(t, ts, aliceToken, map[string]any{"text": "before edit"})
	if _, before := postWithToken(t, ts, "/api/notes/show", aliceToken, map[string]any{"noteId": noteID}); before["updatedAt"] != nil {
		t.Errorf("a fresh note has updatedAt = %v, want null", before["updatedAt"])
	}

	resp, out := post(t, ts, "/api/notes/edit", map[string]any{
		"i": aliceToken, "editId": noteID,
		"text": "after edit", "cw": "cw text", "visibility": "followers",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("notes/edit = %d %v", resp.StatusCode, out)
	}
	updated, _ := out["createdNote"].(map[string]any)
	if updated["text"] != "after edit" || updated["visibility"] != "followers" || updated["cw"] != "cw text" {
		t.Errorf("edited note = %v/%v/%v", updated["text"], updated["visibility"], updated["cw"])
	}

	// The change survives a reload through notes/show. The note is now
	// followers-only, so it is read as its author.
	_, shown := post(t, ts, "/api/notes/show", map[string]any{"i": aliceToken, "noteId": noteID})
	if shown["text"] != "after edit" {
		t.Errorf("notes/show text = %v", shown["text"])
	}
	// The edited marker the client renders is driven by updatedAt.
	if shown["updatedAt"] == nil {
		t.Error("notes/show updatedAt is null after an edit")
	}

	// The edit is recorded as a revision.
	history := postListWithToken(t, ts, "/api/notes/history", aliceToken, map[string]any{"noteId": noteID})
	if len(history) != 1 {
		t.Fatalf("history = %v, want one revision", history)
	}
	rev, _ := history[0].(map[string]any)
	if rev["text"] != "after edit" || rev["noteId"] != noteID {
		t.Errorf("revision = %v", rev)
	}

	// Someone else cannot edit or read the history.
	if resp, _ := post(t, ts, "/api/notes/edit", map[string]any{
		"i": bobToken, "editId": noteID, "text": "hijack",
	}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-author edit = %d, want 403", resp.StatusCode)
	}
	if resp, _ := post(t, ts, "/api/notes/history", map[string]any{
		"i": bobToken, "noteId": noteID,
	}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-author history = %d, want 403", resp.StatusCode)
	}

	// Missing and unknown ids are distinguished.
	resp, body := post(t, ts, "/api/notes/edit", map[string]any{"i": aliceToken, "text": "x"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing editId = %d %v", resp.StatusCode, body)
	}
	if errObj, _ := body["error"].(map[string]any); errObj["code"] != "NEEDS_EDIT_ID" {
		t.Errorf("missing editId error = %v", body["error"])
	}
	if resp, _ := post(t, ts, "/api/notes/edit", map[string]any{
		"i": aliceToken, "editId": "00mw000000000000000", "text": "x",
	}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown editId = %d, want 404", resp.StatusCode)
	}

	// An over-long text is refused.
	long := make([]rune, 3001)
	for i := range long {
		long[i] = 'あ'
	}
	if resp, _ := post(t, ts, "/api/notes/edit", map[string]any{
		"i": aliceToken, "editId": noteID, "text": string(long),
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("over-long text = %d, want 400", resp.StatusCode)
	}
}
