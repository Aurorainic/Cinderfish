package httpapi_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/auth"
)

// TestOAuthAuthorizeConsentAPI proves the SPA consent call: an authenticated
// POST returns the client redirect, and the code it carries is redeemable.
func TestOAuthAuthorizeConsentAPI(t *testing.T) {
	ts := newTestServer(t, "open")
	_, signup := post(t, ts, "/api/signup", map[string]any{"username": uniqueName(t), "password": "hunter2"})
	token := signup["token"].(string)

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := auth.GenerateCodeChallenge(verifier)

	resp, body := post(t, ts, "/api/oauth/authorize", map[string]any{
		"i":                     token,
		"client_id":             "https://client.example/app",
		"redirect_uri":          "https://client.example/callback",
		"response_type":         "code",
		"code_challenge":        challenge,
		"code_challenge_method": "S256",
		"state":                 "abc",
		"scope":                 []string{"read", "write"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("consent = %d %v", resp.StatusCode, body)
	}
	redirect, _ := body["redirect"].(string)
	if redirect == "" {
		t.Fatalf("no redirect in %v", body)
	}
	parsed, err := url.Parse(redirect)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	code := parsed.Query().Get("code")
	if code == "" {
		t.Fatalf("no code in %s", redirect)
	}
	if parsed.Query().Get("state") != "abc" {
		t.Errorf("state = %q, want abc", parsed.Query().Get("state"))
	}

	// The returned code completes the normal token exchange.
	resp, tokenBody := postWithType(t, ts, "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {"https://client.example/app"},
		"redirect_uri":  {"https://client.example/callback"},
		"code_verifier": {verifier},
	}.Encode(), "application/x-www-form-urlencoded")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("token = %d %v", resp.StatusCode, tokenBody)
	}
	if tokenBody["access_token"] == nil {
		t.Errorf("no access_token: %v", tokenBody)
	}
}

// TestOAuthAuthorizeConsentAPIRequiresCredential keeps the consent call
// authenticated: an anonymous POST must not issue a code.
func TestOAuthAuthorizeConsentAPIRequiresCredential(t *testing.T) {
	ts := newTestServer(t, "open")
	resp, _ := post(t, ts, "/api/oauth/authorize", map[string]any{
		"client_id":             "https://client.example/app",
		"redirect_uri":          "https://client.example/callback",
		"response_type":         "code",
		"code_challenge":        auth.GenerateCodeChallenge("verifier"),
		"code_challenge_method": "S256",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous consent = %d, want 401", resp.StatusCode)
	}
}

// TestOAuthAuthorizeConsentAPIRejectsBadParams covers the shared validation:
// the POST path must refuse what the GET path refuses.
func TestOAuthAuthorizeConsentAPIRejectsBadParams(t *testing.T) {
	ts := newTestServer(t, "open")
	_, signup := post(t, ts, "/api/signup", map[string]any{"username": uniqueName(t), "password": "hunter2"})
	token := signup["token"].(string)

	resp, body := post(t, ts, "/api/oauth/authorize", map[string]any{
		"i":                     token,
		"client_id":             "https://client.example/app",
		"redirect_uri":          "https://client.example/callback",
		"response_type":         "code",
		"code_challenge":        "challenge",
		"code_challenge_method": "plain",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("plain PKCE = %d %v, want 400", resp.StatusCode, body)
	}
}
