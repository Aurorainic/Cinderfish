package httpapi_test

import (
	"net/http"
	"testing"
)

// TestOnlineUsersCountEndpoint proves the count follows authenticated activity
// and stays readable without a credential, the way Misskey's endpoint does.
func TestOnlineUsersCountEndpoint(t *testing.T) {
	ts := newTestServer(t, "open")

	resp, body := post(t, ts, "/api/get-online-users-count", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anonymous count: status %d body %v", resp.StatusCode, body)
	}
	if n, _ := body["count"].(float64); n != 0 {
		t.Fatalf("count before any sign-in = %v, want 0", body["count"])
	}

	_, aliceToken := createUser(t, ts, "alice")
	// /api/i authenticates the token, which records the activity.
	if resp, body := post(t, ts, "/api/i", map[string]any{"i": aliceToken}); resp.StatusCode != http.StatusOK {
		t.Fatalf("i: status %d body %v", resp.StatusCode, body)
	}

	resp, body = post(t, ts, "/api/get-online-users-count", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("count after sign-in: status %d body %v", resp.StatusCode, body)
	}
	if n, _ := body["count"].(float64); n != 1 {
		t.Fatalf("count after alice = %v, want 1", body["count"])
	}

	_, bobToken := createUser(t, ts, "bob")
	post(t, ts, "/api/i", map[string]any{"i": bobToken})
	_, body = post(t, ts, "/api/get-online-users-count", map[string]any{})
	if n, _ := body["count"].(float64); n != 2 {
		t.Fatalf("count after bob = %v, want 2", body["count"])
	}

	// The GET alias Misskey advertises (allowGet) answers the same shape.
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/get-online-users-count", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	got, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer got.Body.Close()
	if got.StatusCode != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", got.StatusCode)
	}
}
