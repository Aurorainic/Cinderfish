package httpapi_test

import (
	"bytes"
	"net/http"
	"testing"
)

// TestDriveUsage covers /api/drive: capacity is 0 (unlimited), the byte total
// matches the uploads and drops again after a delete, and the file count keeps
// step. The two content-sensitivity switches are checked through i/update.
func TestDriveUsage(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t)+"a")

	// One 5-byte and one 7-byte upload.
	resp, first := uploadFile(t, ts.URL, token, "a.bin", bytes.NewBufferString("12345"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first upload: %d %v", resp.StatusCode, first)
	}
	firstID, _ := first["id"].(string)
	if resp, second := uploadFile(t, ts.URL, token, "b.bin", bytes.NewBufferString("1234567")); resp.StatusCode != http.StatusOK {
		t.Fatalf("second upload: %d %v", resp.StatusCode, second)
	}

	usage := func() (capacity, used, count float64) {
		_, body := postWithToken(t, ts, "/api/drive", token, map[string]any{})
		c, _ := body["capacity"].(float64)
		u, _ := body["usage"].(float64)
		n, _ := body["filesCount"].(float64)
		return c, u, n
	}
	if capacity, used, count := usage(); capacity != 0 || used != 12 || count != 2 {
		t.Fatalf("usage = capacity %v, used %v, count %v; want 0/12/2", capacity, used, count)
	}

	// Deleting one file lowers both numbers. The endpoint answers 200 with the
	// deleted file, as Misskey does.
	if resp, body := postWithToken(t, ts, "/api/drive/files/delete", token, map[string]any{"fileId": firstID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d %v", resp.StatusCode, body)
	}
	if _, used, count := usage(); used != 7 || count != 1 {
		t.Errorf("usage after delete = %v bytes / %v files, want 7/1", used, count)
	}

	// The two sensitivity switches write and read back.
	resp, updated := post(t, ts, "/api/i/update", map[string]any{
		"i": token, "alwaysMarkNsfw": true, "autoSensitive": true,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sensitivity update = %d %v", resp.StatusCode, updated)
	}
	if updated["alwaysMarkNsfw"] != true || updated["autoSensitive"] != true {
		t.Errorf("switches = %v/%v", updated["alwaysMarkNsfw"], updated["autoSensitive"])
	}
	_, me := post(t, ts, "/api/i", map[string]any{"i": token})
	if me["alwaysMarkNsfw"] != true || me["autoSensitive"] != true {
		t.Errorf("switches after reload = %v/%v", me["alwaysMarkNsfw"], me["autoSensitive"])
	}
}
