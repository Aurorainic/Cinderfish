package httpapi_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"testing"
)

func TestDriveAttachedNotes(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, "alice")
	_, bobToken := createUser(t, ts, "bob")

	img := image.NewRGBA(image.Rect(0, 0, 320, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	file := uploadTyped(t, ts.URL, aliceToken, "a.png", "image/png", buf.Bytes())
	fileID, _ := file["id"].(string)
	bobFile := uploadTyped(t, ts.URL, bobToken, "b.png", "image/png", buf.Bytes())

	first := createNote(t, ts, aliceToken, map[string]any{"text": "first", "fileIds": []string{fileID}})
	second := createNote(t, ts, aliceToken, map[string]any{"text": "second", "fileIds": []string{fileID}})

	resp, raw := postRawWithToken(t, ts, "/api/drive/files/attached-notes", aliceToken, map[string]any{"fileId": fileID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body %v", resp.StatusCode, raw)
	}
	list, _ := raw.([]any)
	if len(list) != 2 {
		t.Fatalf("notes = %v, want 2", raw)
	}
	firstItem, _ := list[0].(map[string]any)
	if firstItem["id"] != second {
		t.Errorf("first item = %v, want the newest note %s", firstItem["id"], second)
	}
	if ids, _ := firstItem["fileIds"].([]any); len(ids) != 1 || ids[0] != fileID {
		t.Errorf("fileIds = %v, want [%s]", firstItem["fileIds"], fileID)
	}

	// Pagination by untilId.
	_, raw = postRawWithToken(t, ts, "/api/drive/files/attached-notes", aliceToken, map[string]any{
		"fileId": fileID, "limit": 1, "untilId": second,
	})
	page, _ := raw.([]any)
	if len(page) != 1 {
		t.Fatalf("page = %v, want one note", raw)
	}
	item, _ := page[0].(map[string]any)
	if item["id"] != first {
		t.Errorf("page item = %v, want %s", item["id"], first)
	}

	// A deleted note disappears from the list.
	if resp, body := postWithToken(t, ts, "/api/notes/delete", aliceToken, map[string]any{"noteId": first}); resp.StatusCode != http.StatusOK {
		t.Fatalf("delete: %d %v", resp.StatusCode, body)
	}
	_, raw = postRawWithToken(t, ts, "/api/drive/files/attached-notes", aliceToken, map[string]any{"fileId": fileID})
	if list, _ := raw.([]any); len(list) != 1 {
		t.Errorf("after delete = %v, want one note", raw)
	}

	// Another account's file answers NO_SUCH_FILE, and anonymous is 401.
	resp, body := postWithToken(t, ts, "/api/drive/files/attached-notes", aliceToken, map[string]any{"fileId": bobFile["id"]})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("foreign file status = %d, want 404 (%v)", resp.StatusCode, body)
	}
	if code, _ := body["error"].(map[string]any)["code"].(string); code != "NO_SUCH_FILE" {
		t.Errorf("foreign file code = %v", code)
	}
	if resp, _ := post(t, ts, "/api/drive/files/attached-notes", map[string]any{"fileId": fileID}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous status = %d, want 401", resp.StatusCode)
	}
}
