package httpapi_test

import (
	"net/http"
	"testing"
)

func TestMiAuthCheckHandsTokenOnce(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	aliceToken, _ := alice["token"].(string)

	resp, out := postWithToken(t, ts, "/api/miauth/gen-token", aliceToken, map[string]any{
		"session": "w15-session", "name": "W15 App",
		"permission": []string{"read:notes", "write:notes"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("gen-token: status %d, body %v", resp.StatusCode, out)
	}
	appToken, _ := out["token"].(string)
	if appToken == "" {
		t.Fatalf("gen-token returned no token: %v", out)
	}

	// The application polls without any credential.
	resp, raw := postRaw(t, ts, "/api/miauth/w15-session/check", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("check: status %d, body %v", resp.StatusCode, raw)
	}
	first, _ := raw.(map[string]any)
	if first["ok"] != true || first["token"] != appToken {
		t.Fatalf("first check = %v, want ok with the issued token", first)
	}
	user, _ := first["user"].(map[string]any)
	if user["username"] != "alice" {
		t.Errorf("check user = %v, want alice", user["username"])
	}

	// A second poll must not leak the token again.
	resp, raw = postRaw(t, ts, "/api/miauth/w15-session/check", map[string]any{})
	second, _ := raw.(map[string]any)
	if second["ok"] != false {
		t.Errorf("second check = %v, want ok=false", second)
	}

	// Unknown sessions answer the same way, not with an error.
	resp, raw = postRaw(t, ts, "/api/miauth/never-approved/check", map[string]any{})
	unknown, _ := raw.(map[string]any)
	if resp.StatusCode != http.StatusOK || unknown["ok"] != false {
		t.Errorf("unknown session = %d %v", resp.StatusCode, unknown)
	}
}
