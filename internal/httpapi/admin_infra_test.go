package httpapi_test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestAdminObjectStorage(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	resp, raw := postRawWithToken(t, ts, "/api/admin/object-storage", "admintoken000000", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("object-storage: status %d, body %v", resp.StatusCode, raw)
	}
	out, _ := raw.(map[string]any)
	if out["backend"] != "local" {
		t.Errorf("backend = %v, want local", out["backend"])
	}
	if _, ok := out["localPath"]; !ok {
		t.Errorf("localPath key missing: %v", out)
	}
	if out["s3"] != nil {
		t.Errorf("s3 = %v, want null for a local backend", out["s3"])
	}
	if out["stored"] != false {
		t.Errorf("stored = %v, want false before any runtime override", out["stored"])
	}
	// Secrets must never appear, even as empty strings.
	for _, key := range []string{"accessKey", "secretKey", "access_key", "secret_key"} {
		if _, leaked := out[key]; leaked {
			t.Errorf("object-storage leaked %q", key)
		}
	}

	// Non-admin is refused.
	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	aliceToken, _ := alice["token"].(string)
	resp, _ = postRawWithToken(t, ts, "/api/admin/object-storage", aliceToken, map[string]any{})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin status = %d, want 403", resp.StatusCode)
	}
}

// TestAdminObjectStorageRuntime covers saving a runtime media configuration:
// the override persists, wins over config.toml on re-read, and switches the
// live backend (verified through the drive's storage directory).
func TestAdminObjectStorageRuntime(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	dir := filepath.Join(t.TempDir(), "runtime-media")
	resp, raw := postRawWithToken(t, ts, "/api/admin/object-storage/update", "admintoken000000", map[string]any{
		"backend": "local", "localPath": dir,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update = %d, body %v", resp.StatusCode, raw)
	}
	updated, _ := raw.(map[string]any)
	if updated["backend"] != "local" || updated["localPath"] != dir || updated["stored"] != true {
		t.Errorf("update response = %v", updated)
	}

	// The override survives a re-read.
	_, raw = postRawWithToken(t, ts, "/api/admin/object-storage", "admintoken000000", map[string]any{})
	reread, _ := raw.(map[string]any)
	if reread["localPath"] != dir || reread["stored"] != true {
		t.Errorf("re-read = %v, want the stored override", reread)
	}

	// The switch is live: NewLocalStorage creates the directory when it
	// builds the backend, so the new path must exist after the update.
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("runtime media directory was not created: %v", err)
	}

	// Validation branches.
	cases := []map[string]any{
		{"backend": "bogus"},
		{"backend": "local", "localPath": ""},
		{"backend": "s3", "s3": map[string]any{"bucket": "b"}},
		{"backend": "s3", "s3": map[string]any{"endpoint": "s3.example.com"}},
	}
	for i, body := range cases {
		if resp, _ := postRawWithToken(t, ts, "/api/admin/object-storage/update", "admintoken000000", body); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("case %d status = %d, want 400", i, resp.StatusCode)
		}
	}

	// An S3 configuration is accepted without credentials (the fields are
	// optional; uploads fail at runtime if the bucket needs them).
	resp, raw = postRawWithToken(t, ts, "/api/admin/object-storage/update", "admintoken000000", map[string]any{
		"backend": "s3",
		"s3": map[string]any{
			"endpoint": "s3.example.com", "bucket": "media",
			"accessKey": "AK", "secretKey": "SK",
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("s3 update = %d, body %v", resp.StatusCode, raw)
	}
	s3, _ := raw.(map[string]any)
	if s3["backend"] != "s3" {
		t.Errorf("backend = %v, want s3", s3["backend"])
	}
	details, _ := s3["s3"].(map[string]any)
	if details == nil || details["bucket"] != "media" || details["hasCredentials"] != true {
		t.Errorf("s3 details = %v", s3["s3"])
	}
	for _, key := range []string{"accessKey", "secretKey"} {
		if _, leaked := details[key]; leaked {
			t.Errorf("s3 details leaked %q", key)
		}
	}

	// Saving again with an empty secret keeps the stored one.
	resp, raw = postRawWithToken(t, ts, "/api/admin/object-storage/update", "admintoken000000", map[string]any{
		"backend": "s3",
		"s3": map[string]any{
			"endpoint": "s3.example.com", "bucket": "media-2",
			"accessKey": "AK",
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("s3 re-update = %d, body %v", resp.StatusCode, raw)
	}
	again, _ := raw.(map[string]any)
	againS3, _ := again["s3"].(map[string]any)
	if againS3["bucket"] != "media-2" || againS3["hasCredentials"] != true {
		t.Errorf("re-update lost the stored secret: %v", again["s3"])
	}
}

func TestAdminDeliverQueueEmpty(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	resp, raw := postRawWithToken(t, ts, "/api/admin/deliver-queue", "admintoken000000", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("deliver-queue: status %d, body %v", resp.StatusCode, raw)
	}
	out, _ := raw.(map[string]any)
	if out["pending"] != float64(0) || out["failed"] != float64(0) {
		t.Errorf("queue counters = %v", out)
	}
	if recent, ok := out["recent"].([]any); !ok || len(recent) != 0 {
		t.Errorf("recent = %v, want an empty array", out["recent"])
	}
	if out["oldestNextAt"] != nil {
		t.Errorf("oldestNextAt = %v, want null", out["oldestNextAt"])
	}

	// Anonymous gets 401.
	resp, _ = postRaw(t, ts, "/api/admin/deliver-queue", map[string]any{})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous status = %d, want 401", resp.StatusCode)
	}
}
