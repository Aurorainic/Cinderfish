package httpapi_test

import (
	"net/http"
	"testing"
)

// TestRegistryRoundTrip pins i/registry set/get/get-detail/keys/get-all/
// remove/clear, including scope isolation and the NO_SUCH_KEY answer.
func TestRegistryRoundTrip(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	scope := []string{"client", "theme"}
	payload := map[string]any{"i": token, "key": "colors", "scope": scope, "value": map[string]any{"accent": "#86b300"}}

	if resp, out := post(t, ts, "/api/i/registry/set", payload); resp.StatusCode != http.StatusOK {
		t.Fatalf("set: %d %v", resp.StatusCode, out)
	}

	// get returns the raw value.
	_, value := postRaw(t, ts, "/api/i/registry/get", map[string]any{"i": token, "key": "colors", "scope": scope})
	got, _ := value.(map[string]any)
	if got["accent"] != "#86b300" {
		t.Errorf("get value = %v", value)
	}

	// get-detail adds updatedAt.
	_, detail := post(t, ts, "/api/i/registry/get-detail", map[string]any{"i": token, "key": "colors", "scope": scope})
	if detail["updatedAt"] == nil {
		t.Errorf("get-detail has no updatedAt: %v", detail)
	}
	if v, _ := detail["value"].(map[string]any); v["accent"] != "#86b300" {
		t.Errorf("get-detail value = %v", detail["value"])
	}

	// A different scope does not see the key.
	if resp, _ := post(t, ts, "/api/i/registry/get", map[string]any{"i": token, "key": "colors", "scope": []string{"other"}}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("other scope get = %d, want 404", resp.StatusCode)
	}
	// ... and the scope order does not matter.
	if resp, _ := post(t, ts, "/api/i/registry/get", map[string]any{"i": token, "key": "colors", "scope": []string{"theme", "client"}}); resp.StatusCode != http.StatusOK {
		t.Errorf("reordered scope get = %d, want 200", resp.StatusCode)
	}

	// keys and get-all list the scope.
	keys := postListWithToken(t, ts, "/api/i/registry/keys", token, map[string]any{"scope": scope})
	if len(keys) != 1 || keys[0] != "colors" {
		t.Errorf("keys = %v, want [colors]", keys)
	}
	// scopes reports the distinct scope it was written under, normalised.
	scopes := postListWithToken(t, ts, "/api/i/registry/scopes", token, map[string]any{})
	if len(scopes) != 1 {
		t.Fatalf("scopes = %v, want one", scopes)
	}
	first, _ := scopes[0].([]any)
	if len(first) != 2 || first[0] != "client" || first[1] != "theme" {
		t.Errorf("scopes[0] = %v, want [client theme]", scopes[0])
	}
	_, all := post(t, ts, "/api/i/registry/get-all", map[string]any{"i": token, "scope": scope})
	if v, _ := all["colors"].(map[string]any); v["accent"] != "#86b300" {
		t.Errorf("get-all = %v", all)
	}

	// An overwrite keeps one row.
	if resp, out := post(t, ts, "/api/i/registry/set", map[string]any{"i": token, "key": "colors", "scope": scope, "value": map[string]any{"accent": "#000000"}}); resp.StatusCode != http.StatusOK {
		t.Fatalf("overwrite: %d %v", resp.StatusCode, out)
	}
	if keys := postListWithToken(t, ts, "/api/i/registry/keys", token, map[string]any{"scope": scope}); len(keys) != 1 {
		t.Errorf("keys after overwrite = %v, want one", keys)
	}

	// remove then clear.
	if resp, out := post(t, ts, "/api/i/registry/remove", map[string]any{"i": token, "key": "colors", "scope": scope}); resp.StatusCode != http.StatusOK {
		t.Fatalf("remove: %d %v", resp.StatusCode, out)
	}
	if resp, _ := post(t, ts, "/api/i/registry/get", map[string]any{"i": token, "key": "colors", "scope": scope}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("get after remove = %d, want 404", resp.StatusCode)
	}
	if resp, _ := post(t, ts, "/api/i/registry/set", map[string]any{"i": token, "key": "a", "scope": scope, "value": 1}); resp.StatusCode != http.StatusOK {
		t.Fatalf("set a: %d", resp.StatusCode)
	}
	if resp, _ := post(t, ts, "/api/i/registry/set", map[string]any{"i": token, "key": "b", "scope": scope, "value": 2}); resp.StatusCode != http.StatusOK {
		t.Fatalf("set b: %d", resp.StatusCode)
	}
	if resp, out := post(t, ts, "/api/i/registry/clear", map[string]any{"i": token, "scope": scope}); resp.StatusCode != http.StatusOK {
		t.Fatalf("clear: %d %v", resp.StatusCode, out)
	}
	if keys := postListWithToken(t, ts, "/api/i/registry/keys", token, map[string]any{"scope": scope}); len(keys) != 0 {
		t.Errorf("keys after clear = %v, want none", keys)
	}
	if scopes := postListWithToken(t, ts, "/api/i/registry/scopes", token, map[string]any{}); len(scopes) != 0 {
		t.Errorf("scopes after clear = %v, want none", scopes)
	}
}

// TestRegistryIsolationAndErrors proves entries are per user, a missing key
// answers NO_SUCH_KEY with the endpoint's id, and anonymous calls 401.
func TestRegistryIsolationAndErrors(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t))
	_, bobToken := createUser(t, ts, uniqueName(t)+"b")

	scope := []string{"client"}
	if resp, _ := post(t, ts, "/api/i/registry/set", map[string]any{"i": aliceToken, "key": "secret", "scope": scope, "value": "mine"}); resp.StatusCode != http.StatusOK {
		t.Fatalf("alice set: %d", resp.StatusCode)
	}
	if resp, _ := post(t, ts, "/api/i/registry/get", map[string]any{"i": bobToken, "key": "secret", "scope": scope}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("bob get = %d, want 404", resp.StatusCode)
	}

	resp, out := post(t, ts, "/api/i/registry/get", map[string]any{"i": bobToken, "key": "missing", "scope": scope})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing key = %d %v, want 404", resp.StatusCode, out)
	}
	e, _ := out["error"].(map[string]any)
	if e["code"] != "NO_SUCH_KEY" || e["id"] != "ac3ed68a-62f0-422b-a7bc-d5e09e8f6a6a" {
		t.Errorf("error = %v, want NO_SUCH_KEY with the registry id", e)
	}

	if resp, _ := post(t, ts, "/api/i/registry/get", map[string]any{"key": "secret", "scope": scope}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous get = %d, want 401", resp.StatusCode)
	}
}
