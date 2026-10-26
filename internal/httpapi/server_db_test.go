package httpapi_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"milnfish/internal/auth"
	"milnfish/internal/config"
	"milnfish/internal/db"
	"milnfish/internal/httpapi"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("MILNFISH_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("MILNFISH_TEST_DATABASE_URL is not set; skipping database test")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
TRUNCATE users, user_profiles, access_tokens, registration_tickets, signins, oauth_grant_codes
RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return pool
}

func newTestServer(t *testing.T, mode string) *httptest.Server {
	t.Helper()
	pool := testPool(t)
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: "fish.example",
		RegistrationMode: mode, FederationMode: "public",
		Media: config.Media{Backend: "local"},
	}
	ts := httptest.NewServer(httpapi.NewServerWithDB(cfg, pool))
	t.Cleanup(ts.Close)
	return ts
}

func post(t *testing.T, ts *httptest.Server, path string, body any) (*http.Response, map[string]any) {
	t.Helper()
	return postWithType(t, ts, path, body, "application/json")
}

func postWithType(t *testing.T, ts *httptest.Server, path string, body any, contentType string) (*http.Response, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	switch v := body.(type) {
	case string:
		buf.WriteString(v)
	case nil:
	default:
		if err := json.NewEncoder(&buf).Encode(v); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })

	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode %s response: %v", path, err)
	}
	return resp, decoded
}

func uniqueName(t *testing.T) string {
	t.Helper()
	sum := sha256.Sum256([]byte(t.Name()))
	return "u" + hex.EncodeToString(sum[:])[:12]
}

