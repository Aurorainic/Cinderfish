package httpapi_test

import (
	"context"
	"net/http"
	"testing"
)

// TestFederationShowInstance covers federation/show-instance: the local host is
// answered from live counts, a host this instance has seen is aggregated from
// stored rows, an unknown host is null, and the endpoint needs a credential.
func TestFederationShowInstance(t *testing.T) {
	f := newFederationServer(t, "public")
	token, _ := f.registerUser(t, "alice")

	// The local host is the public domain, answered with live counts. The
	// listen address in cfg.Host must not be treated as the instance's host.
	resp, local := postWithToken(t, f.ts, "/api/federation/show-instance", token, map[string]any{"host": "fish.example"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("local instance = %d %v", resp.StatusCode, local)
	}
	if local["host"] != "fish.example" || local["softwareName"] != "cinderfish" {
		t.Errorf("local instance = %v", local)
	}
	if local["usersCount"] != float64(1) {
		t.Errorf("usersCount = %v, want 1", local["usersCount"])
	}

	// A remote host is aggregated from what this instance already stores.
	if _, err := f.pool.Exec(context.Background(), `
INSERT INTO users (id, username, username_lower, host, uri)
VALUES ('w63peer', 'peer', 'peer', 'peer.example', 'https://peer.example/users/peer')`); err != nil {
		t.Fatalf("insert peer user: %v", err)
	}
	if _, err := f.pool.Exec(context.Background(), `
INSERT INTO notes (id, user_id, user_host, visibility, text)
VALUES ('w63peernote', 'w63peer', 'peer.example', 'public', 'from the peer')`); err != nil {
		t.Fatalf("insert peer note: %v", err)
	}
	resp, remote := postWithToken(t, f.ts, "/api/federation/show-instance", token, map[string]any{"host": "peer.example"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("remote instance = %d %v", resp.StatusCode, remote)
	}
	if remote["host"] != "peer.example" || remote["usersCount"] != float64(1) || remote["notesCount"] != float64(1) {
		t.Errorf("remote instance = %v", remote)
	}

	// A host never seen answers null rather than inventing a row.
	resp, raw := postRawWithToken(t, f.ts, "/api/federation/show-instance", token, map[string]any{"host": "never.example"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unknown host = %d %v", resp.StatusCode, raw)
	}
	if raw != nil {
		if _, isObj := raw.(map[string]any); isObj {
			t.Errorf("unknown host = %v, want null", raw)
		}
	}

	// Without a credential the endpoint refuses.
	if resp, _ := post(t, f.ts, "/api/federation/show-instance", map[string]any{"host": "peer.example"}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", resp.StatusCode)
	}
	// An empty host is a parameter error.
	if resp, _ := postWithToken(t, f.ts, "/api/federation/show-instance", token, map[string]any{}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty host = %d, want 400", resp.StatusCode)
	}
}
