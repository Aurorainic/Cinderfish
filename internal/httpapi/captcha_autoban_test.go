package httpapi_test

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestSignupCaptcha covers the captcha settings round-trip and the signup
// gate: tokens are verified against the fixed provider endpoint (a stub
// transport stands in for the network), and a missing or rejected token
// creates nothing.
func TestSignupCaptcha(t *testing.T) {
	ts, srv := adminTestServer(t, "open")

	// Configure hCaptcha; the secret is write-only.
	resp, raw := postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"captcha": map[string]any{
			"provider": "hcaptcha", "siteKey": "site-key", "secret": "secret-key",
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update captcha = %d, body %v", resp.StatusCode, raw)
	}
	admin, _ := raw.(map[string]any)
	if admin["captchaProvider"] != "hcaptcha" || admin["captchaSiteKey"] != "site-key" || admin["captchaSecretSet"] != true {
		t.Errorf("admin meta captcha = %v", admin)
	}
	if _, leaked := admin["captchaSecret"]; leaked {
		t.Error("admin meta leaked the captcha secret")
	}

	// The public meta exposes the flag and site key for the client widget.
	_, meta := post(t, ts, "/api/meta", map[string]any{})
	if meta["enableHcaptcha"] != true || meta["hcaptchaSiteKey"] != "site-key" {
		t.Errorf("public meta captcha = %v / %v", meta["enableHcaptcha"], meta["hcaptchaSiteKey"])
	}
	if meta["enableRecaptcha"] != false || meta["enableTurnstile"] != false {
		t.Error("other captcha providers must stay disabled")
	}

	// Without a token signup is refused before anything is created.
	resp, body := post(t, ts, "/api/signup", map[string]any{
		"username": "no-captcha", "password": "password123",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("signup without captcha = %d, want 400", resp.StatusCode)
	}
	if errBody, _ := body["error"].(map[string]any); errBody["code"] != "CAPTCHA_FAILED" {
		t.Errorf("error = %v, want CAPTCHA_FAILED", body["error"])
	}

	// A rejecting provider also refuses.
	var reject atomic.Bool
	reject.Store(true)
	srv.SetCaptchaHTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.HasPrefix(r.URL.String(), "https://hcaptcha.com/") {
			t.Errorf("unexpected captcha endpoint %s", r.URL)
		}
		result := `{"success":true}`
		if reject.Load() {
			result = `{"success":false,"error-codes":["invalid-input-response"]}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(result)),
		}, nil
	})})
	resp, body = post(t, ts, "/api/signup", map[string]any{
		"username": "rejected", "password": "password123", "hcaptcha-response": "token",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("rejected captcha signup = %d, want 400 (body %v)", resp.StatusCode, body)
	}

	// With the provider accepting, signup goes through.
	reject.Store(false)
	resp, body = post(t, ts, "/api/signup", map[string]any{
		"username": "captchaok", "password": "password123", "hcaptcha-response": "token",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("accepted captcha signup = %d, body %v", resp.StatusCode, body)
	}

	// Switching providers requires the new secret.
	if resp, _ := postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"captcha": map[string]any{"provider": "turnstile", "siteKey": "ts-key"},
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("provider switch without secret = %d, want 400", resp.StatusCode)
	}

	// Disabling clears everything.
	resp, raw = postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"captcha": map[string]any{"provider": "none"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("disable captcha = %d", resp.StatusCode)
	}
	disabled, _ := raw.(map[string]any)
	if disabled["captchaProvider"] != "none" || disabled["captchaSecretSet"] != false {
		t.Errorf("disabled captcha = %v", disabled)
	}
	_, meta = post(t, ts, "/api/meta", map[string]any{})
	if meta["enableHcaptcha"] != false {
		t.Error("public meta still enables hCaptcha after disabling")
	}
}

// TestSigninAutoBan covers the automatic IP ban: after the configured number
// of failed signins from one address, a ban is created and the next attempt
// is refused with IP_BANNED.
func TestSigninAutoBan(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	// The default is disabled: three failures do not create a ban.
	for range 3 {
		post(t, ts, "/api/signin", map[string]any{"username": "ghost", "password": "nope"})
	}
	if bans := postListWithToken(t, ts, "/api/admin/ip-bans/list", "admintoken000000", nil); len(bans) != 0 {
		t.Fatalf("bans = %d with auto-ban disabled, want 0", len(bans))
	}

	resp, raw := postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"signinAutoBan": map[string]any{
			"enabled": true, "threshold": 3, "windowMinutes": 60, "banMinutes": 30,
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("enable auto-ban = %d, body %v", resp.StatusCode, raw)
	}
	settings, _ := raw.(map[string]any)
	autoBan, _ := settings["signinAutoBan"].(map[string]any)
	if autoBan["enabled"] != true || autoBan["threshold"] != float64(3) {
		t.Errorf("signinAutoBan = %v", settings["signinAutoBan"])
	}

	// The three failures above are outside the window? They are within it,
	// so the next failure trips the rule: threshold 3 of 4 total.
	post(t, ts, "/api/signin", map[string]any{"username": "ghost", "password": "nope"})
	bans := postListWithToken(t, ts, "/api/admin/ip-bans/list", "admintoken000000", nil)
	if len(bans) != 1 {
		t.Fatalf("bans = %d, want the automatic rule", len(bans))
	}
	rule, _ := bans[0].(map[string]any)
	if rule["ip"] != "127.0.0.1" {
		t.Errorf("ban ip = %v, want 127.0.0.1", rule["ip"])
	}
	if reason, _ := rule["reason"].(string); !strings.Contains(reason, "自动封禁") {
		t.Errorf("ban reason = %v, want the automatic marker", rule["reason"])
	}

	// The next signin from this address is refused outright.
	resp, body := post(t, ts, "/api/signin", map[string]any{"username": "ghost", "password": "nope"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("post-ban signin = %d, want 403", resp.StatusCode)
	}
	if errBody, _ := body["error"].(map[string]any); errBody["code"] != "IP_BANNED" {
		t.Errorf("post-ban error = %v, want IP_BANNED", body["error"])
	}

	// Validation: threshold below 3 and an over-long window are refused.
	for _, body := range []map[string]any{
		{"signinAutoBan": map[string]any{"threshold": 2}},
		{"signinAutoBan": map[string]any{"windowMinutes": 0}},
		{"signinAutoBan": map[string]any{"banMinutes": 50000}},
	} {
		if resp, _ := postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", body); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("invalid auto-ban %v = %d, want 400", body, resp.StatusCode)
		}
	}
}
