package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestAppSessionFlow walks the legacy authorization flow: an app registers,
// opens a session, the user approves it, and the app collects a working token.
func TestAppSessionFlow(t *testing.T) {
	ts := newTestServer(t, "open")

	// The app registers and receives its secret.
	resp, app := post(t, ts, "/api/app/create", map[string]any{
		"name":        "Test App",
		"description": "does things",
		"permission":  []string{"read:account", "write:notes"},
		"callbackUrl": "https://app.example/callback",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("app/create = %d %v", resp.StatusCode, app)
	}
	appID, _ := app["id"].(string)
	secret, _ := app["secret"].(string)
	if appID == "" || secret == "" {
		t.Fatalf("app/create returned no id or secret: %v", app)
	}

	// Generating a session needs the secret.
	resp, generated := post(t, ts, "/api/auth/session/generate", map[string]any{"appSecret": secret})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("session/generate = %d %v", resp.StatusCode, generated)
	}
	sessionToken, _ := generated["token"].(string)
	url, _ := generated["url"].(string)
	if sessionToken == "" || !strings.HasSuffix(url, "/auth/"+sessionToken) {
		t.Fatalf("session/generate = %v", generated)
	}

	// The approval page reads the session.
	resp, shown := post(t, ts, "/api/auth/session/show", map[string]any{"token": sessionToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("session/show = %d %v", resp.StatusCode, shown)
	}
	shownApp, _ := shown["app"].(map[string]any)
	if shownApp["name"] != "Test App" {
		t.Errorf("session app = %v", shown["app"])
	}
	if _, leaked := shownApp["secret"]; leaked {
		t.Error("session/show must not expose the app secret")
	}
	if shownApp["isAuthorized"] != false {
		t.Errorf("isAuthorized = %v, want false", shownApp["isAuthorized"])
	}

	// Before approval the client is told the session is pending.
	resp, pending := post(t, ts, "/api/auth/session/userkey", map[string]any{
		"appSecret": secret, "token": sessionToken,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("pending userkey = %d %v", resp.StatusCode, pending)
	}
	if errObj, _ := pending["error"].(map[string]any); errObj["code"] != "PENDING_SESSION" {
		t.Errorf("pending error = %v", pending["error"])
	}

	// A user approves the session.
	username := uniqueName(t)
	_, signup := post(t, ts, "/api/signup", map[string]any{"username": username, "password": "hunter2"})
	userToken := signup["token"].(string)
	// auth/accept answers 204 with no body, so it cannot go through the JSON
	// helpers, which insist on decoding a response.
	acceptBody, _ := json.Marshal(map[string]any{"i": userToken, "token": sessionToken})
	acceptResp, err := ts.Client().Post(ts.URL+"/api/auth/accept", "application/json", bytes.NewReader(acceptBody))
	if err != nil {
		t.Fatalf("auth/accept: %v", err)
	}
	acceptResp.Body.Close()
	if acceptResp.StatusCode != http.StatusNoContent {
		t.Fatalf("auth/accept = %d, want 204", acceptResp.StatusCode)
	}

	// The app collects the token.
	resp, collected := post(t, ts, "/api/auth/session/userkey", map[string]any{
		"appSecret": secret, "token": sessionToken,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("userkey = %d %v", resp.StatusCode, collected)
	}
	accessToken, _ := collected["accessToken"].(string)
	if accessToken == "" {
		t.Fatalf("no accessToken: %v", collected)
	}
	user, _ := collected["user"].(map[string]any)
	if user["username"] != username {
		t.Errorf("userkey user = %v", user["username"])
	}

	// The token authenticates as the approving user.
	resp, me := post(t, ts, "/api/i", map[string]any{"i": accessToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i with app token = %d %v", resp.StatusCode, me)
	}
	if me["username"] != username {
		t.Errorf("i username = %v", me["username"])
	}

	// A session is collected once.
	resp, _ = post(t, ts, "/api/auth/session/userkey", map[string]any{
		"appSecret": secret, "token": sessionToken,
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("second userkey = %d, want 404", resp.StatusCode)
	}

	// i/apps reports the app's name and permissions for the issued token.
	listed := postListWithToken(t, ts, "/api/i/apps", userToken, nil)
	if len(listed) != 1 {
		t.Fatalf("i/apps = %v, want one entry", listed)
	}
	entry, _ := listed[0].(map[string]any)
	if entry["name"] != "Test App" {
		t.Errorf("i/apps name = %v", entry["name"])
	}
	perms, _ := entry["permission"].([]any)
	if len(perms) != 2 {
		t.Errorf("i/apps permission = %v", entry["permission"])
	}
}

// TestAuthSessionGenerateRejectsUnknownSecret keeps the app lookup honest.
func TestAuthSessionGenerateRejectsUnknownSecret(t *testing.T) {
	ts := newTestServer(t, "open")
	resp, body := post(t, ts, "/api/auth/session/generate", map[string]any{"appSecret": "nope"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("generate with bad secret = %d %v", resp.StatusCode, body)
	}
	if errObj, _ := body["error"].(map[string]any); errObj["code"] != "NO_SUCH_APP" {
		t.Errorf("error = %v", body["error"])
	}
}
