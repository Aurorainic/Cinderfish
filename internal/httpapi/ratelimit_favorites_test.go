package httpapi_test

import (
	"net/http"
	"testing"
)

// TestFavoritesCreateRateLimit pins Misskey's 20-per-hour cap on
// notes/favorites/create, including the 429 shape and per-account isolation.
func TestFavoritesCreateRateLimit(t *testing.T) {
	ts := newTestServer(t, "open")
	_, ownerToken := createUser(t, ts, "owner")
	_, aliceToken := createUser(t, ts, "alice")
	noteID := createNote(t, ts, ownerToken, map[string]any{"text": "favorite me"})

	for i := 0; i < 20; i++ {
		resp, body := postWithToken(t, ts, "/api/notes/favorites/create", aliceToken, map[string]any{"noteId": noteID})
		if resp.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("call %d limited before the cap: %v", i+1, body)
		}
	}
	resp, body := postWithToken(t, ts, "/api/notes/favorites/create", aliceToken, map[string]any{"noteId": noteID})
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("21st call status = %d, want 429 (%v)", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Retry-After"); got == "" {
		t.Error("429 has no Retry-After header")
	}
	errBody, _ := body["error"].(map[string]any)
	if errBody["code"] != "RATE_LIMIT_EXCEEDED" {
		t.Errorf("code = %v, want RATE_LIMIT_EXCEEDED", errBody["code"])
	}

	// The cap is per account: a second user is not affected.
	if resp, body := postWithToken(t, ts, "/api/notes/favorites/create", ownerToken, map[string]any{"noteId": noteID}); resp.StatusCode == http.StatusTooManyRequests {
		t.Fatalf("another account was limited: %v", body)
	}
}
