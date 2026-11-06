package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// TestAccountExportImportRoundTrip exports an account, removes some rows and
// imports the file back: notes, the following and the favorite come back, a
// second import is a no-op, and malformed files are refused.
func TestAccountExportImportRoundTrip(t *testing.T) {
	ts, srv := adminTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t)+"a")
	bobID, bobToken := createUser(t, ts, uniqueName(t)+"b")

	x1 := createNote(t, ts, aliceToken, map[string]any{"text": "export one"})
	x2 := createNote(t, ts, aliceToken, map[string]any{"text": "export two"})
	bobNote := createNote(t, ts, bobToken, map[string]any{"text": "bob target"})

	if resp, body := post(t, ts, "/api/notes/favorites/create", map[string]any{"i": aliceToken, "noteId": bobNote}); resp.StatusCode != http.StatusOK {
		t.Fatalf("favorite: %d %v", resp.StatusCode, body)
	}
	if resp, body := postWithToken(t, ts, "/api/following/create", aliceToken, map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("follow: %d %v", resp.StatusCode, body)
	}

	resp, doc := post(t, ts, "/api/i/export", map[string]any{"i": aliceToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i/export = %d", resp.StatusCode)
	}
	if doc["format"] != "cinderfish-account-export@1" {
		t.Fatalf("export format = %v", doc["format"])
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal export: %v", err)
	}

	// Remove the rows the import has to restore. The notes are hard-deleted so
	// the conflict check does not see them.
	pool := srv.TestPool()
	if _, err := pool.Exec(context.Background(), `DELETE FROM notes WHERE id = ANY($1)`, []string{x1, x2}); err != nil {
		t.Fatalf("delete notes: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM note_favorites WHERE note_id = $1`, bobNote); err != nil {
		t.Fatalf("delete favorite: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM followings WHERE followee_id = $1`, bobID); err != nil {
		t.Fatalf("delete follow: %v", err)
	}

	uploadResp, uploaded := uploadFile(t, ts.URL, aliceToken, "export.json", bytes.NewBuffer(raw))
	if uploadResp.StatusCode != http.StatusOK {
		t.Fatalf("upload export: %d %v", uploadResp.StatusCode, uploaded)
	}
	fileID, _ := uploaded["id"].(string)

	resp, summary := post(t, ts, "/api/i/import", map[string]any{"i": aliceToken, "fileId": fileID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i/import = %d %v", resp.StatusCode, summary)
	}
	if summary["notes"] != float64(2) || summary["followings"] != float64(1) || summary["favorites"] != float64(1) {
		t.Errorf("import summary = %v, want two notes, one following, one favorite", summary)
	}

	// The restored note reads back with its original text.
	_, shown := postWithToken(t, ts, "/api/notes/show", aliceToken, map[string]any{"noteId": x1})
	if shown["text"] != "export one" {
		t.Errorf("restored note text = %v", shown["text"])
	}
	// The favorite is back.
	favorites := postListWithToken(t, ts, "/api/i/favorites", aliceToken, nil)
	found := false
	for _, item := range favorites {
		n, _ := item.(map[string]any)
		if n["noteId"] == bobNote {
			found = true
		}
		if note, _ := n["note"].(map[string]any); note["id"] == bobNote {
			found = true
		}
	}
	if !found {
		t.Errorf("favorite was not restored: %v", favorites)
	}

	// A second import changes nothing.
	resp, again := post(t, ts, "/api/i/import", map[string]any{"i": aliceToken, "fileId": fileID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("second import = %d %v", resp.StatusCode, again)
	}
	if again["notes"] != float64(0) || again["followings"] != float64(0) || again["favorites"] != float64(0) {
		t.Errorf("second import summary = %v, want nothing added", again)
	}

	// Malformed files are refused with a 400.
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"not json", []byte("this is not json")},
		{"wrong format", []byte(`{"format":"something-else@1"}`)},
	} {
		resp, up := uploadFile(t, ts.URL, aliceToken, tc.name+".json", bytes.NewBuffer(tc.data))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("upload %s: %d", tc.name, resp.StatusCode)
		}
		id, _ := up["id"].(string)
		if resp, _ := post(t, ts, "/api/i/import", map[string]any{"i": aliceToken, "fileId": id}); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("import %s = %d, want 400", tc.name, resp.StatusCode)
		}
	}
}
