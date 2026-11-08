package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestInstanceSettingsUpdateAndMeta(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	resp, raw := postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"name": "测试实例", "description": "这是一台测试实例", "maintainerName": "维护者",
		"maintainerEmail": "admin@example.com", "registrationMode": "open",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update-meta: status %d, body %v", resp.StatusCode, raw)
	}
	updated, _ := raw.(map[string]any)
	if updated["name"] != "测试实例" || updated["registrationMode"] != "open" {
		t.Errorf("admin/meta = %v", updated)
	}

	// The public meta reflects the runtime values.
	resp, meta := post(t, ts, "/api/meta", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("meta: status %d", resp.StatusCode)
	}
	if meta["name"] != "测试实例" || meta["description"] != "这是一台测试实例" {
		t.Errorf("public meta name/description = %v / %v", meta["name"], meta["description"])
	}
	if meta["maintainerName"] != "维护者" {
		t.Errorf("maintainerName = %v", meta["maintainerName"])
	}

	// admin/meta returns the same subset.
	resp, raw = postRawWithToken(t, ts, "/api/admin/meta", "admintoken000000", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin/meta: status %d, body %v", resp.StatusCode, raw)
	}
	current, _ := raw.(map[string]any)
	if current["name"] != "测试实例" || current["maintainerEmail"] != "admin@example.com" {
		t.Errorf("admin/meta = %v", current)
	}
}

func TestInstanceRegistrationModeIsRuntime(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	// Open: self-service signup works.
	if resp, _ := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"}); resp.StatusCode != http.StatusOK {
		t.Fatalf("signup in open mode = %d", resp.StatusCode)
	}

	// Switching to admin closes signup immediately.
	resp, raw := postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{"registrationMode": "admin"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("switch to admin: status %d, body %v", resp.StatusCode, raw)
	}
	if resp, _ := post(t, ts, "/api/signup", map[string]any{"username": "bob", "password": "password123"}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("signup in admin mode = %d, want 400", resp.StatusCode)
	}

	// meta says the same thing in both of its spellings; features.registration
	// used to read the startup configuration and disagree with
	// disableRegistration after a runtime switch.
	_, meta := post(t, ts, "/api/meta", map[string]any{"detail": true})
	if meta["disableRegistration"] != true {
		t.Errorf("disableRegistration = %v after switching to admin", meta["disableRegistration"])
	}
	features, _ := meta["features"].(map[string]any)
	if features["registration"] != false {
		t.Errorf("features.registration = %v after switching to admin", features["registration"])
	}

	// Invite mode still refuses a signup without a code.
	postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{"registrationMode": "invite"})
	if resp, _ := post(t, ts, "/api/signup", map[string]any{"username": "carol", "password": "password123"}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("signup in invite mode without code = %d, want 400", resp.StatusCode)
	}

	// Back to open.
	postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{"registrationMode": "open"})
	if resp, _ := post(t, ts, "/api/signup", map[string]any{"username": "dave", "password": "password123"}); resp.StatusCode != http.StatusOK {
		t.Errorf("signup after reopening = %d, want 200", resp.StatusCode)
	}
}

func TestInstanceSettingsValidationAndPermissions(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	cases := []map[string]any{
		{"name": "   "},
		{"description": string(make([]byte, 501))},
		{"maintainerEmail": "not-an-address"},
		{"registrationMode": "bogus"},
	}
	for i, body := range cases {
		resp, _ := postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", body)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("case %d status = %d, want 400", i, resp.StatusCode)
		}
	}

	// Non-admin cannot read or write.
	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	aliceToken, _ := alice["token"].(string)
	resp, _ := postRawWithToken(t, ts, "/api/admin/update-meta", aliceToken, map[string]any{"name": "hijack"})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin update status = %d, want 403", resp.StatusCode)
	}
	resp, _ = postRawWithToken(t, ts, "/api/admin/meta", aliceToken, map[string]any{})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin read status = %d, want 403", resp.StatusCode)
	}
}

