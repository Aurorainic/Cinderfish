package httpapi_test

import (
	"bytes"
	"net/http"
	"testing"
)

func TestPackNoteFiles(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	respA, fileA := uploadFile(t, ts.URL, token, "a.png", bytes.NewBufferString("png bytes"))
	if respA.StatusCode != http.StatusOK {
		t.Fatalf("upload a.png: status %d, body %v", respA.StatusCode, fileA)
	}
	respB, fileB := uploadFile(t, ts.URL, token, "b.txt", bytes.NewBufferString("text bytes"))
	if respB.StatusCode != http.StatusOK {
		t.Fatalf("upload b.txt: status %d, body %v", respB.StatusCode, fileB)
	}
	idA, _ := fileA["id"].(string)
	idB, _ := fileB["id"].(string)
	if idA == "" || idB == "" {
		t.Fatalf("uploads returned no ids: %v %v", fileA, fileB)
	}

	id := createNote(t, ts, token, map[string]any{
		"text":    "with files",
		"fileIds": []string{idA, "missing-file-id", idB},
	})

	resp, body := post(t, ts, "/api/notes/show", map[string]any{"noteId": id})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("notes/show: status %d, body %v", resp.StatusCode, body)
	}

	// fileIds keeps only the ids that resolve to the author's drive files:
	// Misskey drops what does not belong to the author at create time
	// (NoteCreateService), so the unknown id never reaches the note.
	fileIDs, ok := body["fileIds"].([]any)
	if !ok || len(fileIDs) != 2 {
		t.Fatalf("fileIds = %v, want the 2 owned ids", body["fileIds"])
	}

	// Missing drive files are skipped; existing ones keep their order.
	files, ok := body["files"].([]any)
	if !ok {
		t.Fatalf("files = %v, want an array", body["files"])
	}
	if len(files) != 2 {
		t.Fatalf("files has %d entries, want 2 (missing file skipped): %v", len(files), files)
	}
	first, _ := files[0].(map[string]any)
	second, _ := files[1].(map[string]any)
	if first["id"] != idA {
		t.Errorf("files[0][id] = %v, want %q", first["id"], idA)
	}
	if second["id"] != idB {
		t.Errorf("files[1][id] = %v, want %q", second["id"], idB)
	}
	for i, entry := range []map[string]any{first, second} {
		if url, _ := entry["url"].(string); url == "" {
			t.Errorf("files[%d] has no url: %v", i, entry)
		}
		if typ, _ := entry["type"].(string); typ == "" {
			t.Errorf("files[%d] has no type: %v", i, entry)
		}
		if _, present := entry["isSensitive"]; !present {
			t.Errorf("files[%d] has no isSensitive: %v", i, entry)
		}
		if name, _ := entry["name"].(string); name == "" {
			t.Errorf("files[%d] has no name: %v", i, entry)
		}
	}
}

func TestPackNoteWithoutFiles(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	id := createNote(t, ts, token, map[string]any{"text": "no files"})

	resp, body := post(t, ts, "/api/notes/show", map[string]any{"noteId": id})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("notes/show: status %d, body %v", resp.StatusCode, body)
	}
	files, ok := body["files"].([]any)
	if !ok || len(files) != 0 {
		t.Errorf("files = %v, want an empty array", body["files"])
	}
}

// TestCreateNoteRejectsForeignFile proves a fileIds entry from another
// account's drive is dropped, not mounted: the note carries no file and the
// file stays with its owner.
func TestCreateNoteRejectsForeignFile(t *testing.T) {
	ts := newTestServer(t, "open")
	base := uniqueName(t)
	if len(base) > 15 {
		base = base[:15]
	}
	_, aliceToken := createUser(t, ts, base+"a")
	_, bobToken := createUser(t, ts, base+"b")

	respA, fileA := uploadFile(t, ts.URL, aliceToken, "secret.png", bytes.NewBufferString("private bytes"))
	if respA.StatusCode != http.StatusOK {
		t.Fatalf("upload: status %d, body %v", respA.StatusCode, fileA)
	}
	fileID, _ := fileA["id"].(string)
	if fileID == "" {
		t.Fatalf("upload returned no id: %v", fileA)
	}

	noteID := createNote(t, ts, bobToken, map[string]any{
		"text":    "stealing the file",
		"fileIds": []string{fileID},
	})

	resp, body := post(t, ts, "/api/notes/show", map[string]any{"noteId": noteID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("notes/show: status %d, body %v", resp.StatusCode, body)
	}
	if files, ok := body["files"].([]any); !ok || len(files) != 0 {
		t.Errorf("foreign file attached to the note: %v", body["files"])
	}
	if ids, ok := body["fileIds"].([]any); !ok || len(ids) != 0 {
		t.Errorf("foreign fileId stored on the note: %v", body["fileIds"])
	}
}
