package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// postNoDecode posts a JSON body and returns the raw response, for
// endpoints that answer 204 with an empty body.
func postNoDecode(t *testing.T, ts *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestIPinRequiresCredential(t *testing.T) {
	ts := newTestServer(t, "open")
	if resp := postNoDecode(t, ts, "/api/i/pin", map[string]any{"noteId": "x"}); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("pin status = %d, want 401", resp.StatusCode)
	}
	if resp := postNoDecode(t, ts, "/api/i/unpin", map[string]any{"noteId": "x"}); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unpin status = %d, want 401", resp.StatusCode)
	}
}

func TestIPinUnpinAndUsersShow(t *testing.T) {
	ts := newTestServer(t, "open")
	authorName := uniqueName(t)
	authorID, authorToken := createUser(t, ts, authorName)
	_, otherToken := createUser(t, ts, authorName+"b")

	n1 := createNote(t, ts, authorToken, map[string]any{"text": "first"})
	n2 := createNote(t, ts, authorToken, map[string]any{"text": "second"})
	foreign := createNote(t, ts, otherToken, map[string]any{"text": "not mine"})

	// Pin n1 then n2: the list is newest pin first.
	if resp := postNoDecode(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": n1}); resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Fatalf("pin n1: status %d", resp.StatusCode)
	}
	if resp := postNoDecode(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": n2}); resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Fatalf("pin n2: status %d", resp.StatusCode)
	}

	resp, show := post(t, ts, "/api/users/show", map[string]any{"userId": authorID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show: status %d, body %v", resp.StatusCode, show)
	}
	ids, ok := show["pinnedNoteIds"].([]any)
	if !ok || len(ids) != 2 {
		t.Fatalf("pinnedNoteIds = %v, want two ids", show["pinnedNoteIds"])
	}
	if ids[0] != n2 || ids[1] != n1 {
		t.Errorf("pinnedNoteIds = %v, want [%s %s]", ids, n2, n1)
	}
	pinned, ok := show["pinnedNotes"].([]any)
	if !ok || len(pinned) != 2 {
		t.Fatalf("pinnedNotes = %v, want two notes", show["pinnedNotes"])
	}
	first, _ := pinned[0].(map[string]any)
	if first["id"] != n2 {
		t.Errorf("pinnedNotes[0] = %v, want %s", first["id"], n2)
	}
	if packedUser, ok := first["user"].(map[string]any); !ok || packedUser["username"] != authorName {
		t.Errorf("pinned note author packing = %v", first["user"])
	}

	// A foreign note cannot be pinned.
	resp, body := post(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": foreign})
	if resp.StatusCode == 0 || resp.StatusCode < http.StatusBadRequest {
		t.Fatalf("pin foreign note: status %d, body %v", resp.StatusCode, body)
	}
	if errCode, _ := body["error"].(map[string]any); errCode != nil {
		t.Logf("foreign pin error code: %v", errCode["code"])
	}

	// An unknown note cannot be pinned either.
	resp, body = post(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": "nosuchnote"})
	if resp.StatusCode == 0 || resp.StatusCode < http.StatusBadRequest {
		t.Fatalf("pin unknown note: status %d, body %v", resp.StatusCode, body)
	}

	// Unpinning removes the note from the list.
	if resp := postNoDecode(t, ts, "/api/i/unpin", map[string]any{"i": authorToken, "noteId": n2}); resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Fatalf("unpin n2: status %d", resp.StatusCode)
	}
	_, show = post(t, ts, "/api/users/show", map[string]any{"userId": authorID})
	ids = show["pinnedNoteIds"].([]any)
	if len(ids) != 1 || ids[0] != n1 {
		t.Errorf("pinnedNoteIds after unpin = %v, want [%s]", ids, n1)
	}

	// Unpinning a note that is not pinned is a no-op.
	if resp := postNoDecode(t, ts, "/api/i/unpin", map[string]any{"i": authorToken, "noteId": "nosuchnote"}); resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Fatalf("unpin unknown note: status %d", resp.StatusCode)
	}

	// Re-pinning the same note does not duplicate it.
	postNoDecode(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": n1})
	postNoDecode(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": n1})
	_, show = post(t, ts, "/api/users/show", map[string]any{"userId": authorID})
	ids = show["pinnedNoteIds"].([]any)
	if len(ids) != 1 || ids[0] != n1 {
		t.Errorf("pinnedNoteIds after re-pin = %v, want [%s]", ids, n1)
	}
}
