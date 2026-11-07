package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/config"
	"github.com/Aurorainic/cinderfish/internal/httpapi"
)

// newSigninLimitServer is a database-backed server whose signin cap is
// small enough to reach in a test.
func newSigninLimitServer(t *testing.T, cap int) (*httptest.Server, *config.Config) {
	t.Helper()
	pool := testPool(t)
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: "fish.example",
		RegistrationMode: "open", FederationMode: "public",
		Media:           config.Media{Backend: "local"},
		SigninRateLimit: cap,
	}
	srv := httpapi.NewServerWithDB(cfg, pool)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts, cfg
}

func postSignin(t *testing.T, ts *httptest.Server, username, password string) (*http.Response, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(map[string]any{"username": username, "password": password}); err != nil {
		t.Fatalf("encode body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/signin", &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return resp, decoded
}

// TestSigninRateLimit proves the per-IP signin cap answers 429 once the
// configured hourly budget is spent, and that the one-second spacing
// rejects a burst within the budget.
func TestSigninRateLimit(t *testing.T) {
	ts, _ := newSigninLimitServer(t, 5)
	name := uniqueName(t)
	resp, signup := post(t, ts, "/api/signup", map[string]any{"username": name, "password": "password-123"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("signup = %d %v", resp.StatusCode, signup)
	}

	// The first attempts fit the budget; a wrong password answers 403.
	resp, _ = postSignin(t, ts, name, "wrong-1")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("first wrong signin = %d, want 403", resp.StatusCode)
	}
	// A same-second burst inside the budget is swallowed by the spacing
	// check, also as 403.
	resp, _ = postSignin(t, ts, name, "wrong-2")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("burst signin = %d, want 403", resp.StatusCode)
	}

	// Spend the rest of the budget; the next attempt is the 429.
	for i := 0; i < 3; i++ {
		resp, _ = postSignin(t, ts, name, "wrong-3")
		if resp.StatusCode == http.StatusTooManyRequests {
			break
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("signin %d = %d, want 403 before exhaustion", i, resp.StatusCode)
		}
	}
	resp, body := postSignin(t, ts, name, "wrong-4")
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("exhausted signin = %d %v, want 429", resp.StatusCode, body)
	}
	if body["error"] == nil {
		t.Fatalf("429 response has no error envelope: %v", body)
	}
	if resp.Header.Get("Retry-After") == "" {
		t.Error("429 response carries no Retry-After")
	}

	// Even the correct password is refused while the window is spent.
	resp, _ = postSignin(t, ts, name, "password-123")
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("correct password after exhaustion = %d, want 429", resp.StatusCode)
	}
}

// TestSigninRateLimitDisabled proves a negative configuration removes the
// cap, which is what the test servers rely on.
func TestSigninRateLimitDisabled(t *testing.T) {
	ts, _ := newSigninLimitServer(t, -1)
	name := uniqueName(t)
	if resp, _ := post(t, ts, "/api/signup", map[string]any{"username": name, "password": "password-123"}); resp.StatusCode != http.StatusOK {
		t.Fatal("signup failed")
	}
	for i := 0; i < 12; i++ {
		resp, _ := postSignin(t, ts, name, "password-123")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("signin %d = %d, want unlimited 200", i, resp.StatusCode)
		}
	}
}
