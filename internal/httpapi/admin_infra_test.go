package httpapi_test

import (
	"net/http"
	"testing"
)

func TestAdminObjectStorageReadOnly(t *testing.T) {
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