func TestInstanceBlockedHosts(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	resp, raw := postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"blockedHosts": []string{"B.Example.com", " a.example.com ", "", "a.example.com"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update-meta blockedHosts: status %d, body %v", resp.StatusCode, raw)
	}
	updated, _ := raw.(map[string]any)
	hosts, _ := updated["blockedHosts"].([]any)
	if len(hosts) != 2 || hosts[0] != "a.example.com" || hosts[1] != "b.example.com" {
		t.Errorf("blockedHosts = %v, want [a.example.com b.example.com]", hosts)
	}

	// The list survives a re-read.
	resp, raw = postRawWithToken(t, ts, "/api/admin/meta", "admintoken000000", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin/meta: status %d", resp.StatusCode)
	}
	current, _ := raw.(map[string]any)
	if list, _ := current["blockedHosts"].([]any); len(list) != 2 {
		t.Errorf("admin/meta blockedHosts = %v", current["blockedHosts"])
	}

	// A hostname with a scheme is rejected.
	resp, _ = postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"blockedHosts": []string{"https://evil.example"},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("scheme in blockedHosts status = %d, want 400", resp.StatusCode)
	}
}

// TestInstanceRuntimeLimits covers the two runtime limits added with the
// "其它设置" batch: preserved usernames (Misskey meta.preservedUsernames) and
// the note-length cap (maxNoteTextLength). Both are stored in instance_meta
// and take effect without a restart.
func TestInstanceRuntimeLimits(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	// Normalisation: "@" prefixes and stray whitespace are stripped, entries
	// are deduplicated case-insensitively, and the first spelling is kept.
	resp, raw := postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"preservedUsernames": []string{" Admin ", "admin", "@root", ""},
		"maxNoteTextLength":  100,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update-meta limits: status %d, body %v", resp.StatusCode, raw)
	}
	updated, _ := raw.(map[string]any)
	names, _ := updated["preservedUsernames"].([]any)
	if len(names) != 2 || names[0] != "Admin" || names[1] != "root" {
		t.Errorf("preservedUsernames = %v, want [Admin root]", updated["preservedUsernames"])
	}
	if updated["maxNoteTextLength"] != float64(100) {
		t.Errorf("maxNoteTextLength = %v, want 100", updated["maxNoteTextLength"])
	}

	// The public meta carries both, so clients validate against the same list.
	_, meta := post(t, ts, "/api/meta", map[string]any{})
	if meta["maxNoteTextLength"] != float64(100) {
		t.Errorf("public maxNoteTextLength = %v, want 100", meta["maxNoteTextLength"])
	}
	if publicNames, _ := meta["preservedUsernames"].([]any); len(publicNames) != 2 {
		t.Errorf("public preservedUsernames = %v", meta["preservedUsernames"])
	}

	// Signup refuses a preserved name (case-insensitive), with Misskey's code.
	signupResp, signupBody := post(t, ts, "/api/signup", map[string]any{
		"username": "ROOT", "password": "password123",
	})
	if signupResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("signup with a preserved name = %d, want 400", signupResp.StatusCode)
	}
	errBody, _ := signupBody["error"].(map[string]any)
	if errBody["code"] != "USED_USERNAME" {
		t.Errorf("signup error code = %v, want USED_USERNAME", errBody["code"])
	}

	// username/available agrees with the signup answer.
	_, available := post(t, ts, "/api/username/available", map[string]any{"username": "Root"})
	if available["available"] != false {
		t.Errorf("username/available for a preserved name = %v, want false", available["available"])
	}

	// The note-length cap is live: 101 code points are refused, 100 pass.
	_, signup := post(t, ts, "/api/signup", map[string]any{"username": "writer", "password": "password123"})
	writerToken, _ := signup["token"].(string)
	if writerToken == "" {
		t.Fatalf("writer signup failed: %v", signup)
	}
	long := strings.Repeat("a", 101)
	resp, raw = postRawWithToken(t, ts, "/api/notes/create", writerToken, map[string]any{"text": long})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("101-char note = %d, want 400", resp.StatusCode)
	}
	if errBody, _ := raw.(map[string]any); errBody["error"] != nil {
		errObj, _ := errBody["error"].(map[string]any)
		if errObj["code"] != "NOTE_TOO_LONG" {
			t.Errorf("note error code = %v, want NOTE_TOO_LONG", errObj["code"])
		}
	}
	if resp, raw := postRawWithToken(t, ts, "/api/notes/create", writerToken, map[string]any{
		"text": strings.Repeat("a", 100),
	}); resp.StatusCode != http.StatusOK {
		t.Errorf("100-char note = %d, want 200 (body %v)", resp.StatusCode, raw)
	}

	// Out-of-range values are refused.
	if resp, _ := postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"maxNoteTextLength": 99,
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("maxNoteTextLength 99 = %d, want 400", resp.StatusCode)
	}
	if resp, _ := postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"preservedUsernames": []string{"bad name"},
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("preserved name with a space = %d, want 400", resp.StatusCode)
	}
}
