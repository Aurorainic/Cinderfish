package httpapi_test

import (
	"net/http"
	"testing"
)

// TestAdminMailSettings covers the runtime SMTP configuration: the default
// (config.toml, log mode), saving an override, password retention and the
// validation branches. The admin/meta emailServer field follows the live
// host so the old status view stays accurate.
func TestAdminMailSettings(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	resp, raw := postRawWithToken(t, ts, "/api/admin/email-settings", "admintoken000000", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("email-settings = %d, body %v", resp.StatusCode, raw)
	}
	initial, _ := raw.(map[string]any)
	if initial["configured"] != false || initial["stored"] != false || initial["passwordSet"] != false {
		t.Errorf("initial settings = %v, want unconfigured and not stored", initial)
	}
	if initial["port"] != float64(587) || initial["tls"] != "starttls" {
		t.Errorf("initial port/tls = %v / %v", initial["port"], initial["tls"])
	}

	// Save an override; the password is write-only.
	resp, raw = postRawWithToken(t, ts, "/api/admin/email-settings/update", "admintoken000000", map[string]any{
		"host": "smtp.example.com", "port": 465, "username": "noreply@example.com",
		"password": "s3cret", "from": "noreply@example.com", "tls": "tls",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update = %d, body %v", resp.StatusCode, raw)
	}
	updated, _ := raw.(map[string]any)
	if updated["configured"] != true || updated["stored"] != true || updated["passwordSet"] != true {
		t.Errorf("updated settings = %v", updated)
	}
	if updated["host"] != "smtp.example.com" || updated["port"] != float64(465) || updated["tls"] != "tls" {
		t.Errorf("updated fields = %v", updated)
	}
	if _, leaked := updated["password"]; leaked {
		t.Error("email-settings leaked the password")
	}

	// Re-read: the override wins and the password stays set.
	_, raw = postRawWithToken(t, ts, "/api/admin/email-settings", "admintoken000000", map[string]any{})
	reread, _ := raw.(map[string]any)
	if reread["host"] != "smtp.example.com" || reread["passwordSet"] != true || reread["stored"] != true {
		t.Errorf("re-read = %v", reread)
	}

	// Saving without a password keeps the stored one.
	resp, raw = postRawWithToken(t, ts, "/api/admin/email-settings/update", "admintoken000000", map[string]any{
		"host": "smtp.example.com", "port": 587, "tls": "starttls",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("re-update = %d, body %v", resp.StatusCode, raw)
	}
	again, _ := raw.(map[string]any)
	if again["passwordSet"] != true {
		t.Errorf("re-update lost the stored password: %v", again)
	}

	// admin/meta's emailServer follows the live host.
	_, raw = postRawWithToken(t, ts, "/api/admin/meta", "admintoken000000", map[string]any{})
	meta, _ := raw.(map[string]any)
	if meta["emailServer"] != "smtp.example.com" {
		t.Errorf("admin/meta emailServer = %v, want the runtime host", meta["emailServer"])
	}

	// Validation branches.
	for _, body := range []map[string]any{
		{"port": 0, "tls": "bogus"},
		{"port": 70000},
	} {
		if resp, _ := postRawWithToken(t, ts, "/api/admin/email-settings/update", "admintoken000000", body); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("invalid body %v = %d, want 400", body, resp.StatusCode)
		}
	}

	// Non-admin refused.
	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	aliceToken, _ := alice["token"].(string)
	if resp, _ := postRawWithToken(t, ts, "/api/admin/email-settings", aliceToken, map[string]any{}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin = %d, want 403", resp.StatusCode)
	}
}
