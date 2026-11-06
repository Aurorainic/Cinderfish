package httpapi_test

import (
	"net/http"
	"testing"
)

// TestGetSoundsEndpoint answers the settings picker without a credential and
// always starts with the "none" entry.
func TestGetSoundsEndpoint(t *testing.T) {
	ts := newTestServer(t, "open")
	resp, raw := postRaw(t, ts, "/api/get-sounds", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		t.Fatalf("sounds = %v, want an array with the none entry", raw)
	}
	if list[0] != nil {
		t.Errorf("first entry = %v, want null", list[0])
	}
}