func TestSignupAndSignin(t *testing.T) {
	ts := newTestServer(t, "open")
	name := uniqueName(t)

	resp, body := post(t, ts, "/api/signup", map[string]any{
		"username": name,
		"password": "hunter2",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("signup status = %d, body %v", resp.StatusCode, body)
	}
	token, _ := body["token"].(string)
	if token == "" {
		t.Fatalf("signup did not return a token: %v", body)
	}
	if len(token) != 16 {
		t.Errorf("token length = %d, want 16", len(token))
	}
	if body["username"] != name {
		t.Errorf("username = %v, want %q", body["username"], name)
	}

	resp, body = post(t, ts, "/api/signin", map[string]any{
		"username": name,
		"password": "hunter2",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("signin status = %d, body %v", resp.StatusCode, body)
	}
	if body["finished"] != true {
		t.Errorf("finished = %v, want true", body["finished"])
	}
	if body["i"] != token {
		t.Errorf("signin token = %v, want the signup token", body["i"])
	}
}

func TestSigninWrongPassword(t *testing.T) {
	ts := newTestServer(t, "open")
	name := uniqueName(t)
	post(t, ts, "/api/signup", map[string]any{"username": name, "password": "hunter2"})

	resp, body := post(t, ts, "/api/signin", map[string]any{
		"username": name,
		"password": "wrong",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	// Misskey's signin errors carry only an id.
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("no error envelope: %v", body)
	}
	if len(errObj) != 1 {
		t.Errorf("signin error should carry only an id, got %v", errObj)
	}
	if errObj["id"] != "932c904e-9460-45b7-9ce6-7ed33be7eb2c" {
		t.Errorf("error id = %v", errObj["id"])
	}
}

func TestSigninUnknownUser(t *testing.T) {
	ts := newTestServer(t, "open")
	resp, body := post(t, ts, "/api/signin", map[string]any{
		"username": "nosuchuser",
		"password": "hunter2",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if body["error"] == nil {
		t.Errorf("no error envelope: %v", body)
	}
}

func TestSignupAdminModeRefused(t *testing.T) {
	ts := newTestServer(t, "admin")
	resp, body := post(t, ts, "/api/signup", map[string]any{
		"username": uniqueName(t),
		"password": "hunter2",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "REGISTRATION_DISABLED" {
		t.Errorf("error code = %v", errObj["code"])
	}
}

func TestSignupInviteMode(t *testing.T) {
	ts := newTestServer(t, "invite")
	name := uniqueName(t)

	resp, body := post(t, ts, "/api/signup", map[string]any{
		"username": name, "password": "hunter2",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("signup without invite: status = %d, body %v", resp.StatusCode, body)
	}

	resp, body = post(t, ts, "/api/signup", map[string]any{
		"username": name, "password": "hunter2", "invitationCode": "nosuchcode",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("signup with a bad invite: status = %d", resp.StatusCode)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "INVALID_INVITE" {
		t.Errorf("error code = %v, want INVALID_INVITE", errObj["code"])
	}
}

func TestIEndpoint(t *testing.T) {
	ts := newTestServer(t, "open")
	name := uniqueName(t)
	_, signup := post(t, ts, "/api/signup", map[string]any{"username": name, "password": "hunter2"})
	token := signup["token"].(string)

	resp, body := post(t, ts, "/api/i", map[string]any{"i": token})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body %v", resp.StatusCode, body)
	}
	if body["username"] != name {
		t.Errorf("username = %v, want %q", body["username"], name)
	}
	// Native tokens include secrets.
	if _, ok := body["email"]; !ok {
		t.Error("a native token request must include the email field")
	}
}

func TestIEndpointRequiresCredential(t *testing.T) {
	ts := newTestServer(t, "open")
	resp, body := post(t, ts, "/api/i", map[string]any{})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "AUTHENTICATION_FAILED" {
		t.Errorf("error code = %v", errObj["code"])
	}
	if resp.Header.Get("WWW-Authenticate") == "" {
		t.Error("401 response is missing the WWW-Authenticate header")
	}
}

func TestIEndpointRejectsBadToken(t *testing.T) {
	ts := newTestServer(t, "open")
	resp, _ := post(t, ts, "/api/i", map[string]any{"i": "totally-invalid-token"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestUsersShow(t *testing.T) {
	ts := newTestServer(t, "open")
	name := uniqueName(t)
	_, signup := post(t, ts, "/api/signup", map[string]any{"username": name, "password": "hunter2"})
	userID := signup["id"].(string)

	resp, body := post(t, ts, "/api/users/show", map[string]any{"userId": userID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body %v", resp.StatusCode, body)
	}
	if body["username"] != name {
		t.Errorf("username = %v, want %q", body["username"], name)
	}

	resp, _ = post(t, ts, "/api/users/show", map[string]any{"username": name})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("lookup by username: status = %d", resp.StatusCode)
	}

	resp, _ = post(t, ts, "/api/users/show", map[string]any{"userId": "nosuchid"})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown user: status = %d, want 404", resp.StatusCode)
	}
}

func TestUsernameAvailable(t *testing.T) {
	ts := newTestServer(t, "open")
	name := uniqueName(t)

	resp, body := post(t, ts, "/api/username/available", map[string]any{"username": name})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if body["available"] != true {
		t.Errorf("available = %v, want true", body["available"])
	}

	post(t, ts, "/api/signup", map[string]any{"username": name, "password": "hunter2"})
	_, body = post(t, ts, "/api/username/available", map[string]any{"username": name})
	if body["available"] != false {
		t.Errorf("available after signup = %v, want false", body["available"])
	}
}

func TestMiAuthGenToken(t *testing.T) {
	ts := newTestServer(t, "open")
	name := uniqueName(t)
	_, signup := post(t, ts, "/api/signup", map[string]any{"username": name, "password": "hunter2"})
	token := signup["token"].(string)

	resp, body := post(t, ts, "/api/miauth/gen-token", map[string]any{
		"i":          token,
		"session":    "session-id",
		"name":       "test app",
		"permission": []string{"read"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body %v", resp.StatusCode, body)
	}
	miauth, _ := body["token"].(string)
	if miauth == "" {
		t.Fatalf("no token returned: %v", body)
	}
	if len(miauth) != 32 {
		t.Errorf("miauth token length = %d, want 32", len(miauth))
	}

	// The MiAuth token must work as a credential.
	resp, body = post(t, ts, "/api/i", map[string]any{"i": miauth})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i with miauth token: status = %d, body %v", resp.StatusCode, body)
	}
	if body["username"] != name {
		t.Errorf("username = %v", body["username"])
	}
	// A MiAuth token is not a native token, so secrets are withheld.
	if _, ok := body["email"]; ok {
		t.Error("a MiAuth token request must not include secrets")
	}
}

func TestOAuthTokenFlow(t *testing.T) {
	ts := newTestServer(t, "open")
	name := uniqueName(t)
	_, signup := post(t, ts, "/api/signup", map[string]any{"username": name, "password": "hunter2"})
	token := signup["token"].(string)

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := auth.GenerateCodeChallenge(verifier)

	// Authorize: the code is returned through a redirect to the client.
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/oauth/authorize?"+url.Values{
		"client_id":             {"https://client.example/app"},
		"redirect_uri":          {"https://client.example/callback"},
		"response_type":         {"code"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {"xyz"},
	}.Encode(), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := ts.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("authorize status = %d, want 302", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	code := parsed.Query().Get("code")
	if code == "" {
		t.Fatalf("no code in redirect: %s", location)
	}
	if parsed.Query().Get("state") != "xyz" {
		t.Errorf("state = %v, want xyz", parsed.Query().Get("state"))
	}

	// Exchange the code for a token.
	resp, body := postWithType(t, ts, "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {"https://client.example/app"},
		"redirect_uri":  {"https://client.example/callback"},
		"code_verifier": {verifier},
	}.Encode(), "application/x-www-form-urlencoded")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("token status = %d, body %v", resp.StatusCode, body)
	}
	accessToken, _ := body["access_token"].(string)
	if accessToken == "" {
		t.Fatalf("no access_token: %v", body)
	}
	if body["token_type"] != "Bearer" {
		t.Errorf("token_type = %v, want Bearer", body["token_type"])
	}

	// The access token must authenticate API calls.
	resp, body = post(t, ts, "/api/i", map[string]any{"i": accessToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i with oauth token: status = %d, body %v", resp.StatusCode, body)
	}

	// A code must not be redeemable twice.
	resp, _ = postWithType(t, ts, "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {"https://client.example/app"},
		"redirect_uri":  {"https://client.example/callback"},
		"code_verifier": {verifier},
	}.Encode(), "application/x-www-form-urlencoded")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("reusing a code: status = %d, want 400", resp.StatusCode)
	}
}

func TestOAuthTokenRejectsWrongVerifier(t *testing.T) {
	ts := newTestServer(t, "open")
	name := uniqueName(t)
	_, signup := post(t, ts, "/api/signup", map[string]any{"username": name, "password": "hunter2"})
	token := signup["token"].(string)

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := auth.GenerateCodeChallenge(verifier)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/oauth/authorize?"+url.Values{
		"client_id":             {"https://client.example/app"},
		"redirect_uri":          {"https://client.example/callback"},
		"response_type":         {"code"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}.Encode(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client := ts.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	resp.Body.Close()
	code := resp.Header.Get("Location")
	if code == "" {
		t.Fatal("no redirect")
	}
	parsed, _ := url.Parse(code)
	grantCode := parsed.Query().Get("code")

	resp, body := postWithType(t, ts, "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {grantCode},
		"client_id":     {"https://client.example/app"},
		"redirect_uri":  {"https://client.example/callback"},
		"code_verifier": {"wrong-verifier"},
	}.Encode(), "application/x-www-form-urlencoded")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body %v", resp.StatusCode, body)
	}
	// OAuth errors use the flat {"error":"invalid_grant"} shape.
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %v, want invalid_grant", body["error"])
	}
}

func TestOAuthAuthorizeRejectsPlainPKCE(t *testing.T) {
	ts := newTestServer(t, "open")
	resp, err := ts.Client().Get(ts.URL + "/oauth/authorize?" + url.Values{
		"client_id":             {"https://client.example/app"},
		"redirect_uri":          {"https://client.example/callback"},
		"response_type":         {"code"},
		"code_challenge":        {"challenge"},
		"code_challenge_method": {"plain"},
	}.Encode())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestOAuthMetadata(t *testing.T) {
	ts := newTestServer(t, "open")
	resp, err := ts.Client().Get(ts.URL + "/.well-known/oauth-authorization-server")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if m["issuer"] != "https://fish.example" {
		t.Errorf("issuer = %v", m["issuer"])
	}
	challenges, _ := m["code_challenge_methods_supported"].([]any)
	if len(challenges) != 1 || challenges[0] != "S256" {
		t.Errorf("code_challenge_methods_supported = %v, want [S256]", m["code_challenge_methods_supported"])
	}
}

func TestBearerHeaderAuthentication(t *testing.T) {
	ts := newTestServer(t, "open")
	name := uniqueName(t)
	_, signup := post(t, ts, "/api/signup", map[string]any{"username": name, "password": "hunter2"})
	token := signup["token"].(string)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/i", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
