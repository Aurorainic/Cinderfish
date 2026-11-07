package httpapi_test

import (
	"net/http"
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
