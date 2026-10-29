package httpapi_test

import (
	"net/http"
	"testing"
)

func TestIPinRequiresCredential(t *testing.T) {
	ts := newTestServer(t, "open")
	resp, body := post(t, ts, "/api/i/pin", map[string]any{"noteId": "x"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("pin status = %d, want 401; body %v", resp.StatusCode, body)
	}
	resp, body = post(t, ts, "/api/i/unpin", map[string]any{"noteId": "x"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unpin status = %d, want 401; body %v", resp.StatusCode, body)
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

	// Pin n1 then n2: the list is newest pin first. Both endpoints
	// answer 200 with the packed MeDetailed user.
	resp, me := post(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": n1})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pin n1: status %d, body %v", resp.StatusCode, me)
	}
	if me["id"] != authorID || me["username"] != authorName {
		t.Errorf("pin response is not the packed user: %v", me)
	}
	resp, me = post(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": n2})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pin n2: status %d, body %v", resp.StatusCode, me)
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

	// A foreign note cannot be pinned; Misskey answers NO_SUCH_NOTE so
	// the note's existence is not leaked.
	resp, body := post(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": foreign})
	errObj, _ := body["error"].(map[string]any)
	if resp.StatusCode != http.StatusNotFound || errObj["code"] != "NO_SUCH_NOTE" {
		t.Fatalf("pin foreign note: status %d, body %v", resp.StatusCode, body)
	}

	// An unknown note cannot be pinned either.
	resp, body = post(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": "nosuchnote"})
	errObj, _ = body["error"].(map[string]any)
	if resp.StatusCode != http.StatusNotFound || errObj["code"] != "NO_SUCH_NOTE" {
		t.Fatalf("pin unknown note: status %d, body %v", resp.StatusCode, body)
	}

	// Unpinning a pinned note answers 200 with the packed user.
	resp, me = post(t, ts, "/api/i/unpin", map[string]any{"i": authorToken, "noteId": n2})
	if resp.StatusCode != http.StatusOK || me["id"] != authorID {
		t.Fatalf("unpin n2: status %d, body %v", resp.StatusCode, me)
	}
	_, show = post(t, ts, "/api/users/show", map[string]any{"userId": authorID})
	ids = show["pinnedNoteIds"].([]any)
	if len(ids) != 1 || ids[0] != n1 {
		t.Errorf("pinnedNoteIds after unpin = %v, want [%s]", ids, n1)
	}

	// api/i carries the caller's own pin list as well.
	resp, me = post(t, ts, "/api/i", map[string]any{"i": authorToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("api/i: status %d, body %v", resp.StatusCode, me)
	}
	myIDs, _ := me["pinnedNoteIds"].([]any)
	if len(myIDs) != 1 || myIDs[0] != n1 {
		t.Errorf("api/i pinnedNoteIds = %v, want [%s]", myIDs, n1)
	}
	myNotes, _ := me["pinnedNotes"].([]any)
	if len(myNotes) != 1 {
		t.Errorf("api/i pinnedNotes = %v, want one packed note", myNotes)
	}

	// Unpinning a note the user does not own is refused like Misskey:
	// NO_SUCH_NOTE, without revealing the note's existence.
	resp, body = post(t, ts, "/api/i/unpin", map[string]any{"i": authorToken, "noteId": foreign})
	errObj, _ = body["error"].(map[string]any)
	if resp.StatusCode != http.StatusNotFound || errObj["code"] != "NO_SUCH_NOTE" {
		t.Fatalf("unpin foreign note: status %d, body %v", resp.StatusCode, body)
	}
	resp, body = post(t, ts, "/api/i/unpin", map[string]any{"i": authorToken, "noteId": "nosuchnote"})
	errObj, _ = body["error"].(map[string]any)
	if resp.StatusCode != http.StatusNotFound || errObj["code"] != "NO_SUCH_NOTE" {
		t.Fatalf("unpin unknown note: status %d, body %v", resp.StatusCode, body)
	}

	// Unpinning a note that is not pinned is a no-op success.
	resp, body = post(t, ts, "/api/i/unpin", map[string]any{"i": authorToken, "noteId": n2})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unpin not-pinned note: status %d, body %v", resp.StatusCode, body)
	}

	// Re-pinning the same note does not duplicate it.
	post(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": n1})
	resp, me = post(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": n1})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("re-pin: status %d, body %v", resp.StatusCode, me)
	}
	_, show = post(t, ts, "/api/users/show", map[string]any{"userId": authorID})
	ids = show["pinnedNoteIds"].([]any)
	if len(ids) != 1 || ids[0] != n1 {
		t.Errorf("pinnedNoteIds after re-pin = %v, want [%s]", ids, n1)
	}
}

func TestIPinLimit(t *testing.T) {
	ts := newTestServer(t, "open")
	authorName := uniqueName(t)
	authorID, authorToken := createUser(t, ts, authorName)

	var ids []string
	for i := 0; i < 6; i++ {
		ids = append(ids, createNote(t, ts, authorToken, map[string]any{"text": "pin target"}))
	}

	// The first five pins succeed; the sixth is refused.
	for _, id := range ids[:5] {
		resp, body := post(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": id})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("pin %s: status %d, body %v", id, resp.StatusCode, body)
		}
	}
	resp, body := post(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": ids[5]})
	errObj, _ := body["error"].(map[string]any)
	if resp.StatusCode != http.StatusBadRequest || errObj["code"] != "PIN_LIMIT_EXCEEDED" {
		t.Fatalf("sixth pin: status %d, body %v", resp.StatusCode, body)
	}

	// users/show agrees: exactly five ids and five packed notes.
	resp, show := post(t, ts, "/api/users/show", map[string]any{"userId": authorID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show: status %d, body %v", resp.StatusCode, show)
	}
	pinnedIDs, _ := show["pinnedNoteIds"].([]any)
	pinnedNotes, _ := show["pinnedNotes"].([]any)
	if len(pinnedIDs) != 5 || len(pinnedNotes) != 5 {
		t.Errorf("pinnedNoteIds len = %d, pinnedNotes len = %d, want 5/5",
			len(pinnedIDs), len(pinnedNotes))
	}

	// After unpinning one, pinning a new note works again.
	if resp, body = post(t, ts, "/api/i/unpin", map[string]any{"i": authorToken, "noteId": ids[0]}); resp.StatusCode != http.StatusOK {
		t.Fatalf("unpin: status %d, body %v", resp.StatusCode, body)
	}
	if resp, body = post(t, ts, "/api/i/pin", map[string]any{"i": authorToken, "noteId": ids[5]}); resp.StatusCode != http.StatusOK {
		t.Fatalf("pin after unpin: status %d, body %v", resp.StatusCode, body)
	}
}
