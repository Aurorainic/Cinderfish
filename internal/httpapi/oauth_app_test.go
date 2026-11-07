package httpapi_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// TestOAuthAuthorizeWithoutSession covers the no-web-build branch: a browser
// asking for the consent page without a credential gets 401 HTML, not a
// redirect or a JSON error.
func TestOAuthAuthorizeWithoutSession(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	q := url.Values{}
	q.Set("client_id", "https://client.example/app")
	q.Set("redirect_uri", "https://client.example/callback")
	q.Set("response_type", "code")
	q.Set("scope", "read write")
	q.Set("code_challenge", "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")
	q.Set("code_challenge_method", "S256")
	q.Set("state", "xyz")
	resp, err := ts.Client().Get(ts.URL + "/oauth/authorize?" + q.Encode())
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("authorize = %d, want 401", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content type = %q, want text/html", ct)
	}
}

// TestAppShowSecretOwnerOnly proves the secret is returned to the owning user
// only, and that an unknown app answers NO_SUCH_APP.
func TestAppShowSecretOwnerOnly(t *testing.T) {
	ts, _ := adminTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t)+"a")
	_, bobToken := createUser(t, ts, uniqueName(t)+"b")

	resp, created := postWithToken(t, ts, "/api/app/create", aliceToken, map[string]any{
		"name": "Owner Test", "description": "d", "permission": []string{"read"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("app/create = %d %v", resp.StatusCode, created)
	}
	appID, _ := created["id"].(string)
	if appID == "" || created["secret"] == "" {
		t.Fatalf("app/create returned no id/secret: %v", created)
	}

	// The owner with a native user token sees the secret.
	_, owner := postWithToken(t, ts, "/api/app/show", aliceToken, map[string]any{"appId": appID})
	if owner["secret"] != created["secret"] {
		t.Errorf("owner secret = %v, want %v", owner["secret"], created["secret"])
	}

	// Another user and an anonymous caller do not.
	_, other := postWithToken(t, ts, "/api/app/show", bobToken, map[string]any{"appId": appID})
	if other["secret"] != nil {
		t.Errorf("other user saw the secret: %v", other["secret"])
	}
	_, anon := post(t, ts, "/api/app/show", map[string]any{"appId": appID})
	if anon["secret"] != nil {
		t.Errorf("anonymous caller saw the secret: %v", anon["secret"])
	}

	// An unknown id is 404 NO_SUCH_APP.
	resp, missing := post(t, ts, "/api/app/show", map[string]any{"appId": "000000000000000000000000"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown app = %d %v, want 404", resp.StatusCode, missing)
	}
	if e, _ := missing["error"].(map[string]any); e["code"] != "NO_SUCH_APP" {
		t.Errorf("unknown app code = %v, want NO_SUCH_APP", e["code"])
	}
}

// TestAppShowSecretHidesFromAccessTokens proves an app-session access token
// does not see the secret even when it belongs to the owning user; only the
// native user token does.
func TestAppShowSecretHidesFromAccessTokens(t *testing.T) {
	ts, _ := adminTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t)+"a")

	resp, created := postWithToken(t, ts, "/api/app/create", aliceToken, map[string]any{
		"name": "Access Token App", "permission": []string{"read"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("app/create = %d %v", resp.StatusCode, created)
	}
	appID, _ := created["id"].(string)
	appSecret, _ := created["secret"].(string)

	resp, session := post(t, ts, "/api/auth/session/generate", map[string]any{"appSecret": appSecret})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("session/generate = %d %v", resp.StatusCode, session)
	}
	sessionToken, _ := session["token"].(string)
	resp, issued := postWithToken(t, ts, "/api/miauth/gen-token", aliceToken, map[string]any{
		"session": sessionToken, "name": "Access Token App", "permission": []string{"read"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("gen-token = %d %v", resp.StatusCode, issued)
	}
	accessToken, _ := issued["token"].(string)

	_, shown := postWithToken(t, ts, "/api/app/show", accessToken, map[string]any{"appId": appID})
	if shown["secret"] != nil {
		t.Errorf("access token saw the secret: %v", shown["secret"])
	}
	_, native := postWithToken(t, ts, "/api/app/show", aliceToken, map[string]any{"appId": appID})
	if native["secret"] == nil {
		t.Errorf("native token did not see the secret")
	}
}
