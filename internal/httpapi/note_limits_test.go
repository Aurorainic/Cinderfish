package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestNotesCreateParamLimits pins notes/create's schema limits: text counts
// Unicode code points (not bytes), CW is 1–100, at most 16 files, and polls
// take 2–10 distinct choices of 1–50 code points.
func TestNotesCreateParamLimits(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, "alice")

	// 3000 CJK code points are 9000 bytes; they must still be accepted.
	text := strings.Repeat("漢", 3000)
	if resp, body := postWithToken(t, ts, "/api/notes/create", token, map[string]any{"text": text}); resp.StatusCode != http.StatusOK {
		t.Fatalf("3000 CJK chars: status %d body %v", resp.StatusCode, body)
	}
	if resp, _ := postWithToken(t, ts, "/api/notes/create", token, map[string]any{"text": text + "字"}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("3001 chars: status = %d, want 400", resp.StatusCode)
	}

	// Content warning: 100 passes, 101 fails, empty fails.
	if resp, body := postWithToken(t, ts, "/api/notes/create", token, map[string]any{
		"text": "cw", "cw": strings.Repeat("x", 100),
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("100-char cw: status %d body %v", resp.StatusCode, body)
	}
	for _, cw := range []string{strings.Repeat("x", 101), ""} {
		resp, body := postWithToken(t, ts, "/api/notes/create", token, map[string]any{"text": "cw", "cw": cw})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("cw %d chars: status = %d, want 400 (%v)", len(cw), resp.StatusCode, body)
		}
		if code, _ := body["error"].(map[string]any)["code"].(string); code != "INVALID_PARAM" {
			t.Errorf("cw %d chars: code = %v, want INVALID_PARAM", len(cw), code)
		}
	}

	// More than 16 file ids is a client error even if the ids do not exist.
	ids := make([]string, 17)
	for i := range ids {
		ids[i] = "0000000000000000000" + string(rune('0'+i%10))
	}
	if resp, _ := postWithToken(t, ts, "/api/notes/create", token, map[string]any{"text": "files", "fileIds": ids}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("17 files: status = %d, want 400", resp.StatusCode)
	}

	// Text-only notes may not be whitespace.
	if resp, _ := postWithToken(t, ts, "/api/notes/create", token, map[string]any{"text": "   "}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("whitespace-only text: status = %d, want 400", resp.StatusCode)
	}

	// The edit path counts code points too.
	noteID := createNote(t, ts, token, map[string]any{"text": "edit me"})
	if resp, body := post(t, ts, "/api/notes/edit", map[string]any{
		"i": token, "editId": noteID, "text": strings.Repeat("漢", 3000),
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("edit 3000 CJK: status %d body %v", resp.StatusCode, body)
	}
	if resp, _ := post(t, ts, "/api/notes/edit", map[string]any{
		"i": token, "editId": noteID, "text": strings.Repeat("漢", 3001),
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("edit 3001 CJK: status = %d, want 400", resp.StatusCode)
	}

	polls := []struct {
		name    string
		choices []string
		ok      bool
	}{
		{"valid", []string{"a", "b"}, true},
		{"one choice", []string{"a"}, false},
		{"eleven choices", []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}, false},
		{"duplicate", []string{"a", "a"}, false},
		{"empty choice", []string{"a", ""}, false},
		{"51-code-point choice", []string{"a", strings.Repeat("漢", 51)}, false},
		{"50-code-point choice", []string{"a", strings.Repeat("漢", 50)}, true},
	}
	for _, tc := range polls {
		resp, body := postWithToken(t, ts, "/api/notes/create", token, map[string]any{
			"poll": map[string]any{"choices": tc.choices},
		})
		if tc.ok && resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status %d, want 200 (%v)", tc.name, resp.StatusCode, body)
		}
		if !tc.ok && resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400 (%v)", tc.name, resp.StatusCode, body)
		}
	}
}
