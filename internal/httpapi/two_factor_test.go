package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/auth"
)

// TestTwoFactorLifecycle covers the TOTP endpoints and the signin gate they
// control: register, prove a code, sign in with password + code, refuse a bad
// code, and unregister.
func TestTwoFactorLifecycle(t *testing.T) {
	ts := newTestServer(t, "open")
	_, signup := post(t, ts, "/api/signup", map[string]any{"username": uniqueName(t), "password": "hunter2"})
	token, _ := signup["token"].(string)
	username, _ := signup["username"].(string)
	if token == "" || username == "" {
		t.Fatalf("signup returned no token/username: %v", signup)
	}

	// done before register is refused.
	resp, out := postWithToken(t, ts, "/api/i/2fa/done", token, map[string]any{"token": "123456"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("done before register = %d %v, want 400", resp.StatusCode, out)
	}
	if e, _ := out["error"].(map[string]any); e["code"] != "TWO_FACTOR_NOT_STARTED" {
		t.Errorf("code = %v, want TWO_FACTOR_NOT_STARTED", e["code"])
	}

	// register needs the password.
	resp, _ = postWithToken(t, ts, "/api/i/2fa/register", token, map[string]any{"password": "wrong"})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("register wrong password = %d, want 403", resp.StatusCode)
	}
	resp, registered := postWithToken(t, ts, "/api/i/2fa/register", token, map[string]any{"password": "hunter2"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register = %d %v", resp.StatusCode, registered)
	}
	secret, _ := registered["secret"].(string)
	if secret == "" {
		t.Fatalf("register returned no secret: %v", registered)
	}
	if url, _ := registered["url"].(string); url == "" || registered["qr"] != "" {
		t.Errorf("register url/qr = %v/%v", registered["url"], registered["qr"])
	}
	// The authenticator app shows the issuer, so it must be the public domain
	// rather than the listen address in cfg.Host.
	if registered["issuer"] != "fish.example" {
		t.Errorf("issuer = %v, want fish.example", registered["issuer"])
	}

	// A wrong code does not enable it.
	resp, out = postWithToken(t, ts, "/api/i/2fa/done", token, map[string]any{"token": "000000"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("done wrong code = %d %v, want 400", resp.StatusCode, out)
	}
	if e, _ := out["error"].(map[string]any); e["code"] != "TOTP_TOKEN_INVALID" {
		t.Errorf("wrong code = %v, want TOTP_TOKEN_INVALID", e["code"])
	}

	code, err := auth.TOTPCode(secret, time.Now())
	if err != nil {
		t.Fatalf("totp code: %v", err)
	}
	if status := postNoContentWithToken(t, ts, "/api/i/2fa/done", token, map[string]any{"token": code}); status != http.StatusNoContent {
		t.Fatalf("done = %d, want 204", status)
	}
	if _, me := postWithToken(t, ts, "/api/i", token, map[string]any{}); me["twoFactorEnabled"] != true {
		t.Errorf("twoFactorEnabled = %v, want true", me["twoFactorEnabled"])
	}

	// Password only: the dialog learns it must ask for a code.
	resp, out = post(t, ts, "/api/signin", map[string]any{"username": username, "password": "hunter2"})
	if resp.StatusCode != http.StatusOK || out["finished"] == true {
		t.Fatalf("password-only signin = %d %v, want an empty success", resp.StatusCode, out)
	}
	resp, out = post(t, ts, "/api/signin", map[string]any{"username": username, "password": "hunter2", "token": "000000"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong totp signin = %d %v, want 403", resp.StatusCode, out)
	}
	code, err = auth.TOTPCode(secret, time.Now())
	if err != nil {
		t.Fatalf("totp code: %v", err)
	}
	resp, out = post(t, ts, "/api/signin", map[string]any{"username": username, "password": "hunter2", "token": code})
	if resp.StatusCode != http.StatusOK || out["finished"] != true || out["i"] == nil {
		t.Fatalf("signin with code = %d %v", resp.StatusCode, out)
	}

	// Unregister needs the right password and clears the gate.
	resp, _ = postWithToken(t, ts, "/api/i/2fa/unregister", token, map[string]any{"password": "wrong"})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("unregister wrong password = %d, want 403", resp.StatusCode)
	}
	if status := postNoContentWithToken(t, ts, "/api/i/2fa/unregister", token, map[string]any{"password": "hunter2"}); status != http.StatusNoContent {
		t.Fatalf("unregister = %d, want 204", status)
	}
	if _, me := postWithToken(t, ts, "/api/i", token, map[string]any{}); me["twoFactorEnabled"] != false {
		t.Errorf("twoFactorEnabled after unregister = %v", me["twoFactorEnabled"])
	}
	resp, out = post(t, ts, "/api/signin", map[string]any{"username": username, "password": "hunter2"})
	if resp.StatusCode != http.StatusOK || out["finished"] != true {
		t.Errorf("signin after unregister = %d %v, want finished", resp.StatusCode, out)
	}

	// Security keys are not supported, so password-less stays off.
	resp, out = postWithToken(t, ts, "/api/i/2fa/password-less", token, map[string]any{"value": true})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("password-less on = %d %v, want 400", resp.StatusCode, out)
	}
	if e, _ := out["error"].(map[string]any); e["code"] != "NO_SECURITY_KEY" {
		t.Errorf("password-less code = %v", e["code"])
	}
	if status := postNoContentWithToken(t, ts, "/api/i/2fa/password-less", token, map[string]any{"value": false}); status != http.StatusNoContent {
		t.Errorf("password-less off = %d, want 204", status)
	}
}
